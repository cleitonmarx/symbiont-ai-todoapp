package usecases

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"
	"unicode"

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

	// Maximum number of repeated action call hits to prevent infinite loops
	MAX_REPEATED_ACTION_CALL_HIT = 5

	// Keep action-calling deterministic to reduce malformed function arguments.
	CHAT_TEMPERATURE = 0.2
	CHAT_TOP_P       = 0.7

	MAX_TOOL_SELECTION_CHARS = 400
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
	Execute(ctx context.Context, userMessage, model string, onEvent domain.AssistantEventCallback, opts ...StreamChatOption) error
}

// StreamChatImpl is the implementation of the StreamChat use case
type StreamChatImpl struct {
	chatMessageRepo         domain.ChatMessageRepository
	conversationSummaryRepo domain.ConversationSummaryRepository
	conversationRepo        domain.ConversationRepository
	uow                     domain.UnitOfWork
	timeProvider            domain.CurrentTimeProvider
	assistant               domain.Assistant
	actionRegistry          domain.AssistantActionRegistry
	embeddingModel          string
	maxActionCycles         int
}

// NewStreamChatImpl creates a new instance of StreamChatImpl
func NewStreamChatImpl(
	chatMessageRepo domain.ChatMessageRepository,
	conversationSummaryRepo domain.ConversationSummaryRepository,
	conversationRepo domain.ConversationRepository,
	timeProvider domain.CurrentTimeProvider,
	assistant domain.Assistant,
	actionRegistry domain.AssistantActionRegistry,
	uow domain.UnitOfWork,
	embeddingModel string,
	maxActionCycles int,
) StreamChatImpl {
	return StreamChatImpl{
		chatMessageRepo:         chatMessageRepo,
		conversationSummaryRepo: conversationSummaryRepo,
		conversationRepo:        conversationRepo,
		uow:                     uow,
		timeProvider:            timeProvider,
		assistant:               assistant,
		actionRegistry:          actionRegistry,
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

	messages, err := sc.fetchChatHistory(spanCtx, conversation.ID)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	messages = append(messages, domain.AssistantMessage{
		Role:    domain.ChatRole_User,
		Content: userMessage,
	})

	req := domain.AssistantTurnRequest{
		Model:       model,
		Messages:    messages,
		Stream:      true,
		Temperature: common.Ptr(CHAT_TEMPERATURE),
		TopP:        common.Ptr(CHAT_TOP_P),
		AvailableActions: sc.actionRegistry.ListRelevant(
			spanCtx,
			buildActionSelectionText(messages),
		),
	}

	state := streamChatExecutionState{
		conversation:        conversation,
		conversationCreated: conversationCreated,
		turnID:              uuid.New(),
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

		err = sc.assistant.RunTurn(spanCtx, req, func(eventType domain.AssistantEventType, data any) error {
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
		CreatedAt:        sc.timeProvider.Now(),
	}
	assistantMsg.UpdatedAt = assistantMsg.CreatedAt

	// Append the final assistant message with the full content only if there is content
	if assistantMsg.Content == "" {
		assistantMsg.Content = "Sorry, I could not process your request. Please try again."
		if err := onEvent(domain.AssistantEventType_MessageDelta,
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

	// Send done event
	if err := onEvent(domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
		AssistantMessageID: assistantMsg.ID.String(),
		CompletedAt:        sc.timeProvider.Now().Format(time.RFC3339),
		Usage:              state.tokenUsage,
	}); err != nil {
		return err
	}
	return nil
}

// createOrRetrieveConversation either creates a new conversation or
// retrieves an existing one based on the provided parameters.
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

// buildActionSelectionText constructs the text used for selecting relevant actions
// based on the current user input and recent conversation history.
func buildActionSelectionText(messages []domain.AssistantMessage) string {
	if len(messages) == 0 {
		return ""
	}

	lastMessage := messages[len(messages)-1]
	if lastMessage.Role != domain.ChatRole_User {
		return ""
	}

	currentInput := strings.TrimSpace(lastMessage.Content)
	if currentInput == "" {
		return ""
	}

	selectionText := currentInput
	if isAmbiguousActionSelectionInput(currentInput) && len(messages) > 1 {
		if previousInput, ok := previousUserInput(messages[:len(messages)-1]); ok {
			selectionText = previousInput + "\n" + currentInput
		}
	}

	return truncateToLastChars(selectionText, MAX_TOOL_SELECTION_CHARS)
}

// isAmbiguousActionSelectionInput checks if the user input contains phrases
// or words that may ambiguously refer to previous actions or messages,
// which can help determine if we should include previous user
// input for better action relevance.
func isAmbiguousActionSelectionInput(userInput string) bool {
	lowered := strings.ToLower(strings.TrimSpace(userInput))
	if lowered == "" {
		return false
	}

	ambiguousPhrases := []string{
		"same as before",
		"as before",
		"do it",
		"do that",
		"that one",
		"this one",
		"same thing",
	}
	for _, phrase := range ambiguousPhrases {
		if strings.Contains(lowered, phrase) {
			return true
		}
	}

	ambiguousWords := map[string]struct{}{
		"it": {}, "that": {}, "this": {}, "them": {}, "same": {}, "also": {}, "again": {}, "too": {}, "there": {},
	}
	words := strings.FieldsFunc(lowered, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	for _, word := range words {
		if _, ok := ambiguousWords[word]; ok {
			return true
		}
	}
	return false
}

// previousUserInput scans the messages in reverse to find the most recent non-empty user message.
func previousUserInput(messages []domain.AssistantMessage) (string, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != domain.ChatRole_User {
			continue
		}

		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}

		return content, true
	}

	return "", false
}

// truncateToLastChars truncates the input string to the last maxChars characters,
// ensuring it does not cut off in the middle of a rune.
func truncateToLastChars(input string, maxChars int) string {
	trimmed := strings.TrimSpace(input)
	if maxChars <= 0 {
		return ""
	}

	runes := []rune(trimmed)
	if len(runes) <= maxChars {
		return trimmed
	}

	return string(runes[len(runes)-maxChars:])
}

// streamChatExecutionState holds mutable state during a stream-chat execution.
type streamChatExecutionState struct {
	conversation        domain.Conversation
	conversationCreated bool
	assistantMsgContent strings.Builder
	assistantMsgID      uuid.UUID
	tokenUsage          domain.AssistantUsage
	turnID              uuid.UUID
	turnSequence        int64
	userMsg             domain.ChatMessage
	userMsgPersisted    bool
	userMsgPersistTried bool
	tracker             *actionCycleTracker
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
	eventType domain.AssistantEventType,
	data any,
	model string,
	req *domain.AssistantTurnRequest,
	state *streamChatExecutionState,
	onEvent domain.AssistantEventCallback,
) (bool, error) {
	switch eventType {
	case domain.AssistantEventType_TurnStarted:
		return false, sc.handleMetaEvent(ctx, data, state, onEvent)
	case domain.AssistantEventType_ActionRequested:
		return sc.handleActionCallEvent(ctx, data, model, req, state, onEvent)
	case domain.AssistantEventType_MessageDelta:
		return false, sc.handleDeltaEvent(data, state, onEvent)
	case domain.AssistantEventType_TurnCompleted:
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
	onEvent domain.AssistantEventCallback,
) error {
	// Capture IDs from the first meta event and persist the user message immediately.
	if state.assistantMsgID != uuid.Nil {
		return nil
	}

	meta := data.(domain.AssistantTurnStarted)
	meta.ConversationID = state.conversation.ID
	meta.ConversationCreated = state.conversationCreated
	state.assistantMsgID = meta.AssistantMessageID
	state.userMsg.ID = meta.UserMessageID
	state.userMsg.CreatedAt = sc.timeProvider.Now()
	state.userMsg.UpdatedAt = state.userMsg.CreatedAt
	state.userMsgPersistTried = true
	if err := sc.persistChatMessage(ctx, state.userMsg, state.conversation); err != nil {
		return err
	}
	state.userMsgPersisted = true
	return onEvent(domain.AssistantEventType_TurnStarted, meta)
}

// handleActionCallEvent persists assistant action-call and action-result messages, then updates request context.
func (sc StreamChatImpl) handleActionCallEvent(
	ctx context.Context,
	data any,
	model string,
	req *domain.AssistantTurnRequest,
	state *streamChatExecutionState,
	onEvent domain.AssistantEventCallback,
) (bool, error) {
	actionCall := data.(domain.AssistantActionCall)
	if state.tracker.hasExceededMaxCycles() || state.tracker.hasExceededMaxActionCalls(actionCall.Name, actionCall.Input) {
		return false, nil
	}

	assistantActionCallMsg := domain.ChatMessage{
		ID:             uuid.New(),
		ConversationID: state.conversation.ID,
		TurnID:         state.turnID,
		TurnSequence:   state.nextTurnSequence(),
		ChatRole:       domain.ChatRole_Assistant,
		ActionCalls:    []domain.AssistantActionCall{actionCall},
		Model:          model,
		MessageState:   domain.ChatMessageState_Completed,
		CreatedAt:      sc.timeProvider.Now(),
	}
	assistantActionCallMsg.UpdatedAt = assistantActionCallMsg.CreatedAt
	if err := sc.persistChatMessage(ctx, assistantActionCallMsg, state.conversation); err != nil {
		return false, err
	}

	actionCall.Text = sc.actionRegistry.StatusMessage(actionCall.Name)
	if err := onEvent(domain.AssistantEventType_ActionStarted, actionCall); err != nil {
		return false, err
	}

	actionMessage := sc.actionRegistry.Execute(ctx, domain.AssistantActionCall{
		ID:    actionCall.ID,
		Name:  actionCall.Name,
		Input: actionCall.Input,
		Text:  actionCall.Text,
	}, req.Messages)
	actionSucceeded := actionMessage.IsActionCallSuccess()
	now := sc.timeProvider.Now()
	actionChatMsg := domain.ChatMessage{
		ID:             uuid.New(),
		ConversationID: state.conversation.ID,
		TurnID:         state.turnID,
		TurnSequence:   state.nextTurnSequence(),
		ChatRole:       domain.ChatRole_Tool,
		ActionCallID:   &actionCall.ID,
		Content:        actionMessage.Content,
		Model:          model,
		MessageState:   domain.ChatMessageState_Completed,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if !actionSucceeded {
		actionChatMsg.MessageState = domain.ChatMessageState_Failed
		actionChatMsg.ErrorMessage = &actionMessage.Content
	}

	if err := sc.persistChatMessage(ctx, actionChatMsg, state.conversation); err != nil {
		return false, err
	}

	actionCompleted := domain.AssistantActionCompleted{
		ID:            actionCall.ID,
		Name:          actionCall.Name,
		Success:       actionSucceeded,
		ShouldRefetch: actionSucceeded,
	}
	if !actionSucceeded {
		actionCompleted.Error = &actionMessage.Content
	}
	if err := onEvent(domain.AssistantEventType_ActionCompleted, actionCompleted); err != nil {
		return false, err
	}

	req.Messages = append(req.Messages,
		domain.AssistantMessage{
			Role:        domain.ChatRole_Assistant,
			ActionCalls: []domain.AssistantActionCall{actionCall},
		},
		domain.AssistantMessage{
			Role:         actionMessage.Role,
			Content:      actionMessage.Content,
			ActionCallID: actionMessage.ActionCallID,
			ActionCalls:  actionMessage.ActionCalls,
		},
	)

	return true, nil
}

// handleDeltaEvent appends assistant delta text and forwards the delta to the caller callback.
func (sc StreamChatImpl) handleDeltaEvent(
	data any,
	state *streamChatExecutionState,
	onEvent domain.AssistantEventCallback,
) error {
	delta := data.(domain.AssistantMessageDelta)
	state.assistantMsgContent.WriteString(delta.Text)
	return onEvent(domain.AssistantEventType_MessageDelta, data)
}

// handleDoneEvent accumulates usage from one stream completion event.
func (sc StreamChatImpl) handleDoneEvent(data any, state *streamChatExecutionState) {
	done := data.(domain.AssistantTurnCompleted)
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
	state.userMsg.CreatedAt = sc.timeProvider.Now()
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

	failedAt := sc.timeProvider.Now()
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
func (sc StreamChatImpl) buildSystemPrompt(ctx context.Context, conversationID uuid.UUID) ([]domain.AssistantMessage, error) {
	file, err := chatPrompt.Open("prompts/chat.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to open chat prompt: %w", err)
	}
	defer file.Close() //nolint:errcheck

	messages := []domain.AssistantMessage{}
	err = yaml.NewDecoder(file).Decode(&messages)
	if err != nil {
		return nil, fmt.Errorf("failed to decode summary prompt: %w", err)
	}
	for i, msg := range messages {
		if msg.Role == domain.ChatRole_Developer || msg.Role == domain.ChatRole_System {
			now := sc.timeProvider.Now()
			messages[i].Content = fmt.Sprintf(
				msg.Content,
				now.Format(time.DateOnly),
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
	messages = append(messages, domain.AssistantMessage{
		Role: domain.ChatRole_Developer,
		Content: fmt.Sprintf(
			"Conversation summary context:\n%s\n\nUse this as compact memory, but prioritize explicit user instructions in this turn.",
			summaryText,
		),
	})

	return messages, nil
}

// fetchChatHistory retrieves the chat history excluding old system messages
func (sc StreamChatImpl) fetchChatHistory(ctx context.Context, conversationID uuid.UUID) ([]domain.AssistantMessage, error) {
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
	messages := make([]domain.AssistantMessage, 0, len(systemPrompt)+len(history)+1)
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
			messages = append(messages, domain.AssistantMessage{
				Role:         msg.ChatRole,
				Content:      msg.Content,
				ActionCallID: msg.ActionCallID,
				ActionCalls:  msg.ActionCalls,
			})
		}
	}
	return messages, nil
}

// actionCycleTracker helps track repeated action calls to prevent infinite loops
type actionCycleTracker struct {
	maxActionCycles          int
	maxRepeatedActionCallHit int
	actionCycles             int
	lastActionCallSignature  string
	repeatActionCallCount    int
}

// newActionCycleTracker creates a new actionCycleTracker
func newActionCycleTracker(maxActionCycles, maxRepeatedActionCallHit int) *actionCycleTracker {
	return &actionCycleTracker{
		maxActionCycles:          maxActionCycles,
		maxRepeatedActionCallHit: maxRepeatedActionCallHit,
	}
}

// hasExceededMaxCycles checks if the maximum number of action cycles has been exceeded
func (t *actionCycleTracker) hasExceededMaxCycles() bool {
	t.actionCycles++
	return t.actionCycles > t.maxActionCycles
}

// hasExceededMaxActionCalls checks if the same action call has been repeated too many times
func (t *actionCycleTracker) hasExceededMaxActionCalls(functionName, arguments string) bool {
	signature := functionName + ":" + arguments
	if signature == t.lastActionCallSignature {
		t.repeatActionCallCount++
		return t.repeatActionCallCount >= t.maxRepeatedActionCallHit
	}
	t.lastActionCallSignature = signature
	t.repeatActionCallCount = 0
	return false
}

// InitStreamChat is the initializer for the StreamChat use case
type InitStreamChat struct {
	ChatMessageRepo         domain.ChatMessageRepository         `resolve:""`
	ConversationSummaryRepo domain.ConversationSummaryRepository `resolve:""`
	ConversationRepo        domain.ConversationRepository        `resolve:""`
	Uow                     domain.UnitOfWork                    `resolve:""`
	TimeProvider            domain.CurrentTimeProvider           `resolve:""`
	AssistantActionRegistry domain.AssistantActionRegistry       `resolve:""`
	Assistant               domain.Assistant                     `resolve:""`
	EmbeddingModel          string                               `config:"LLM_EMBEDDING_MODEL"`
	// Maximum number of action cycles to prevent infinite loops
	// It restricts how many times the Assistant can invoke actions in a single chat session
	MaxActionCycles int `config:"LLM_MAX_ACTION_CYCLES" default:"50"`
}

// Initialize registers the StreamChat use case in the dependency container
func (i InitStreamChat) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[StreamChat](NewStreamChatImpl(
		i.ChatMessageRepo,
		i.ConversationSummaryRepo,
		i.ConversationRepo,
		i.TimeProvider,
		i.Assistant,
		i.AssistantActionRegistry,
		i.Uow,
		i.EmbeddingModel,
		i.MaxActionCycles,
	))
	return ctx, nil
}
