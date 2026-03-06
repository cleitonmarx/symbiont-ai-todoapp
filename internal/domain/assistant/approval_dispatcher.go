package assistant

import (
	"context"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/google/uuid"
)

// ActionApprovalKey uniquely identifies one action approval flow.
type ActionApprovalKey struct {
	ConversationID uuid.UUID
	TurnID         uuid.UUID
	ActionCallID   string
}

// ActionApprovalDecision represents the final approval decision for an action.
type ActionApprovalDecision struct {
	Key        ActionApprovalKey
	ActionName string
	Status     ChatMessageApprovalStatus
	Reason     *string
	DecidedAt  time.Time
}

// Validate checks the integrity of the approval decision fields.
func (d ActionApprovalDecision) Validate() error {
	switch {
	case d.Key.ConversationID == uuid.Nil:
		return core.NewValidationErr("conversation_id is required")
	case d.Key.TurnID == uuid.Nil:
		return core.NewValidationErr("turn_id is required")
	case strings.TrimSpace(d.Key.ActionCallID) == "":
		return core.NewValidationErr("action_call_id is required")
	case strings.TrimSpace(d.ActionName) == "":
		return core.NewValidationErr("action_name is required")
	case d.DecidedAt.IsZero():
		return core.NewValidationErr("decided_at is required")
	}

	switch d.Status {
	case ChatMessageApprovalStatus_Approved,
		ChatMessageApprovalStatus_Rejected:
		return nil
	default:
		return core.NewValidationErr("status must be APPROVED or REJECTED")
	}
}

// ActionApprovalDispatcher coordinates in-flight human approvals.
type ActionApprovalDispatcher interface {
	// Wait blocks until a decision is available for the given key or the context is canceled.
	Wait(ctx context.Context, key ActionApprovalKey) (ActionApprovalDecision, error)
	// Dispatch pushes a final decision to a waiting stream. Returns false when no waiter exists.
	Dispatch(ctx context.Context, decision ActionApprovalDecision) bool
}
