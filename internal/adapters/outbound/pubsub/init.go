package pubsub

import (
	"context"
	"fmt"
	"log"

	pubsubV2 "cloud.google.com/go/pubsub/v2"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitClient initializes the Pub/Sub client and registers it in the dependency container
type InitClient struct {
	Logger    *log.Logger `resolve:""`
	ProjectID string      `config:"PUBSUB_PROJECT_ID"`
	client    *pubsubV2.Client
}

// Initialize initializes the Pub/Sub client and registers it in the dependency container.
func (i *InitClient) Initialize(ctx context.Context) (context.Context, error) {
	if i.client == nil {
		cfg := &pubsubV2.ClientConfig{
			EnableOpenTelemetryTracing: true,
		}
		client, err := pubsubV2.NewClientWithConfig(ctx, i.ProjectID, cfg)
		if err != nil {
			return ctx, fmt.Errorf("failed to create pubsub client: %w", err)
		}
		i.client = client
	}

	depend.Register(i.client)

	return ctx, nil
}

// Close closes the Pub/Sub client and logs any errors that occur during closure.
func (i *InitClient) Close() {
	if err := i.client.Close(); err != nil {
		i.Logger.Printf("InitClient:failed to close pubsub client: %v", err)
	}
}

// InitPublisher initializes the TodoEventPublisher implementation
type InitPublisher struct {
	Client *pubsubV2.Client `resolve:""`
}

// Initialize registers the PubSubEventPublisher as the implementation of TodoEventPublisher
func (i *InitPublisher) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[outbox.EventPublisher](NewPubSubEventPublisher(i.Client))
	return ctx, nil
}
