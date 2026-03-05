package usecases

import (
	"context"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
)

// persistUserMessageIfNeeded persists the user message when streaming completed without an earlier meta event.
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

// persistFailureMessages stores the user message if needed and then records a failed assistant message.
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
		SelectedSkills:   state.selectedSkills,
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

// persistChatMessage stores one chat message and publishes the corresponding outbox event.
func (sc StreamChatImpl) persistChatMessage(ctx context.Context, message domain.ChatMessage, conversation domain.Conversation) error {
	return sc.uow.Execute(ctx, func(uowCtx context.Context, uow domain.UnitOfWork) error {
		if err := uow.ChatMessage().CreateChatMessages(uowCtx, []domain.ChatMessage{message}); err != nil {
			return err
		}

		if err := uow.Outbox().CreateChatEvent(uowCtx, domain.ChatMessageEvent{
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
		if err := uow.Conversation().UpdateConversation(uowCtx, conversation); err != nil {
			return err
		}

		return nil
	})
}

// streamChatExecutionState tracks mutable per-turn state while Execute is streaming.
type streamChatExecutionState struct {
	conversation             domain.Conversation
	conversationCreated      bool
	assistantMsgContent      strings.Builder
	assistantMsgID           uuid.UUID
	selectedSkills           []domain.AssistantSelectedSkill
	tokenUsage               domain.AssistantUsage
	turnID                   uuid.UUID
	turnSequence             int64
	userMsg                  domain.ChatMessage
	userMsgPersisted         bool
	userMsgPersistTried      bool
	runTurnRecoveryAttempted bool
	tracker                  *actionCycleTracker
}

// nextTurnSequence returns the current turn sequence value and increments it for the next message.
func (s *streamChatExecutionState) nextTurnSequence() int64 {
	current := s.turnSequence
	s.turnSequence++
	return current
}

// actionCycleTracker tracks action loop counts and repeated calls to prevent infinite tool loops.
type actionCycleTracker struct {
	maxActionCycles          int
	maxRepeatedActionCallHit int
	actionCycles             int
	lastActionCallSignature  string
	repeatActionCallCount    int
}

// newActionCycleTracker initializes a tracker with the configured loop limits.
func newActionCycleTracker(maxActionCycles, maxRepeatedActionCallHit int) *actionCycleTracker {
	return &actionCycleTracker{
		maxActionCycles:          maxActionCycles,
		maxRepeatedActionCallHit: maxRepeatedActionCallHit,
	}
}

// hasExceededMaxCycles increments the cycle count and reports whether the limit was exceeded.
func (t *actionCycleTracker) hasExceededMaxCycles() bool {
	t.actionCycles++
	return t.actionCycles > t.maxActionCycles
}

// hasExceededMaxActionCalls reports whether the same action signature repeated too many times in sequence.
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

// prepareRunTurnRecovery compacts the request and injects one recovery system message for a retry turn.
func (sc StreamChatImpl) prepareRunTurnRecovery(
	runTurnErr error,
	req *domain.AssistantTurnRequest,
	state *streamChatExecutionState,
) bool {
	if state.runTurnRecoveryAttempted {
		return false
	}

	state.runTurnRecoveryAttempted = true
	req.AvailableActions = nil
	req.Messages = compactToLastMessages(req.Messages, MAX_RECOVERY_MESSAGES)
	req.Messages = append(req.Messages, domain.AssistantMessage{
		Role: domain.ChatRole_System,
		Content: "The previous assistant turn failed due to an internal processing issue " +
			"(commonly tool execution failure or context size limit). " +
			"Internal error: " + truncateToFirstChars(strings.TrimSpace(runTurnErr.Error()), 400) + ". " +
			"Reply to the user with a short apology and explain that the request failed due to an internal error. " +
			"Suggest retrying with a smaller scope. Do not claim actions succeeded.",
	})

	return true
}
