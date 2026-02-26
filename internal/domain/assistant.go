package domain

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AssistantEventType represents the type of event in an assistant stream.
type AssistantEventType string

const (
	AssistantEventType_TurnStarted            AssistantEventType = "turn_started"
	AssistantEventType_MessageDelta           AssistantEventType = "message_delta"
	AssistantEventType_ActionRequested        AssistantEventType = "action_requested"
	AssistantEventType_ActionApprovalRequired AssistantEventType = "action_approval_required"
	AssistantEventType_ActionApprovalResolved AssistantEventType = "action_approval_resolved"
	AssistantEventType_ActionStarted          AssistantEventType = "action_started"
	AssistantEventType_ActionCompleted        AssistantEventType = "action_completed"
	AssistantEventType_TurnCompleted          AssistantEventType = "turn_completed"
)

// AssistantUsage contains token usage for one assistant turn.
type AssistantUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// AssistantTurnStarted contains metadata for a streaming assistant session.
type AssistantTurnStarted struct {
	ConversationID      uuid.UUID `json:"conversation_id"`
	UserMessageID       uuid.UUID `json:"user_message_id"`
	AssistantMessageID  uuid.UUID `json:"assistant_message_id"`
	ConversationCreated bool      `json:"conversation_created"`
}

// AssistantMessageDelta contains a text delta from the stream.
type AssistantMessageDelta struct {
	Text string `json:"text"`
}

// AssistantActionCall contains one action invocation requested by the assistant.
type AssistantActionCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
	Text  string `json:"text"`
}

// AssistantActionApprovalRequired indicates an action is blocked waiting for human approval.
type AssistantActionApprovalRequired struct {
	ConversationID uuid.UUID     `json:"conversation_id"`
	TurnID         uuid.UUID     `json:"turn_id"`
	ActionCallID   string        `json:"action_call_id"`
	Name           string        `json:"name"`
	Input          string        `json:"input"`
	Title          string        `json:"title"`
	Description    string        `json:"description"`
	PreviewFields  []string      `json:"preview_fields,omitempty"`
	Timeout        time.Duration `json:"timeout"`
}

// AssistantActionApprovalResolved indicates the final approval decision for one action.
type AssistantActionApprovalResolved struct {
	ConversationID uuid.UUID                 `json:"conversation_id"`
	TurnID         uuid.UUID                 `json:"turn_id"`
	ActionCallID   string                    `json:"action_call_id"`
	Name           string                    `json:"name"`
	Status         ChatMessageApprovalStatus `json:"status"`
	Reason         *string                   `json:"reason,omitempty"`
}

// AssistantActionCompleted indicates an action invocation has finished.
type AssistantActionCompleted struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Success       bool    `json:"success"`
	Error         *string `json:"error,omitempty"`
	ShouldRefetch bool    `json:"should_refetch"`
}

// AssistantTurnCompleted contains completion metadata and usage.
type AssistantTurnCompleted struct {
	Usage              AssistantUsage `json:"usage"`
	AssistantMessageID string         `json:"assistant_message_id"`
	CompletedAt        string         `json:"completed_at"`
}

// AssistantEventCallback is called for each assistant turn event.
type AssistantEventCallback func(context.Context, AssistantEventType, any) error

// AssistantMessage represents a message exchanged during assistant turns.
type AssistantMessage struct {
	Role         ChatRole
	Content      string
	ActionCallID *string
	ActionCalls  []AssistantActionCall
}

// IsActionCallSuccess returns true when this message is a successful action result.
func (m AssistantMessage) IsActionCallSuccess() bool {
	return m.Role == ChatRole_Tool &&
		m.ActionCallID != nil &&
		!strings.Contains(m.Content, "error")
}

// AssistantActionDefinition describes one action that can be used by the assistant.
type AssistantActionDefinition struct {
	Name        string
	Description string
	Input       AssistantActionInput
	Hints       AssistantActionHints
	Approval    AssistantActionApproval
}

// AssistantActionApproval holds human approval policy metadata for one action.
type AssistantActionApproval struct {
	Required bool
	// Title is a short approval title for UI prompts.
	Title string
	// Description explains what the action will do and why approval is needed.
	Description string
	// PreviewFields are JSON paths (e.g. todos[].title) used by the UI to render a readable approval preview.
	PreviewFields []string
	// Timeout controls how long the system should wait for a decision.
	Timeout time.Duration
}

// ComposeHint composes the action hints into a single string for prompting.
func (d AssistantActionDefinition) ComposeHint() string {
	parts := make([]string, 0, 3)
	if useWhen := strings.TrimSpace(d.Hints.UseWhen); useWhen != "" {
		parts = append(parts, "Use: "+useWhen)
	}
	if avoidWhen := strings.TrimSpace(d.Hints.AvoidWhen); avoidWhen != "" {
		parts = append(parts, "Avoid: "+avoidWhen)
	}
	if argRules := strings.TrimSpace(d.Hints.ArgRules); argRules != "" {
		parts = append(parts, "Args: "+argRules)
	}

	if len(parts) == 0 {
		return "Follow the tool schema and description."
	}
	return strings.Join(parts, " ")
}

// HasHints returns true if any of the hint fields are non-empty after trimming whitespace.
func (d AssistantActionDefinition) HasHints() bool {
	return strings.TrimSpace(d.Hints.UseWhen) != "" ||
		strings.TrimSpace(d.Hints.AvoidWhen) != "" ||
		strings.TrimSpace(d.Hints.ArgRules) != ""
}

// RequiresApproval returns true when the action policy requires explicit human approval.
func (d AssistantActionDefinition) RequiresApproval() bool {
	return d.Approval.Required
}

// AssistantActionHints holds compact, runtime guidance for dynamic prompt injection.
type AssistantActionHints struct {
	UseWhen   string
	AvoidWhen string
	ArgRules  string
}

// AssistantActionField represents one action input field.
type AssistantActionField struct {
	Type        string
	Description string
	Required    bool
	Fields      map[string]AssistantActionField
	Items       *AssistantActionField
	Format      string
	Enum        []any
}

// AssistantActionInput describes the action input shape.
type AssistantActionInput struct {
	Type   string
	Fields map[string]AssistantActionField
}

// AssistantTurnRequest is the domain request for one assistant turn.
type AssistantTurnRequest struct {
	Model    string
	Messages []AssistantMessage
	Stream   bool
	// Optional generation settings.
	Temperature      *float64
	TopP             *float64
	MaxTokens        *int
	FrequencyPenalty *float64
	AvailableActions []AssistantActionDefinition
}

// AssistantTurnResponse contains the final assistant message and usage for non-stream mode.
type AssistantTurnResponse struct {
	Content string
	Usage   AssistantUsage
}

// Assistant defines assistant interaction in domain terms.
type Assistant interface {
	// RunTurn streams one assistant turn.
	RunTurn(ctx context.Context, req AssistantTurnRequest, onEvent AssistantEventCallback) error

	// RunTurnSync executes one assistant turn and returns the final response.
	RunTurnSync(ctx context.Context, req AssistantTurnRequest) (AssistantTurnResponse, error)
}

// AssistantAction represents one executable assistant action.
type AssistantAction interface {
	// Definition returns the action definition for this action.
	Definition() AssistantActionDefinition
	// StatusMessage returns a status message about the action execution, or a default message if not implemented.
	StatusMessage() string
	// Execute runs the action with the given input and returns the resulting assistant message.
	Execute(context.Context, AssistantActionCall, []AssistantMessage) AssistantMessage
}

// AssistantActionRegistry resolves and executes assistant actions.
type AssistantActionRegistry interface {
	// Execute runs the given action call and returns the resulting assistant message.
	Execute(context.Context, AssistantActionCall, []AssistantMessage) AssistantMessage
	// GetDefinition returns one action definition by name.
	GetDefinition(actionName string) (AssistantActionDefinition, bool)
	// StatusMessage returns a status message about the action execution, or a default message if not implemented.
	StatusMessage(actionName string) string
	// ListRelevant returns relevant assistant action definitions based on the user input.
	ListRelevant(ctx context.Context, userInput string) []AssistantActionDefinition
}
