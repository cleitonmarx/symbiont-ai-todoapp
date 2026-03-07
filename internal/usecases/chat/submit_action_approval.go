package chat

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/google/uuid"
)

// SubmitActionApproval is the use case for submitting one human approval decision.
type SubmitActionApproval interface {
	Execute(ctx context.Context, input SubmitActionApprovalInput) error
}

// SubmitActionApprovalInput defines the payload required to dispatch one decision.
type SubmitActionApprovalInput struct {
	ConversationID uuid.UUID
	TurnID         uuid.UUID
	ActionCallID   string
	ActionName     string
	Status         assistant.ChatMessageApprovalStatus
	Reason         *string
}

// SubmitActionApprovalImpl publishes approval decisions to the approvals topic.
type SubmitActionApprovalImpl struct {
	publisher outbox.EventPublisher
}

// NewSubmitActionApprovalImpl creates a SubmitActionApprovalImpl.
func NewSubmitActionApprovalImpl(publisher outbox.EventPublisher) *SubmitActionApprovalImpl {
	return &SubmitActionApprovalImpl{publisher: publisher}
}

// Execute validates and publishes one approval decision.
func (uc SubmitActionApprovalImpl) Execute(ctx context.Context, input SubmitActionApprovalInput) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	payload := assistant.ActionApprovalDecision{
		Key: assistant.ActionApprovalKey{
			ConversationID: input.ConversationID,
			TurnID:         input.TurnID,
			ActionCallID:   strings.TrimSpace(input.ActionCallID),
		},
		ActionName: strings.TrimSpace(input.ActionName),
		Status:     input.Status,
		Reason:     input.Reason,
		DecidedAt:  time.Now().UTC(),
	}
	if err := payload.Validate(); err != nil {
		return err
	}

	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return uc.publisher.PublishEvent(spanCtx, outbox.Event{
		ID:         uuid.New(),
		EntityType: outbox.EntityType_ChatMessage,
		EntityID:   input.ConversationID,
		Topic:      outbox.Topic_ActionApprovals,
		EventType:  outbox.EventType_ACTION_APPROVAL_DECIDED,
		Payload:    encodedPayload,
	})
}
