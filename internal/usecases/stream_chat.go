package usecases

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

const (
	// Maximum number of chat history messages to include in the context
	MAX_CHAT_HISTORY_MESSAGES = 5

	// Maximum number of repeated tool call hits to prevent infinite loops
	MAX_REPEATED_TOOL_CALL_HIT = 5

	// Keep tool-calling deterministic to reduce malformed function arguments.
	CHAT_TEMPERATURE = 0.2
	CHAT_TOP_P       = 0.7
)

//go:embed prompts/chat.yml
var chatPrompt embed.FS

// StreamChatParams holds optional parameters for StreamChat execution.
type StreamChatParams struct {
	ConversationID *uuid.UUID
}

// StreamChatOption defines a functional option for configuring StreamChatParams.
type StreamChatOption func(*StreamChatParams)

func WithConversationID(conversationID uuid.UUID) StreamChatOption {
	return func(params *StreamChatParams) {
		params.ConversationID = &conversationID
	}
}

// StreamChat defines the interface for the StreamChat use case
type StreamChat interface {
	// Execute streams a chat response and persists the conversation
	Execute(ctx context.Context, userMessage, model string, onEvent domain.LLMStreamEventCallback, opts ...StreamChatOption) error
}

// StreamChatImpl is the implementation of the StreamChat use case
type StreamChatImpl struct {
	chatMessageRepo         domain.ChatMessageRepository
	conversationSummaryRepo domain.ConversationSummaryRepository
	conversationRepo        domain.ConversationRepository
	uow                     domain.UnitOfWork
	timeProvider            domain.CurrentTimeProvider
	llmClient               domain.LLMClient
	llmToolRegistry         domain.LLMToolRegistry
	llmEmbeddingModel       string
	maxToolCycles           int
}

// NewStreamChatImpl creates a new instance of StreamChatImpl
func NewStreamChatImpl(
	chatMessageRepo domain.ChatMessageRepository,
	conversationSummaryRepo domain.ConversationSummaryRepository,
	conversationRepo domain.ConversationRepository,
	timeProvider domain.CurrentTimeProvider,
	llmClient domain.LLMClient,
	llmToolRegistry domain.LLMToolRegistry,
	uow domain.UnitOfWork,
	llmEmbeddingModel string,
	maxToolCycles int,
) StreamChatImpl {
	return StreamChatImpl{
		chatMessageRepo:         chatMessageRepo,
		conversationSummaryRepo: conversationSummaryRepo,
		conversationRepo:        conversationRepo,
		uow:                     uow,
		timeProvider:            timeProvider,
		llmClient:               llmClient,
		llmToolRegistry:         llmToolRegistry,
		llmEmbeddingModel:       llmEmbeddingModel,
		maxToolCycles:           maxToolCycles,
	}
}

// Execute streams a chat response and persists the conversation
func (sc StreamChatImpl) Execute(ctx context.Context, userMessage, model string, onEvent domain.LLMStreamEventCallback, opts ...StreamChatOption) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	params := &StreamChatParams{}
	for _, opt := range opts {
		opt(params)
	}

	if strings.TrimSpace(userMessage) == "" {
		return domain.NewValidationErr("message cannot be empty")
	}

	if model == "" {
		return domain.NewValidationErr("model cannot be empty")
	}

	var (
		conversation        domain.Conversation
		conversationCreated bool
	)

	if params.ConversationID == nil {
		// Create a new conversation for this chat interaction
		title := domain.GenerateAutoConversationTitle(userMessage)
		newConversation, err := sc.conversationRepo.CreateConversation(spanCtx, title, domain.ConversationTitleSource_Auto)
		if telemetry.RecordErrorAndStatus(span, err) {
			return err
		}
		conversation = newConversation
		conversationCreated = true
	} else {
		c, found, err := sc.conversationRepo.GetConversation(spanCtx, *params.ConversationID)
		if telemetry.RecordErrorAndStatus(span, err) {
			return err
		}
		if !found {
			return domain.NewValidationErr("conversation not found")
		}
		conversation = c
	}

	// Fetch chat history and append user message
	messages, err := sc.fetchChatHistory(spanCtx, conversation.ID)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}
	messages = append(messages, domain.LLMChatMessage{
		Role:    domain.ChatRole_User,
		Content: userMessage,
	})

	req := domain.LLMChatRequest{
		Model:       model,
		Messages:    messages,
		Stream:      true,
		Temperature: common.Ptr(CHAT_TEMPERATURE),
		TopP:        common.Ptr(CHAT_TOP_P),
		Tools:       sc.llmToolRegistry.List(),
	}

	state := streamChatExecutionState{
		conversation:        conversation,
		conversationCreated: conversationCreated,
		turnID:              uuid.New(),
		tracker: newToolCycleTracker(
			sc.maxToolCycles,
			MAX_REPEATED_TOOL_CALL_HIT,
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

		err = sc.llmClient.ChatStream(spanCtx, req, func(eventType domain.LLMStreamEventType, data any) error {
			shouldContinue, eventErr := sc.handleStreamEvent(spanCtx, eventType, data, model, &req, &state, onEvent)
			if shouldContinue {
				continueChatStreaming = true
			}
			return eventErr
		})
		if telemetry.RecordErrorAndStatus(span, err) {
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
		Model:            model,
		MessageState:     domain.ChatMessageState_Completed,
		PromptTokens:     state.tokenUsage.PromptTokens,
		CompletionTokens: state.tokenUsage.CompletionTokens,
		TotalTokens:      state.tokenUsage.TotalTokens,
		CreatedAt:        sc.timeProvider.Now().UTC(),
	}
	assistantMsg.UpdatedAt = assistantMsg.CreatedAt

	// Append the final assistant message with the full content only if there is content
	if assistantMsg.Content == "" {
		assistantMsg.Content = "Sorry, I could not process your request. Please try again."
		if err := onEvent(domain.LLMStreamEventType_Delta,
			domain.LLMStreamEventDelta{
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

	// Send done event
	if err := onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
		AssistantMessageID: assistantMsg.ID.String(),
		CompletedAt:        sc.timeProvider.Now().UTC().Format(time.RFC3339),
		Usage:              state.tokenUsage,
	}); err != nil {
		return err
	}
	return nil
}

// streamChatExecutionState holds mutable state during a stream-chat execution.
type streamChatExecutionState struct {
	conversation        domain.Conversation
	conversationCreated bool
	assistantMsgContent strings.Builder
	assistantMsgID      uuid.UUID
	tokenUsage          domain.LLMUsage
	turnID              uuid.UUID
	turnSequence        int64
	userMsg             domain.ChatMessage
	userMsgPersisted    bool
	userMsgPersistTried bool
	tracker             *toolCycleTracker
}

// nextTurnSequence returns the current sequence value and advances the counter.
func (s *streamChatExecutionState) nextTurnSequence() int64 {
	current := s.turnSequence
	s.turnSequence++
	return current
}

// handleStreamEvent routes one stream event to the corresponding specialized handler.
func (sc StreamChatImpl) handleStreamEvent(
	ctx context.Context,
	eventType domain.LLMStreamEventType,
	data any,
	model string,
	req *domain.LLMChatRequest,
	state *streamChatExecutionState,
	onEvent domain.LLMStreamEventCallback,
) (bool, error) {
	switch eventType {
	case domain.LLMStreamEventType_Meta:
		return false, sc.handleMetaEvent(ctx, data, state, onEvent)
	case domain.LLMStreamEventType_ToolCall:
		return sc.handleToolCallEvent(ctx, data, model, req, state, onEvent)
	case domain.LLMStreamEventType_Delta:
		return false, sc.handleDeltaEvent(data, state, onEvent)
	case domain.LLMStreamEventType_Done:
		sc.handleDoneEvent(data, state)
		return false, nil
	default:
		return false, nil
	}
}

// handleMetaEvent persists the user message as soon as stream IDs are available.
func (sc StreamChatImpl) handleMetaEvent(
	ctx context.Context,
	data any,
	state *streamChatExecutionState,
	onEvent domain.LLMStreamEventCallback,
) error {
	// Capture IDs from the first meta event and persist the user message immediately.
	if state.assistantMsgID != uuid.Nil {
		return nil
	}

	meta := data.(domain.LLMStreamEventMeta)
	meta.ConversationID = state.conversation.ID
	meta.ConversationCreated = state.conversationCreated
	state.assistantMsgID = meta.AssistantMessageID
	state.userMsg.ID = meta.UserMessageID
	state.userMsg.CreatedAt = sc.timeProvider.Now().UTC()
	state.userMsg.UpdatedAt = state.userMsg.CreatedAt
	state.userMsgPersistTried = true
	if err := sc.persistChatMessage(ctx, state.userMsg, state.conversation); err != nil {
		return err
	}
	state.userMsgPersisted = true
	return onEvent(domain.LLMStreamEventType_Meta, meta)
}

// handleToolCallEvent persists assistant tool-call and tool-result messages, then updates request context.
func (sc StreamChatImpl) handleToolCallEvent(
	ctx context.Context,
	data any,
	model string,
	req *domain.LLMChatRequest,
	state *streamChatExecutionState,
	onEvent domain.LLMStreamEventCallback,
) (bool, error) {
	toolCall := data.(domain.LLMStreamEventToolCall)
	if state.tracker.hasExceededMaxCycles() || state.tracker.hasExceededMaxToolCalls(toolCall.Function, toolCall.Arguments) {
		return false, nil
	}

	assistantToolCallMsg := domain.ChatMessage{
		ID:             uuid.New(),
		ConversationID: state.conversation.ID,
		TurnID:         state.turnID,
		TurnSequence:   state.nextTurnSequence(),
		ChatRole:       domain.ChatRole_Assistant,
		ToolCalls:      []domain.LLMStreamEventToolCall{toolCall},
		Model:          model,
		MessageState:   domain.ChatMessageState_Completed,
		CreatedAt:      sc.timeProvider.Now().UTC(),
	}
	assistantToolCallMsg.UpdatedAt = assistantToolCallMsg.CreatedAt
	if err := sc.persistChatMessage(ctx, assistantToolCallMsg, state.conversation); err != nil {
		return false, err
	}

	toolCall.Text = sc.llmToolRegistry.StatusMessage(toolCall.Function)
	toolStarted := domain.LLMStreamEventToolCallStarted{
		ID:        toolCall.ID,
		Function:  toolCall.Function,
		Arguments: toolCall.Arguments,
		Text:      toolCall.Text,
	}
	if err := onEvent(domain.LLMStreamEventType_ToolStarted, toolStarted); err != nil {
		return false, err
	}

	toolMessage := sc.llmToolRegistry.Call(ctx, toolCall, req.Messages)
	toolSucceeded := toolMessage.IsToolCallSuccess()
	now := sc.timeProvider.Now().UTC()
	toolChatMsg := domain.ChatMessage{
		ID:             uuid.New(),
		ConversationID: state.conversation.ID,
		TurnID:         state.turnID,
		TurnSequence:   state.nextTurnSequence(),
		ChatRole:       domain.ChatRole_Tool,
		ToolCallID:     &toolCall.ID,
		Content:        toolMessage.Content,
		Model:          model,
		MessageState:   domain.ChatMessageState_Completed,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if !toolSucceeded {
		toolChatMsg.MessageState = domain.ChatMessageState_Failed
		toolChatMsg.ErrorMessage = &toolMessage.Content
	}

	if err := sc.persistChatMessage(ctx, toolChatMsg, state.conversation); err != nil {
		return false, err
	}

	toolCompleted := domain.LLMStreamEventToolCallCompleted{
		ID:            toolCall.ID,
		Function:      toolCall.Function,
		Success:       toolSucceeded,
		ShouldRefetch: toolSucceeded,
	}
	if !toolSucceeded {
		toolCompleted.Error = &toolMessage.Content
	}
	if err := onEvent(domain.LLMStreamEventType_ToolCompleted, toolCompleted); err != nil {
		return false, err
	}

	req.Messages = append(req.Messages,
		domain.LLMChatMessage{
			Role:      domain.ChatRole_Assistant,
			ToolCalls: []domain.LLMStreamEventToolCall{toolCall},
		},
		toolMessage,
	)

	return true, nil
}

// handleDeltaEvent appends assistant delta text and forwards the delta to the caller callback.
func (sc StreamChatImpl) handleDeltaEvent(
	data any,
	state *streamChatExecutionState,
	onEvent domain.LLMStreamEventCallback,
) error {
	delta := data.(domain.LLMStreamEventDelta)
	state.assistantMsgContent.WriteString(delta.Text)
	return onEvent(domain.LLMStreamEventType_Delta, data)
}

// handleDoneEvent accumulates usage from one stream completion event.
func (sc StreamChatImpl) handleDoneEvent(data any, state *streamChatExecutionState) {
	done := data.(domain.LLMStreamEventDone)
	state.tokenUsage.CompletionTokens += done.Usage.CompletionTokens
	state.tokenUsage.PromptTokens += done.Usage.PromptTokens
	state.tokenUsage.TotalTokens += done.Usage.TotalTokens
}

// persistUserMessageIfNeeded ensures the user message is persisted exactly once when no meta event was received.
func (sc StreamChatImpl) persistUserMessageIfNeeded(ctx context.Context, state *streamChatExecutionState) error {
	if state.userMsgPersisted || state.userMsgPersistTried {
		return nil
	}

	state.userMsg.ID = uuid.New()
	state.userMsg.ConversationID = state.conversation.ID
	state.userMsg.CreatedAt = sc.timeProvider.Now().UTC()
	state.userMsg.UpdatedAt = state.userMsg.CreatedAt
	state.userMsgPersistTried = true
	if err := sc.persistChatMessage(ctx, state.userMsg, state.conversation); err != nil {
		return err
	}
	state.userMsgPersisted = true
	return nil
}

// persistFailureMessages persists fallback user and assistant failure messages when streaming fails.
func (sc StreamChatImpl) persistFailureMessages(
	ctx context.Context,
	streamErr error,
	model string,
	state *streamChatExecutionState,
) error {
	if err := sc.persistUserMessageIfNeeded(ctx, state); err != nil {
		return err
	}

	if state.assistantMsgID == uuid.Nil {
		state.assistantMsgID = uuid.New()
	}

	failedAt := sc.timeProvider.Now().UTC()
	errorMessage := streamErr.Error()
	failedAssistantMsg := domain.ChatMessage{
		ID:               state.assistantMsgID,
		ConversationID:   state.conversation.ID,
		TurnID:           state.turnID,
		TurnSequence:     state.nextTurnSequence(),
		ChatRole:         domain.ChatRole_Assistant,
		Content:          "",
		Model:            model,
		MessageState:     domain.ChatMessageState_Failed,
		ErrorMessage:     &errorMessage,
		PromptTokens:     state.tokenUsage.PromptTokens,
		CompletionTokens: state.tokenUsage.CompletionTokens,
		TotalTokens:      state.tokenUsage.TotalTokens,
		CreatedAt:        failedAt,
		UpdatedAt:        failedAt,
	}
	return sc.persistChatMessage(ctx, failedAssistantMsg, state.conversation)
}

// persistChatMessage persists a chat message and emits a corresponding domain event for outbox processing.
func (sc StreamChatImpl) persistChatMessage(ctx context.Context, message domain.ChatMessage, conversation domain.Conversation) error {
	return sc.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		if err := uow.ChatMessage().CreateChatMessages(ctx, []domain.ChatMessage{message}); err != nil {
			return err
		}

		if err := uow.Outbox().CreateChatEvent(ctx, domain.ChatMessageEvent{
			Type:           domain.EventType_CHAT_MESSAGE_SENT,
			ChatRole:       message.ChatRole,
			ChatMessageID:  message.ID,
			ConversationID: message.ConversationID,
			CreatedAt:      message.CreatedAt,
		}); err != nil {
			return err
		}

		lastMessageAt := message.CreatedAt
		if conversation.LastMessageAt == nil || message.CreatedAt.After(*conversation.LastMessageAt) {
			conversation.LastMessageAt = &lastMessageAt
		}
		if message.CreatedAt.After(conversation.UpdatedAt) {
			conversation.UpdatedAt = message.CreatedAt
		}
		if err := uow.Conversation().UpdateConversation(ctx, conversation); err != nil {
			return err
		}

		return nil
	})
}

// buildSystemPrompt creates the base chat prompt and injects the latest conversation summary context.
func (sc StreamChatImpl) buildSystemPrompt(ctx context.Context, conversationID uuid.UUID) ([]domain.LLMChatMessage, error) {
	file, err := chatPrompt.Open("prompts/chat.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to open chat prompt: %w", err)
	}
	defer file.Close() //nolint:errcheck

	messages := []domain.LLMChatMessage{}
	err = yaml.NewDecoder(file).Decode(&messages)
	if err != nil {
		return nil, fmt.Errorf("failed to decode summary prompt: %w", err)
	}
	for i, msg := range messages {
		if msg.Role == domain.ChatRole_Developer || msg.Role == domain.ChatRole_System {
			messages[i].Content = fmt.Sprintf(
				msg.Content,
				sc.timeProvider.Now().Format(time.DateOnly),
				sc.timeProvider.Now().Unix(),
			)
		}
	}

	latestSummary, found, err := sc.conversationSummaryRepo.GetConversationSummary(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to load conversation summary: %w", err)
	}

	summaryText := "No conversation summary available."
	if found && strings.TrimSpace(latestSummary.CurrentStateSummary) != "" {
		summaryText = strings.TrimSpace(latestSummary.CurrentStateSummary)
	}
	messages = append(messages, domain.LLMChatMessage{
		Role: domain.ChatRole_Developer,
		Content: fmt.Sprintf(
			"Conversation summary context:\n%s\n\nUse this as compact memory, but prioritize explicit user instructions in this turn.",
			summaryText,
		),
	})

	return messages, nil
}

// fetchChatHistory retrieves the chat history excluding old system messages
func (sc StreamChatImpl) fetchChatHistory(ctx context.Context, conversationID uuid.UUID) ([]domain.LLMChatMessage, error) {
	// Build system prompt with todo context
	systemPrompt, err := sc.buildSystemPrompt(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// Load prior conversation to preserve context
	history, _, err := sc.chatMessageRepo.ListChatMessages(ctx, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES)
	if err != nil {
		return nil, err
	}

	// Build chat request: system + history (excluding old system messages) + current user turn
	messages := make([]domain.LLMChatMessage, 0, len(systemPrompt)+len(history)+1)
	messages = append(messages, systemPrompt...)

	//Remove orfaned tool messages from history
	// If the first message in history is a tool message, remove it
	if len(history) > 0 {
		if history[0].ChatRole == domain.ChatRole_Tool {
			history = history[1:]
		}
	}

	// Append prior conversation history, skipping previous system messages
	for _, msg := range history {
		if msg.ChatRole != domain.ChatRole_System {
			messages = append(messages, domain.LLMChatMessage{
				Role:       msg.ChatRole,
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
				ToolCalls:  msg.ToolCalls,
			})
		}
	}
	return messages, nil
}

// toolCycleTracker helps track repeated tool calls to prevent infinite loops
type toolCycleTracker struct {
	maxToolCycles          int
	maxRepeatedToolCallHit int
	toolCycles             int
	lastToolCallSignature  string
	repeatToolCallCount    int
}

// newToolCycleTracker creates a new toolCycleTracker
func newToolCycleTracker(maxToolCycles, maxRepeatedToolCallHit int) *toolCycleTracker {
	return &toolCycleTracker{
		maxToolCycles:          maxToolCycles,
		maxRepeatedToolCallHit: maxRepeatedToolCallHit,
	}
}

// hasExceededMaxCycles checks if the maximum number of tool cycles has been exceeded
func (t *toolCycleTracker) hasExceededMaxCycles() bool {
	t.toolCycles++
	return t.toolCycles > t.maxToolCycles
}

// hasExceededMaxToolCalls checks if the same tool call has been repeated too many times
func (t *toolCycleTracker) hasExceededMaxToolCalls(functionName, arguments string) bool {
	signature := functionName + ":" + arguments
	if signature == t.lastToolCallSignature {
		t.repeatToolCallCount++
		return t.repeatToolCallCount >= t.maxRepeatedToolCallHit
	}
	t.lastToolCallSignature = signature
	t.repeatToolCallCount = 0
	return false
}

// InitStreamChat is the initializer for the StreamChat use case
type InitStreamChat struct {
	ChatMessageRepo         domain.ChatMessageRepository         `resolve:""`
	ConversationSummaryRepo domain.ConversationSummaryRepository `resolve:""`
	ConversationRepo        domain.ConversationRepository        `resolve:""`
	Uow                     domain.UnitOfWork                    `resolve:""`
	TimeProvider            domain.CurrentTimeProvider           `resolve:""`
	LLMToolRegistry         domain.LLMToolRegistry               `resolve:""`
	LLMClient               domain.LLMClient                     `resolve:""`
	EmbeddingModel          string                               `config:"LLM_EMBEDDING_MODEL"`
	// Maximum number of tool cycles to prevent infinite loops
	// It restricts how many times the LLM can invoke tools in a single chat session
	MaxToolCycles int `config:"LLM_MAX_TOOL_CYCLES" default:"50"`
}

// Initialize registers the StreamChat use case in the dependency container
func (i InitStreamChat) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[StreamChat](NewStreamChatImpl(
		i.ChatMessageRepo,
		i.ConversationSummaryRepo,
		i.ConversationRepo,
		i.TimeProvider,
		i.LLMClient,
		i.LLMToolRegistry,
		i.Uow,
		i.EmbeddingModel,
		i.MaxToolCycles,
	))
	return ctx, nil
}
