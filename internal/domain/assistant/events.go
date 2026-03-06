package assistant

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of event in an assistant stream.
type EventType string

const (
	// EventType_TurnStarted indicates a chat turn has started.
	EventType_TurnStarted EventType = "turn_started"
	// EventType_MessageDelta indicates a streaming text delta event.
	EventType_MessageDelta EventType = "message_delta"
	// EventType_ActionRequested indicates the model requested a tool/action call.
	EventType_ActionRequested EventType = "action_requested"
	// EventType_ActionApprovalRequired indicates an action is waiting for human approval.
	EventType_ActionApprovalRequired EventType = "action_approval_required"
	// EventType_ActionApprovalResolved indicates an action approval decision was made.
	EventType_ActionApprovalResolved EventType = "action_approval_resolved"
	// EventType_ActionStarted indicates action execution started.
	EventType_ActionStarted EventType = "action_started"
	// EventType_ActionCompleted indicates action execution completed.
	EventType_ActionCompleted EventType = "action_completed"
	// EventType_TurnCompleted indicates a chat turn finished.
	EventType_TurnCompleted EventType = "turn_completed"
)

// Usage contains token usage for one assistant turn.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// TurnStarted contains metadata for a streaming assistant session.
type TurnStarted struct {
	ConversationID      uuid.UUID       `json:"conversation_id"`
	UserMessageID       uuid.UUID       `json:"user_message_id"`
	AssistantMessageID  uuid.UUID       `json:"assistant_message_id"`
	ConversationCreated bool            `json:"conversation_created"`
	TurnID              uuid.UUID       `json:"turn_id"`
	SelectedSkills      []SelectedSkill `json:"selected_skills,omitempty"`
}

// MessageDelta contains a text delta from the stream.
type MessageDelta struct {
	Text string `json:"text"`
}

// ActionApprovalRequired indicates an action is blocked waiting for human approval.
type ActionApprovalRequired struct {
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

// ActionApprovalResolved indicates the final approval decision for one action.
type ActionApprovalResolved struct {
	ConversationID uuid.UUID                 `json:"conversation_id"`
	TurnID         uuid.UUID                 `json:"turn_id"`
	ActionCallID   string                    `json:"action_call_id"`
	Name           string                    `json:"name"`
	Status         ChatMessageApprovalStatus `json:"status"`
	Reason         *string                   `json:"reason,omitempty"`
}

// ActionCompleted indicates an action invocation has finished.
type ActionCompleted struct {
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	Success         bool                       `json:"success"`
	Error           *string                    `json:"error,omitempty"`
	ShouldRefetch   bool                       `json:"should_refetch"`
	ApprovalStatus  *ChatMessageApprovalStatus `json:"approval_status,omitempty"`
	ActionExecuted  *bool                      `json:"action_executed,omitempty"`
	OutputPreview   *string                    `json:"output_preview,omitempty"`
	OutputTruncated bool                       `json:"output_truncated,omitempty"`
}

// TurnCompleted contains completion metadata and usage.
type TurnCompleted struct {
	Usage              Usage  `json:"usage"`
	AssistantMessageID string `json:"assistant_message_id"`
	CompletedAt        string `json:"completed_at"`
}

// EventCallback is called for each assistant turn event.
type EventCallback func(context.Context, EventType, any) error
