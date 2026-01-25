package pubsub

import (
	"context"
	"fmt"
	"log"

	pubsubV2 "cloud.google.com/go/pubsub/v2"
	"github.com/cleitonmarx/symbiont/depend"
)

type InitClient struct {
	Logger    *log.Logger `resolve:""`
	ProjectID string      `config:"PUBSUB_PROJECT_ID"`
	client    *pubsubV2.Client
}

func (i *InitClient) Initialize(ctx context.Context) (context.Context, error) {
	if i.client == nil {
		client, err := pubsubV2.NewClient(ctx, i.ProjectID)
		if err != nil {
			return ctx, fmt.Errorf("failed to create pubsub client: %w", err)
		}
		i.client = client
	}

	depend.Register(i.client)

	return ctx, nil
}

func (i *InitClient) Close() {
	if err := i.client.Close(); err != nil {
		i.Logger.Printf("InitClient:failed to close pubsub client: %v", err)
	}
}
