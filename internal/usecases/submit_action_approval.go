package usecases

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
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
	Status         domain.ChatMessageApprovalStatus
	Reason         *string
}

// SubmitActionApprovalImpl publishes approval decisions to the approvals topic.
type SubmitActionApprovalImpl struct {
	publisher domain.EventPublisher
}

// NewSubmitActionApprovalImpl creates a SubmitActionApprovalImpl.
func NewSubmitActionApprovalImpl(publisher domain.EventPublisher) *SubmitActionApprovalImpl {
	return &SubmitActionApprovalImpl{publisher: publisher}
}

// Execute validates and publishes one approval decision.
func (uc SubmitActionApprovalImpl) Execute(ctx context.Context, input SubmitActionApprovalInput) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	payload := domain.AssistantActionApprovalDecision{
		Key: domain.AssistantActionApprovalKey{
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

	return uc.publisher.PublishEvent(spanCtx, domain.OutboxEvent{
		ID:         uuid.New(),
		EntityType: domain.OutboxEntityType_ChatMessage,
		EntityID:   input.ConversationID,
		Topic:      domain.OutboxTopic_ActionApprovals,
		EventType:  domain.EventType_ACTION_APPROVAL_DECIDED,
		Payload:    encodedPayload,
	})
}

// InitSubmitActionApproval initializes and registers the SubmitActionApproval use case.
type InitSubmitActionApproval struct {
	Publisher domain.EventPublisher `resolve:""`
}

// Initialize registers SubmitActionApproval in the dependency container.
func (i InitSubmitActionApproval) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[SubmitActionApproval](NewSubmitActionApprovalImpl(i.Publisher))
	return ctx, nil
}
