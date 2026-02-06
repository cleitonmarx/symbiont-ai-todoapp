package usecases

import (
	"context"
	"log"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
)

// RelayOutbox defines the interface for relaying outbox events
type RelayOutbox interface {
	// Execute processes pending outbox events and relays them
	Execute(ctx context.Context) error
}

// RelayOutboxImpl implements domain.OutboxRelay
type RelayOutboxImpl struct {
	Uow       domain.UnitOfWork         `resolve:""`
	Publisher domain.TodoEventPublisher `resolve:""` // publishes to event bus
	Logger    *log.Logger               `resolve:""`
}

// NewRelayOutboxImpl creates a new instance
func NewRelayOutboxImpl(uow domain.UnitOfWork, publisher domain.TodoEventPublisher, logger *log.Logger) RelayOutboxImpl {
	return RelayOutboxImpl{
		Uow:       uow,
		Publisher: publisher,
		Logger:    logger,
	}
}

// Execute processes pending outbox events and relays them
func (r RelayOutboxImpl) Execute(ctx context.Context) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	err := r.Uow.Execute(spanCtx, func(uow domain.UnitOfWork) error {
		events, err := uow.Outbox().FetchPendingEvents(spanCtx, 100)
		if err != nil {
			return err
		}

		for _, event := range events {
			if err := r.relayEvent(spanCtx, uow, event); err != nil {
				r.Logger.Printf("relay failed for event %s: %v", event.ID, err)
			}
		}
		return nil
	})
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}
	return nil
}

// relayEvent processes and relays a single outbox event
func (r RelayOutboxImpl) relayEvent(ctx context.Context, uow domain.UnitOfWork, event domain.OutboxEvent) error {

	if err := r.Publisher.PublishEvent(ctx, event); err != nil {
		if event.RetryCount+1 >= event.MaxRetries {
			return uow.Outbox().UpdateEvent(ctx, event.ID, "FAILED", event.RetryCount+1, err.Error())
		}
		return uow.Outbox().UpdateEvent(ctx, event.ID, "PENDING", event.RetryCount+1, err.Error())
	}
	return uow.Outbox().DeleteEvent(ctx, event.ID)
}

// InitRelayOutbox is used to initialize the RelayOutbox in the dependency container
type InitRelayOutbox struct {
	Uow       domain.UnitOfWork         `resolve:""`
	Logger    *log.Logger               `resolve:""`
	Publisher domain.TodoEventPublisher `resolve:""`
}

// Initialize registers the RelayOutbox implementation in the dependency container
func (iro InitRelayOutbox) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[RelayOutbox](NewRelayOutboxImpl(iro.Uow, iro.Publisher, iro.Logger))
	return ctx, nil
}
