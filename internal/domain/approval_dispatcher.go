package domain

import (
	"context"
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

// AssistantActionApprovalDispatcher coordinates in-flight human approvals.
type AssistantActionApprovalDispatcher interface {
	// Wait blocks until a decision is available for the given key or the context is canceled.
	Wait(ctx context.Context, key AssistantActionApprovalKey) (AssistantActionApprovalDecision, error)
	// Dispatch pushes a final decision to a waiting stream. Returns false when no waiter exists.
	Dispatch(decision AssistantActionApprovalDecision) bool
}
