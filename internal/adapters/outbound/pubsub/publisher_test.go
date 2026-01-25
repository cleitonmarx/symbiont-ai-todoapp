package pubsub

import (
	"context"
	"testing"
	"time"

	pubsubV2 "cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"cloud.google.com/go/pubsub/v2/pstest"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestPubSubEventPublisher_PublishEvent(t *testing.T) {
	eventID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	todoID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		event           domain.OutboxEvent
		expectErr       bool
		validateMessage func(*testing.T, *pubsubV2.Client, string)
	}{
		"success-publish-event": {
			event: domain.OutboxEvent{
				ID:         eventID,
				EventType:  "TODO_CREATED",
				EntityID:   todoID,
				Topic:      "todo-events",
				Payload:    []byte(`{"id":"223e4567-e89b-12d3-a456-426614174000","title":"Test Todo"}`),
				CreatedAt:  fixedTime,
				RetryCount: 0,
				MaxRetries: 3,
			},
			expectErr: false,
			validateMessage: func(t *testing.T, client *pubsubV2.Client, subName string) {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				messages := make([]*pubsubV2.Message, 0)

				err := client.Subscriber(subName).Receive(ctx, func(ctx context.Context, msg *pubsubV2.Message) {
					messages = append(messages, msg)
					msg.Ack() //nolint:errcheck
				})
				if err != nil && err != context.DeadlineExceeded {
					t.Fatalf("failed to receive: %v", err)
				}

				assert.Len(t, messages, 1)
				msg := messages[0]
				assert.Equal(t, []byte(`{"id":"223e4567-e89b-12d3-a456-426614174000","title":"Test Todo"}`), msg.Data)
				assert.Equal(t, "TODO_CREATED", msg.Attributes["event_type"])
				assert.Equal(t, todoID.String(), msg.Attributes["entity_id"])
			},
		},
		"error-topic-not-found": {
			event: domain.OutboxEvent{
				ID:         eventID,
				EventType:  "TODO_CREATED",
				EntityID:   todoID,
				Topic:      "non-existent-topic",
				Payload:    []byte(`{"id":"test"}`),
				CreatedAt:  fixedTime,
				RetryCount: 0,
				MaxRetries: 3,
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := pstest.NewServer()
			defer server.Close() //nolint:errcheck

			projectID := "test-project"
			subID := tt.event.Topic + "-sub"

			conn, err := grpc.NewClient(server.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			assert.NoError(t, err)
			defer conn.Close() //nolint:errcheck

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			client, err := pubsubV2.NewClient(
				ctx,
				projectID,
				option.WithGRPCConn(conn),
			)
			assert.NoError(t, err)
			defer client.Close() //nolint:errcheck

			// Only create topic and subscription for success cases
			if !tt.expectErr {
				// Create topic
				topicName := "projects/" + projectID + "/topics/" + tt.event.Topic
				topic, err := client.TopicAdminClient.CreateTopic(
					ctx,
					&pubsubpb.Topic{Name: topicName},
				)
				assert.NoError(t, err)

				// Create subscription
				subName := "projects/" + projectID + "/subscriptions/" + subID
				_, err = client.SubscriptionAdminClient.CreateSubscription(
					ctx,
					&pubsubpb.Subscription{
						Name:  subName,
						Topic: topic.GetName(),
					},
				)
				assert.NoError(t, err)
			}

			publisher := NewPubSubEventPublisher(client)

			publishCtx, publishCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer publishCancel()

			err = publisher.PublishEvent(publishCtx, tt.event)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tt.validateMessage(t, client, subID)
			}
		})
	}
}

func TestInitPublisher_Initialize(t *testing.T) {
	init := &InitPublisher{
		Client: &pubsubV2.Client{},
	}

	_, err := init.Initialize(context.Background())
	assert.NoError(t, err)

	res, err := depend.Resolve[domain.TodoEventPublisher]()
	assert.NoError(t, err)
	assert.NotEmpty(t, res)
}
