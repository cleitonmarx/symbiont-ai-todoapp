package pubsub

import (
	"context"

	pubsubV2 "cloud.google.com/go/pubsub/v2"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// PubSubEventPublisher implements outbox.EventPublisher using Google Cloud Pub/Sub
type PubSubEventPublisher struct {
	Client *pubsubV2.Client
}

// NewPubSubEventPublisher creates a new instance of PubSubEventPublisher
func NewPubSubEventPublisher(client *pubsubV2.Client) PubSubEventPublisher {
	return PubSubEventPublisher{Client: client}
}

// PublishEvent publishes the given event to the appropriate Pub/Sub topic
func (p PubSubEventPublisher) PublishEvent(ctx context.Context, event outbox.Event) error {
	spanCtx, span := telemetry.Start(ctx,
		trace.WithAttributes(
			attribute.String("event_id", event.ID.String()),
			attribute.String("event_type", string(event.EventType)),
			attribute.String("topic", string(event.Topic)),
		),
	)
	defer span.End()

	result := p.Client.Publisher(string(event.Topic)).Publish(spanCtx, &pubsubV2.Message{
		Data: event.Payload,
		Attributes: map[string]string{
			"event_type": string(event.EventType),
			"entity_id":  event.EntityID.String(),
		},
	})

	_, err := result.Get(ctx)
	return err
}
