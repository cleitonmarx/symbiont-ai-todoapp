package domain

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AssistantActionApprovalKey uniquely identifies one action approval flow.
type AssistantActionApprovalKey struct {
	ConversationID uuid.UUID
	TurnID         uuid.UUID
	ActionCallID   string
}

// AssistantActionApprovalDecision represents the final approval decision for an action.
type AssistantActionApprovalDecision struct {
	Key        AssistantActionApprovalKey
	ActionName string
	Status     ChatMessageApprovalStatus
	Reason     *string
	DecidedAt  time.Time
}

// Validate checks the integrity of the approval decision fields.
func (d AssistantActionApprovalDecision) Validate() error {
	switch {
	case d.Key.ConversationID == uuid.Nil:
		return NewValidationErr("conversation_id is required")
	case d.Key.TurnID == uuid.Nil:
		return NewValidationErr("turn_id is required")
	case strings.TrimSpace(d.Key.ActionCallID) == "":
		return NewValidationErr("action_call_id is required")
	case strings.TrimSpace(d.ActionName) == "":
		return NewValidationErr("action_name is required")
	case d.DecidedAt.IsZero():
		return NewValidationErr("decided_at is required")
	}

	switch d.Status {
	case ChatMessageApprovalStatus_Approved,
		ChatMessageApprovalStatus_Rejected:
		return nil
	default:
		return NewValidationErr("status must be APPROVED or REJECTED")
	}
}

// AssistantActionApprovalDispatcher coordinates in-flight human approvals.
type AssistantActionApprovalDispatcher interface {
	// Wait blocks until a decision is available for the given key or the context is canceled.
	Wait(ctx context.Context, key AssistantActionApprovalKey) (AssistantActionApprovalDecision, error)
	// Dispatch pushes a final decision to a waiting stream. Returns false when no waiter exists.
	Dispatch(decision AssistantActionApprovalDecision) bool
}
