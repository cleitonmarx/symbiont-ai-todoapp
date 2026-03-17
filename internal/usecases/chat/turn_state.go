package chat

import (
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/google/uuid"
)

const (
	// MAX_REPEATED_ACTION_CALL_HIT is the limit for repeated action-call detections before aborting.
	MAX_REPEATED_ACTION_CALL_HIT = 5
)

// TurnState encapsulates the mutable state and operations for a single assistant turn, including recovery and action loop tracking.
type TurnState interface {
	// Conversation returns the target conversation for the turn.
	Conversation() assistant.Conversation
	// ConversationCreated reports whether the conversation was created for this turn.
	ConversationCreated() bool
	// Request returns a snapshot of the current turn request.
	Request() assistant.TurnRequest
	// AppendRequestMessages appends follow-up messages to the current turn request.
	AppendRequestMessages(messages ...assistant.Message)
	// ApplyRecoveryPolicy prepares the request for one recovery-only retry after an internal turn failure.
	ApplyRecoveryPolicy(runErr error, maxMessages int)
	// Model returns the current request model name.
	Model() string
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
	// AppendAssistantContent appends streamed assistant text to the final response buffer.
	AppendAssistantContent(text string)
	// AssistantContent returns the accumulated assistant response content for the turn.
	AssistantContent() string
	// MarkActionCallPersisted records that an assistant action-call message was stored for this turn.
	MarkActionCallPersisted()
	// HasPersistedActionCall reports whether the turn stored any assistant action-call message.
	HasPersistedActionCall() bool
	// HasExceededMaxActionCycles increments the action cycle counter and reports whether the limit was exceeded.
	HasExceededMaxActionCycles() bool
	// HasExceededRepeatedActionCalls reports whether the same action signature repeated too many times.
	HasExceededRepeatedActionCalls(functionName, arguments string) bool
}

// turnState is the default TurnState implementation.
type turnState struct {
	conversation            assistant.Conversation
	conversationCreated     bool
	model                   string
	request                 assistant.TurnRequest
	selectedSkills          []assistant.SelectedSkill
	tokenUsage              assistant.Usage
	turnID                  uuid.UUID
	turnSequence            int64
	assistantMessageContent strings.Builder
	actionCallPersisted     bool
	tracker                 *actionCycleTracker
}

// NewTurnState initializes the prepared request and mutable state used while streaming a response.
func NewTurnState(
	conversation assistant.Conversation,
	conversationCreated bool,
	selectedSkills []assistant.SelectedSkill,
	request assistant.TurnRequest,
	maxActionCycles int,
) TurnState {
	state := &turnState{
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
	return state
}

// NextTurnSequence returns the current turn sequence value and increments it for the next message.
func (s *turnState) NextTurnSequence() int64 {
	current := s.turnSequence
	s.turnSequence++
	return current
}

// AppendAssistantContent appends streamed assistant text to the final assistant message buffer.
func (s *turnState) AppendAssistantContent(text string) {
	s.assistantMessageContent.WriteString(text)
}

// AssistantContent returns the accumulated assistant response content for the turn.
func (s *turnState) AssistantContent() string {
	return s.assistantMessageContent.String()
}

// MarkActionCallPersisted records that an assistant action-call message was stored for this turn.
func (s *turnState) MarkActionCallPersisted() {
	s.actionCallPersisted = true
}

// HasPersistedActionCall reports whether the turn stored any assistant action-call message.
func (s *turnState) HasPersistedActionCall() bool {
	return s.actionCallPersisted
}

// HasExceededMaxActionCycles increments the action cycle count and reports whether the limit was exceeded.
func (s *turnState) HasExceededMaxActionCycles() bool {
	return s.tracker.hasExceededMaxCycles()
}

// HasExceededRepeatedActionCalls reports whether the same action signature repeated too many times.
func (s *turnState) HasExceededRepeatedActionCalls(functionName, arguments string) bool {
	return s.tracker.hasExceededMaxActionCalls(functionName, arguments)
}

// Conversation returns the target conversation for the turn.
func (s *turnState) Conversation() assistant.Conversation {
	return s.conversation
}

// ConversationCreated reports whether the conversation was created for this turn.
func (s *turnState) ConversationCreated() bool {
	return s.conversationCreated
}

// Request returns a snapshot of the current request.
func (s *turnState) Request() assistant.TurnRequest {
	return s.request
}

// AppendRequestMessages appends follow-up messages to the current request.
func (s *turnState) AppendRequestMessages(messages ...assistant.Message) {
	s.request.Messages = append(s.request.Messages, messages...)
}

// ApplyRecoveryPolicy removes tools, compacts recent context, and appends one recovery instruction.
func (s *turnState) ApplyRecoveryPolicy(runErr error, maxMessages int) {
	s.request.AvailableActions = nil
	s.request.Messages = compactToLastMessages(s.request.Messages, maxMessages)
	s.request.Messages = append(s.request.Messages, assistant.Message{
		Role: assistant.ChatRole_System,
		Content: "The previous assistant turn failed due to an internal processing issue " +
			"(commonly tool execution failure or context size limit). " +
			"Internal error: " + truncateToFirstChars(strings.TrimSpace(runErr.Error()), 400) + ". " +
			"Reply to the user with a short apology and explain that the request failed due to an internal error. " +
			"Suggest retrying with a smaller scope. Do not claim actions succeeded.",
	})
}

// Model returns the current request model name.
func (s *turnState) Model() string {
	return s.model
}

// SelectedSkills returns the skills selected for the turn.
func (s *turnState) SelectedSkills() []assistant.SelectedSkill {
	return s.selectedSkills
}

// TokenUsage returns the accumulated token usage for the turn.
func (s *turnState) TokenUsage() assistant.Usage {
	return s.tokenUsage
}

// AddTokenUsage accumulates token usage into the current turn totals.
func (s *turnState) AddTokenUsage(usage assistant.Usage) {
	s.tokenUsage.CompletionTokens += usage.CompletionTokens
	s.tokenUsage.PromptTokens += usage.PromptTokens
	s.tokenUsage.TotalTokens += usage.TotalTokens
}

// TurnID returns the current turn identifier.
func (s *turnState) TurnID() uuid.UUID {
	return s.turnID
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
