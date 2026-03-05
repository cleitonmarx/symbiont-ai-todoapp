package usecases

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
)

const (
	// Maximum number of chat history messages to include in the context
	MAX_CHAT_HISTORY_MESSAGES = 5

	// Maximum number of repeated action call hits to prevent infinite loops
	MAX_REPEATED_ACTION_CALL_HIT = 5

	// Keep action-calling deterministic to reduce malformed function arguments.
	CHAT_TEMPERATURE = 0.2
	CHAT_TOP_P       = 0.7

	MAX_SKILLS_PROMPT_CHARS = 4000
	MAX_RECOVERY_MESSAGES   = 8
)

// StreamChatParams holds optional parameters for StreamChat execution.
type StreamChatParams struct {
	ConversationID *uuid.UUID
}

// StreamChatOption defines a functional option for configuring StreamChatParams.
type StreamChatOption func(*StreamChatParams)

// WithConversationID binds Execute to an existing conversation instead of creating a new one.
func WithConversationID(conversationID uuid.UUID) StreamChatOption {
	return func(params *StreamChatParams) {
		params.ConversationID = &conversationID
	}
}

// StreamChat defines the interface for the StreamChat use case
type StreamChat interface {
	// Execute streams a chat response and persists the conversation
	Execute(ctx context.Context, userMessage, model string, onEvent domain.AssistantEventCallback, opts ...StreamChatOption) error
}

// StreamChatImpl is the implementation of the StreamChat use case
type StreamChatImpl struct {
	logger                  *log.Logger
	chatMessageRepo         domain.ChatMessageRepository
	conversationSummaryRepo domain.ConversationSummaryRepository
	conversationRepo        domain.ConversationRepository
	uow                     domain.UnitOfWork
	timeProvider            domain.CurrentTimeProvider
	assistant               domain.Assistant
	actionRegistry          domain.AssistantActionRegistry
	skillRegistry           domain.AssistantSkillRegistry
	approvalDispatcher      domain.AssistantActionApprovalDispatcher
	embeddingModel          string
	maxActionCycles         int
}

// NewStreamChatImpl creates a new instance of StreamChatImpl
func NewStreamChatImpl(
	logger *log.Logger,
	chatMessageRepo domain.ChatMessageRepository,
	conversationSummaryRepo domain.ConversationSummaryRepository,
	conversationRepo domain.ConversationRepository,
	timeProvider domain.CurrentTimeProvider,
	assistant domain.Assistant,
	actionRegistry domain.AssistantActionRegistry,
	assistantSkillRegistry domain.AssistantSkillRegistry,
	approvalDispatcher domain.AssistantActionApprovalDispatcher,
	uow domain.UnitOfWork,
	embeddingModel string,
	maxActionCycles int,
) StreamChatImpl {
	return StreamChatImpl{
		logger:                  logger,
		chatMessageRepo:         chatMessageRepo,
		conversationSummaryRepo: conversationSummaryRepo,
		conversationRepo:        conversationRepo,
		uow:                     uow,
		timeProvider:            timeProvider,
		assistant:               assistant,
		actionRegistry:          actionRegistry,
		skillRegistry:           assistantSkillRegistry,
		approvalDispatcher:      approvalDispatcher,
		embeddingModel:          embeddingModel,
		maxActionCycles:         maxActionCycles,
	}
}

// Execute streams a chat response and persists the conversation
func (sc StreamChatImpl) Execute(ctx context.Context, userMessage, model string, onEvent domain.AssistantEventCallback, opts ...StreamChatOption) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	if strings.TrimSpace(userMessage) == "" {
		return domain.NewValidationErr("message cannot be empty")
	}

	if model == "" {
		return domain.NewValidationErr("model cannot be empty")
	}

	params := &StreamChatParams{}
	for _, opt := range opts {
		opt(params)
	}

	conversation, conversationCreated, err := sc.createOrRetrieveConversation(spanCtx, params, userMessage)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	messagesHistory, summaryContext, err := sc.fetchChatHistory(spanCtx, conversation.ID)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	messagesHistory = append(messagesHistory, domain.AssistantMessage{
		Role:    domain.ChatRole_User,
		Content: userMessage,
	})

	skills := sc.skillRegistry.ListRelevant(spanCtx, domain.AssistantSkillQueryContext{
		Messages:            messagesHistory,
		ConversationSummary: summaryContext,
	})
	selectedSkills := make([]domain.AssistantSelectedSkill, 0, len(skills))
	relevantActions := make([]domain.AssistantActionDefinition, 0, len(skills))
	uniqueActionNames := make(map[string]struct{})
	for _, s := range skills {
		selectedSkills = append(selectedSkills, domain.NewAssistantSelectedSkill(s))
		sc.logger.Printf("StreamChat: skill '%s' is relevant for the current conversation context", s.Name)
		for _, tool := range s.Tools {
			if action, ok := sc.actionRegistry.GetDefinition(tool); ok {
				if _, exists := uniqueActionNames[action.Name]; !exists {
					relevantActions = append(relevantActions, action)
					uniqueActionNames[action.Name] = struct{}{}
				}
			}
		}
	}

	if skillsPrompt := buildSkillsPrompt(skills); skillsPrompt != "" {
		messagesHistory = append(messagesHistory, domain.AssistantMessage{
			Role:    domain.ChatRole_System,
			Content: skillsPrompt,
		})
	}

	req := domain.AssistantTurnRequest{
		Model:            model,
		Messages:         messagesHistory,
		Stream:           true,
		Temperature:      common.Ptr(CHAT_TEMPERATURE),
		TopP:             common.Ptr(CHAT_TOP_P),
		AvailableActions: relevantActions,
	}

	state := streamChatExecutionState{
		conversation:        conversation,
		conversationCreated: conversationCreated,
		turnID:              uuid.New(),
		selectedSkills:      selectedSkills,
		tracker: newActionCycleTracker(
			sc.maxActionCycles,
			MAX_REPEATED_ACTION_CALL_HIT,
		),
	}

	state.userMsg = domain.ChatMessage{
		ConversationID: conversation.ID,
		TurnID:         state.turnID,
		TurnSequence:   state.nextTurnSequence(),
		ChatRole:       domain.ChatRole_User,
		Content:        userMessage,
		Model:          model,
		MessageState:   domain.ChatMessageState_Completed,
	}

	for continueChatStreaming := true; continueChatStreaming; {
		continueChatStreaming = false
		var streamEventErr error

		err = sc.assistant.RunTurn(spanCtx, req, func(turnCtx context.Context, eventType domain.AssistantEventType, data any) error {
			shouldContinue, eventErr := sc.handleStreamEvent(turnCtx, eventType, data, model, &req, &state, onEvent)
			if shouldContinue {
				continueChatStreaming = true
			}
			if eventErr != nil && streamEventErr == nil {
				streamEventErr = eventErr
			}
			return eventErr
		})
		if err != nil {
			if streamEventErr == nil && sc.prepareRunTurnRecovery(err, &req, &state) {
				continueChatStreaming = true
				sc.logger.Printf("StreamChat: encountered error during RunTurn, but prepared recovery. err=%v", err)
				continue
			}

			telemetry.RecordErrorAndStatus(span, err)
			if persistErr := sc.persistFailureMessages(spanCtx, err, model, &state); persistErr != nil {
				return persistErr
			}
			return err
		}
	}

	if err := sc.persistUserMessageIfNeeded(spanCtx, &state); telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	if state.assistantMsgID == uuid.Nil {
		state.assistantMsgID = uuid.New()
	}

	assistantMsg := domain.ChatMessage{
		ID:               state.assistantMsgID,
		ConversationID:   state.conversation.ID,
		TurnID:           state.turnID,
		TurnSequence:     state.nextTurnSequence(),
		ChatRole:         domain.ChatRole_Assistant,
		Content:          state.assistantMsgContent.String(),
		SelectedSkills:   state.selectedSkills,
		Model:            model,
		MessageState:     domain.ChatMessageState_Completed,
		PromptTokens:     state.tokenUsage.PromptTokens,
		CompletionTokens: state.tokenUsage.CompletionTokens,
		TotalTokens:      state.tokenUsage.TotalTokens,
		CreatedAt:        sc.timeProvider.Now(),
	}
	assistantMsg.UpdatedAt = assistantMsg.CreatedAt

	if assistantMsg.Content == "" {
		assistantMsg.Content = "Sorry, I could not process your request. Please try again."
		if err := onEvent(ctx, domain.AssistantEventType_MessageDelta,
			domain.AssistantMessageDelta{
				Text: assistantMsg.Content + "\n",
			},
		); err != nil {
			return err
		}
	}

	err = sc.persistChatMessage(spanCtx, assistantMsg, state.conversation)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	RecordLLMTokensUsed(spanCtx, state.tokenUsage.PromptTokens, state.tokenUsage.CompletionTokens)

	if err := onEvent(ctx, domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
		AssistantMessageID: assistantMsg.ID.String(),
		CompletedAt:        sc.timeProvider.Now().Format(time.RFC3339),
		Usage:              state.tokenUsage,
	}); err != nil {
		return err
	}
	return nil
}

// createOrRetrieveConversation resolves the target conversation for the turn,
// creating a new one when no conversation option was provided.
func (sc StreamChatImpl) createOrRetrieveConversation(ctx context.Context, params *StreamChatParams, userMessage string) (domain.Conversation, bool, error) {
	var (
		conversation        domain.Conversation
		conversationCreated bool
	)

	if params.ConversationID == nil {
		title := domain.GenerateAutoConversationTitle(userMessage)
		newConversation, err := sc.conversationRepo.CreateConversation(ctx, title, domain.ConversationTitleSource_Auto)
		if err != nil {
			return domain.Conversation{}, false, err
		}
		conversation = newConversation
		conversationCreated = true
	} else {
		c, found, err := sc.conversationRepo.GetConversation(ctx, *params.ConversationID)
		if err != nil {
			return domain.Conversation{}, false, err
		}
		if !found {
			return domain.Conversation{}, false, domain.NewValidationErr("conversation not found")
		}
		conversation = c
	}
	return conversation, conversationCreated, nil
}

// InitStreamChat is the initializer for the StreamChat use case
type InitStreamChat struct {
	Logger                  *log.Logger                              `resolve:""`
	ChatMessageRepo         domain.ChatMessageRepository             `resolve:""`
	ConversationSummaryRepo domain.ConversationSummaryRepository     `resolve:""`
	ConversationRepo        domain.ConversationRepository            `resolve:""`
	Uow                     domain.UnitOfWork                        `resolve:""`
	TimeProvider            domain.CurrentTimeProvider               `resolve:""`
	AssistantActionRegistry domain.AssistantActionRegistry           `resolve:""`
	AssistantSkillRegistry  domain.AssistantSkillRegistry            `resolve:""`
	ApprovalDispatcher      domain.AssistantActionApprovalDispatcher `resolve:""`
	Assistant               domain.Assistant                         `resolve:""`
	EmbeddingModel          string                                   `config:"LLM_EMBEDDING_MODEL"`

	// Maximum number of action cycles to prevent infinite loops
	// It restricts how many times the Assistant can invoke actions in a single chat session
	MaxActionCycles int `config:"LLM_MAX_ACTION_CYCLES" default:"50"`
}

// Initialize registers the StreamChat use case in the dependency container
func (i InitStreamChat) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[StreamChat](NewStreamChatImpl(
		i.Logger,
		i.ChatMessageRepo,
		i.ConversationSummaryRepo,
		i.ConversationRepo,
		i.TimeProvider,
		i.Assistant,
		i.AssistantActionRegistry,
		i.AssistantSkillRegistry,
		i.ApprovalDispatcher,
		i.Uow,
		i.EmbeddingModel,
		i.MaxActionCycles,
	))
	return ctx, nil
}
