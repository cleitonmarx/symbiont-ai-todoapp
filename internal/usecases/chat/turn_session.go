package chat

import (
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/google/uuid"
)

// TurnSession encapsulates the mutable state and operations for a single assistant turn, including recovery and action loop tracking.
type TurnSession interface {
	// Conversation returns the target conversation for the turn.
	Conversation() assistant.Conversation
	// ConversationCreated reports whether the conversation was created for this turn.
	ConversationCreated() bool
	// Request returns a snapshot of the current turn request.
	Request() assistant.TurnRequest
	// UpdateRequest exposes the current request for in-place mutation.
	UpdateRequest(apply func(request *assistant.TurnRequest))
	// Model returns the current request model name.
	Model() string
	// AssistantMessageID returns the assigned assistant message identifier, if any.
	AssistantMessageID() uuid.UUID
	// SetAssistantMessageID stores the assistant message identifier.
	SetAssistantMessageID(id uuid.UUID)
	// SelectedSkills returns the skills selected for the turn.
	SelectedSkills() []assistant.SelectedSkill
	// TokenUsage returns the accumulated token usage for the turn.
	TokenUsage() assistant.Usage
	// AddTokenUsage accumulates token usage into the current turn totals.
	AddTokenUsage(usage assistant.Usage)
	// TurnID returns the current turn identifier.
	TurnID() uuid.UUID
	// NextTurnSequence returns the current sequence and increments it.
	NextTurnSequence() int64
	// TryBuildUserMessage builds the user message once and returns it when it should be persisted.
	TryBuildUserMessage(id uuid.UUID, now time.Time) (assistant.ChatMessage, bool)
	// BuildFailureMessage constructs the assistant failure message for an aborted turn.
	BuildFailureMessage(now time.Time, streamErr error) assistant.ChatMessage
	// BuildFinalAssistantMessage constructs the final assistant chat message for the turn.
	BuildFinalAssistantMessage(now time.Time) assistant.ChatMessage
	// AppendAssistantContent appends streamed assistant text to the final response buffer.
	AppendAssistantContent(text string)
	// TryMarkRunTurnRecoveryAttempted marks recovery as attempted and reports whether this is the first attempt.
	TryMarkRunTurnRecoveryAttempted() bool
	// HasExceededMaxActionCycles increments the action cycle counter and reports whether the limit was exceeded.
	HasExceededMaxActionCycles() bool
	// HasExceededRepeatedActionCalls reports whether the same action signature repeated too many times.
	HasExceededRepeatedActionCalls(functionName, arguments string) bool
}

// turnSession is the default TurnSession implementation.
type turnSession struct {
	conversation              assistant.Conversation
	conversationCreated       bool
	model                     string
	request                   assistant.TurnRequest
	assistantMessageID        uuid.UUID
	selectedSkills            []assistant.SelectedSkill
	tokenUsage                assistant.Usage
	turnID                    uuid.UUID
	turnSequence              int64
	userMessage               assistant.ChatMessage
	userMessageBuildAttempted bool
	runTurnRecoveryAttempted  bool
	assistantMessageContent   strings.Builder
	tracker                   *actionCycleTracker
}

// NewTurnSession initializes the prepared request and mutable state used while streaming a response.
func NewTurnSession(
	conversation assistant.Conversation,
	conversationCreated bool,
	userMessage string,
	selectedSkills []assistant.SelectedSkill,
	request assistant.TurnRequest,
	maxActionCycles int,
) TurnSession {
	session := &turnSession{
		conversation:        conversation,
		conversationCreated: conversationCreated,
		model:               request.Model,
		request:             request,
		turnID:              uuid.New(),
		selectedSkills:      selectedSkills,
		tracker: newActionCycleTracker(
			maxActionCycles,
			MAX_REPEATED_ACTION_CALL_HIT,
		),
	}

	session.userMessage = assistant.ChatMessage{
		ConversationID: conversation.ID,
		TurnID:         session.turnID,
		TurnSequence:   session.NextTurnSequence(),
		ChatRole:       assistant.ChatRole_User,
		Content:        userMessage,
		Model:          session.model,
		MessageState:   assistant.ChatMessageState_Completed,
	}
	return session
}

// NextTurnSequence returns the current turn sequence value and increments it for the next message.
func (s *turnSession) NextTurnSequence() int64 {
	current := s.turnSequence
	s.turnSequence++
	return current
}

// BuildFinalAssistantMessage constructs the persisted assistant message for a completed turn.
func (s *turnSession) BuildFinalAssistantMessage(now time.Time) assistant.ChatMessage {
	if s.assistantMessageID == uuid.Nil {
		s.assistantMessageID = uuid.New()
	}

	return assistant.ChatMessage{
		ID:               s.assistantMessageID,
		ConversationID:   s.conversation.ID,
		TurnID:           s.turnID,
		TurnSequence:     s.NextTurnSequence(),
		ChatRole:         assistant.ChatRole_Assistant,
		Content:          s.assistantMessageContent.String(),
		SelectedSkills:   s.selectedSkills,
		Model:            s.model,
		MessageState:     assistant.ChatMessageState_Completed,
		PromptTokens:     s.tokenUsage.PromptTokens,
		CompletionTokens: s.tokenUsage.CompletionTokens,
		TotalTokens:      s.tokenUsage.TotalTokens,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// AppendAssistantContent appends streamed assistant text to the final assistant message buffer.
func (s *turnSession) AppendAssistantContent(text string) {
	s.assistantMessageContent.WriteString(text)
}

// TryBuildUserMessage builds the user message for persistence, ensuring it's only built once after the assistant assigns turn metadata.
func (s *turnSession) TryBuildUserMessage(id uuid.UUID, now time.Time) (assistant.ChatMessage, bool) {
	if s.userMessageBuildAttempted {
		return assistant.ChatMessage{}, false
	}

	s.userMessageBuildAttempted = true
	if s.userMessage.ID == uuid.Nil {
		if id != uuid.Nil {
			s.userMessage.ID = id
		} else {
			s.userMessage.ID = uuid.New()
		}
	}
	if s.userMessage.CreatedAt.IsZero() {
		s.userMessage.CreatedAt = now
	}
	if s.userMessage.UpdatedAt.IsZero() {
		s.userMessage.UpdatedAt = s.userMessage.CreatedAt
	}

	return s.userMessage, true
}

// BuildFailureMessage constructs the assistant failure message for an aborted turn.
func (s *turnSession) BuildFailureMessage(now time.Time, streamErr error) assistant.ChatMessage {
	if s.assistantMessageID == uuid.Nil {
		s.assistantMessageID = uuid.New()
	}

	errorMessage := streamErr.Error()
	return assistant.ChatMessage{
		ID:               s.assistantMessageID,
		ConversationID:   s.conversation.ID,
		TurnID:           s.turnID,
		TurnSequence:     s.NextTurnSequence(),
		ChatRole:         assistant.ChatRole_Assistant,
		Content:          "",
		SelectedSkills:   s.selectedSkills,
		Model:            s.model,
		MessageState:     assistant.ChatMessageState_Failed,
		ErrorMessage:     &errorMessage,
		PromptTokens:     s.tokenUsage.PromptTokens,
		CompletionTokens: s.tokenUsage.CompletionTokens,
		TotalTokens:      s.tokenUsage.TotalTokens,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// HasExceededMaxActionCycles increments the action cycle count and reports whether the limit was exceeded.
func (s *turnSession) HasExceededMaxActionCycles() bool {
	return s.tracker.hasExceededMaxCycles()
}

// HasExceededRepeatedActionCalls reports whether the same action signature repeated too many times.
func (s *turnSession) HasExceededRepeatedActionCalls(functionName, arguments string) bool {
	return s.tracker.hasExceededMaxActionCalls(functionName, arguments)
}

// Conversation returns the target conversation for the turn.
func (s *turnSession) Conversation() assistant.Conversation {
	return s.conversation
}

// ConversationCreated reports whether the conversation was created for this turn.
func (s *turnSession) ConversationCreated() bool {
	return s.conversationCreated
}

// Request returns a snapshot of the current request.
func (s *turnSession) Request() assistant.TurnRequest {
	return s.request
}

// UpdateRequest exposes the current request for in-place mutation.
func (s *turnSession) UpdateRequest(apply func(request *assistant.TurnRequest)) {
	apply(&s.request)
	s.model = s.request.Model
}

// Model returns the current request model name.
func (s *turnSession) Model() string {
	return s.model
}

// AssistantMessageID returns the assigned assistant message identifier, if any.
func (s *turnSession) AssistantMessageID() uuid.UUID {
	return s.assistantMessageID
}

// SetAssistantMessageID stores the assistant message identifier.
func (s *turnSession) SetAssistantMessageID(id uuid.UUID) {
	s.assistantMessageID = id
}

// SelectedSkills returns the skills selected for the turn.
func (s *turnSession) SelectedSkills() []assistant.SelectedSkill {
	return s.selectedSkills
}

// TokenUsage returns the accumulated token usage for the turn.
func (s *turnSession) TokenUsage() assistant.Usage {
	return s.tokenUsage
}

// AddTokenUsage accumulates token usage into the current turn totals.
func (s *turnSession) AddTokenUsage(usage assistant.Usage) {
	s.tokenUsage.CompletionTokens += usage.CompletionTokens
	s.tokenUsage.PromptTokens += usage.PromptTokens
	s.tokenUsage.TotalTokens += usage.TotalTokens
}

// TurnID returns the current turn identifier.
func (s *turnSession) TurnID() uuid.UUID {
	return s.turnID
}

// TryMarkRunTurnRecoveryAttempted marks recovery as attempted and reports whether this is the first attempt.
func (s *turnSession) TryMarkRunTurnRecoveryAttempted() bool {
	if s.runTurnRecoveryAttempted {
		return false
	}
	s.runTurnRecoveryAttempted = true
	return true
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
