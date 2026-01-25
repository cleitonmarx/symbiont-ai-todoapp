package pubsub

import (
	"context"

	pubsubV2 "cloud.google.com/go/pubsub/v2"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// PubSubEventPublisher implements domain.EventPublisher using Google Cloud Pub/Sub
type PubSubEventPublisher struct {
	Client *pubsubV2.Client
}

// NewPubSubEventPublisher creates a new instance of PubSubEventPublisher
func NewPubSubEventPublisher(client *pubsubV2.Client) PubSubEventPublisher {
	return PubSubEventPublisher{Client: client}
}

// PublishEvent publishes the given event to the appropriate Pub/Sub topic
func (p PubSubEventPublisher) PublishEvent(ctx context.Context, event domain.OutboxEvent) error {
	spanCtx, span := tracing.Start(ctx,
		trace.WithAttributes(
			attribute.String("event_id", event.ID.String()),
			attribute.String("event_type", event.EventType),
			attribute.String("topic", event.Topic),
		),
	)
	defer span.End()

	result := p.Client.Publisher(event.Topic).Publish(spanCtx, &pubsubV2.Message{
		Data: event.Payload,
		Attributes: map[string]string{
			"event_type": event.EventType,
			"entity_id":  event.EntityID.String(),
		},
	})

	_, err := result.Get(ctx)
	return err
}

// InitPublisher initializes the TodoEventPublisher implementation
type InitPublisher struct {
	Client *pubsubV2.Client `resolve:""`
}

// Initialize registers the PubSubEventPublisher as the implementation of TodoEventPublisher
func (i *InitPublisher) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.TodoEventPublisher](NewPubSubEventPublisher(i.Client))
	return ctx, nil
}
