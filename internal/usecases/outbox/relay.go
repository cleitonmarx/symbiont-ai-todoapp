package outbox

import (
	"context"
	"log"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// Relay defines the interface for relaying outbox events
type Relay interface {
	// Execute processes pending outbox events and relays them
	Execute(ctx context.Context) error
}

// RelayImpl implements Relay interface.
type RelayImpl struct {
	Uow       transaction.UnitOfWork `resolve:""`
	Publisher outbox.EventPublisher  `resolve:""`
	Logger    *log.Logger            `resolve:""`
}

// NewRelayImpl creates a new instance of RelayImpl.
func NewRelayImpl(uow transaction.UnitOfWork, publisher outbox.EventPublisher, logger *log.Logger) RelayImpl {
	return RelayImpl{
		Uow:       uow,
		Publisher: publisher,
		Logger:    logger,
	}
}

// Execute processes pending outbox events and relays them.
func (r RelayImpl) Execute(ctx context.Context) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	err := r.Uow.Execute(spanCtx, func(uowCtx context.Context, scope transaction.Scope) error {
		outboxRepo := scope.Outbox()

		events, err := outboxRepo.FetchPendingEvents(uowCtx, 100)
		if err != nil {
			return err
		}

		for _, event := range events {
			if err := r.relayEvent(uowCtx, outboxRepo, event); err != nil {
				r.Logger.Printf("relay failed for event %s: %v", event.ID, err)
			}
		}
		return nil
	})
	if telemetry.IsErrorRecorded(span, err) {
		return err
	}
	return nil
}

// relayEvent processes and relays a single outbox event.
func (r RelayImpl) relayEvent(ctx context.Context, outboxRepo outbox.Repository, event outbox.Event) error {
	if err := r.Publisher.PublishEvent(ctx, event); err != nil {
		if event.RetryCount+1 >= event.MaxRetries {
			return outboxRepo.UpdateEvent(ctx, event.ID, outbox.Status_Failed, event.RetryCount+1, err.Error())
		}
		return outboxRepo.UpdateEvent(ctx, event.ID, outbox.Status_Pending, event.RetryCount+1, err.Error())
	}
	return outboxRepo.UpdateEvent(ctx, event.ID, outbox.Status_Processed, event.RetryCount, "")
}
