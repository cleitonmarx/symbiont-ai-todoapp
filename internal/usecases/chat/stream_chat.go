package chat

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/metrics"
	"github.com/google/uuid"
)

const (
	// MAX_CHAT_HISTORY_MESSAGES is the maximum number of prior messages included in chat context.
	MAX_CHAT_HISTORY_MESSAGES = 5

	// MAX_REPEATED_ACTION_CALL_HIT is the limit for repeated action-call detections before aborting.
	MAX_REPEATED_ACTION_CALL_HIT = 5

	// CHAT_TEMPERATURE controls generation randomness for streamed chat turns.
	CHAT_TEMPERATURE = 0.2
	// CHAT_TOP_P controls nucleus sampling for streamed chat turns.
	CHAT_TOP_P       = 0.7

	// MAX_SKILLS_PROMPT_CHARS is the maximum size of injected skill prompt content.
	MAX_SKILLS_PROMPT_CHARS = 4000
	// MAX_RECOVERY_MESSAGES is the maximum number of recovery messages retained in loops.
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
	Execute(ctx context.Context, userMessage, model string, onEvent assistant.EventCallback, opts ...StreamChatOption) error
}

// StreamChatImpl is the implementation of the StreamChat use case
type StreamChatImpl struct {
	logger                  *log.Logger
	chatMessageRepo         assistant.ChatMessageRepository
	conversationSummaryRepo assistant.ConversationSummaryRepository
	conversationRepo        assistant.ConversationRepository
	uow                     transaction.UnitOfWork
	timeProvider            core.CurrentTimeProvider
	assistant               assistant.Assistant
	actionRegistry          assistant.ActionRegistry
	skillRegistry           assistant.SkillRegistry
	approvalDispatcher      assistant.ActionApprovalDispatcher
	embeddingModel          string
	maxActionCycles         int
}

// NewStreamChatImpl creates a new instance of StreamChatImpl
func NewStreamChatImpl(
	logger *log.Logger,
	chatMessageRepo assistant.ChatMessageRepository,
	conversationSummaryRepo assistant.ConversationSummaryRepository,
	conversationRepo assistant.ConversationRepository,
	timeProvider core.CurrentTimeProvider,
	assistant assistant.Assistant,
	actionRegistry assistant.ActionRegistry,
	assistantSkillRegistry assistant.SkillRegistry,
	approvalDispatcher assistant.ActionApprovalDispatcher,
	uow transaction.UnitOfWork,
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
func (sc StreamChatImpl) Execute(ctx context.Context, userMessage, model string, onEvent assistant.EventCallback, opts ...StreamChatOption) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	if strings.TrimSpace(userMessage) == "" {
		return core.NewValidationErr("message cannot be empty")
	}

	if model == "" {
		return core.NewValidationErr("model cannot be empty")
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

	messagesHistory = append(messagesHistory, assistant.Message{
		Role:    assistant.ChatRole_User,
		Content: userMessage,
	})

	skills := sc.skillRegistry.ListRelevant(spanCtx, assistant.SkillQueryContext{
		Messages:            messagesHistory,
		ConversationSummary: summaryContext,
	})
	selectedSkills := make([]assistant.SelectedSkill, 0, len(skills))
	relevantActions := make([]assistant.ActionDefinition, 0, len(skills))
	uniqueActionNames := make(map[string]struct{})
	for _, s := range skills {
		selectedSkills = append(selectedSkills, assistant.NewSelectedSkill(s))
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
		messagesHistory = append(messagesHistory, assistant.Message{
			Role:    assistant.ChatRole_System,
			Content: skillsPrompt,
		})
	}

	req := assistant.TurnRequest{
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

	state.userMsg = assistant.ChatMessage{
		ConversationID: conversation.ID,
		TurnID:         state.turnID,
		TurnSequence:   state.nextTurnSequence(),
		ChatRole:       assistant.ChatRole_User,
		Content:        userMessage,
		Model:          model,
		MessageState:   assistant.ChatMessageState_Completed,
	}

	for continueChatStreaming := true; continueChatStreaming; {
		continueChatStreaming = false
		var streamEventErr error

		err = sc.assistant.RunTurn(spanCtx, req, func(turnCtx context.Context, eventType assistant.EventType, data any) error {
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

	assistantMsg := assistant.ChatMessage{
		ID:               state.assistantMsgID,
		ConversationID:   state.conversation.ID,
		TurnID:           state.turnID,
		TurnSequence:     state.nextTurnSequence(),
		ChatRole:         assistant.ChatRole_Assistant,
		Content:          state.assistantMsgContent.String(),
		SelectedSkills:   state.selectedSkills,
		Model:            model,
		MessageState:     assistant.ChatMessageState_Completed,
		PromptTokens:     state.tokenUsage.PromptTokens,
		CompletionTokens: state.tokenUsage.CompletionTokens,
		TotalTokens:      state.tokenUsage.TotalTokens,
		CreatedAt:        sc.timeProvider.Now(),
	}
	assistantMsg.UpdatedAt = assistantMsg.CreatedAt

	if assistantMsg.Content == "" {
		assistantMsg.Content = "Sorry, I could not process your request. Please try again."
		if err := onEvent(ctx, assistant.EventType_MessageDelta,
			assistant.MessageDelta{
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

	metrics.RecordLLMTokensUsed(spanCtx, state.tokenUsage.PromptTokens, state.tokenUsage.CompletionTokens)

	if err := onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
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
func (sc StreamChatImpl) createOrRetrieveConversation(ctx context.Context, params *StreamChatParams, userMessage string) (assistant.Conversation, bool, error) {
	var (
		conversation        assistant.Conversation
		conversationCreated bool
	)

	if params.ConversationID == nil {
		title := assistant.GenerateAutoConversationTitle(userMessage)
		newConversation, err := sc.conversationRepo.CreateConversation(ctx, title, assistant.ConversationTitleSource_Auto)
		if err != nil {
			return assistant.Conversation{}, false, err
		}
		conversation = newConversation
		conversationCreated = true
	} else {
		c, found, err := sc.conversationRepo.GetConversation(ctx, *params.ConversationID)
		if err != nil {
			return assistant.Conversation{}, false, err
		}
		if !found {
			return assistant.Conversation{}, false, core.NewValidationErr("conversation not found")
		}
		conversation = c
	}
	return conversation, conversationCreated, nil
}
