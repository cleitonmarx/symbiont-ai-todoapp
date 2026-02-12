package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	pubsubV2 "cloud.google.com/go/pubsub/v2"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// publishRawMessage sends a single payload to the given Pub/Sub topic.
func publishRawMessage(ctx context.Context, client *pubsubV2.Client, topicName string, payload []byte) error {
	result := client.Publisher(topicName).Publish(ctx, &pubsubV2.Message{
		Data: payload,
	})
	_, err := result.Get(ctx)
	return err
}

// publishRawMessages sends many payloads to the same Pub/Sub topic.
func publishRawMessages(ctx context.Context, client *pubsubV2.Client, topicName string, payloads [][]byte) error {
	for _, payload := range payloads {
		if err := publishRawMessage(ctx, client, topicName, payload); err != nil {
			return err
		}
	}
	return nil
}

// chatMessageEventPayload marshals a chat message event payload for tests.
func chatMessageEventPayload(t *testing.T, event domain.ChatMessageEvent) []byte {
	t.Helper()

	data, err := json.Marshal(event)
	assert.NoError(t, err)
	return data
}

// runChatSubscriber starts the chat subscriber and returns a cancel function and done channel.
func runChatSubscriber(
	t *testing.T,
	ctx context.Context,
	subscriber ChatEventSubscriber,
) (context.CancelFunc, chan struct{}) {
	t.Helper()

	runCtx, cancel := context.WithCancel(ctx)
	doneChan := make(chan struct{}, 1)

	go func() {
		err := subscriber.Run(runCtx)
		assert.NoError(t, err)
		doneChan <- struct{}{}
	}()

	return cancel, doneChan
}

// waitChatSubscriberStop waits until the subscriber goroutine exits.
func waitChatSubscriberStop(t *testing.T, doneChan chan struct{}) {
	t.Helper()

	select {
	case <-doneChan:
	case <-time.After(1 * time.Second):
		t.Fatal("chat subscriber did not shut down in time")
	}
}

// TestChatEventSubscriber_Run verifies event decoding, coalescing and summary trigger behavior.
func TestChatEventSubscriber_Run(t *testing.T) {
	firstMessageID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	secondMessageID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	thirdMessageID := uuid.MustParse("323e4567-e89b-12d3-a456-426614174002")

	tests := map[string]struct {
		payloads       [][]byte
		expectedEvents []domain.ChatMessageEvent
	}{
		"coalesces-events-per-conversation": {
			payloads: [][]byte{
				chatMessageEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  firstMessageID,
					ConversationID: "global",
				}),
				chatMessageEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  secondMessageID,
					ConversationID: "global",
				}),
				chatMessageEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  thirdMessageID,
					ConversationID: "global",
				}),
			},
			expectedEvents: []domain.ChatMessageEvent{
				{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  thirdMessageID,
					ConversationID: "global",
				},
			},
		},
		"calls-summary-once-per-conversation": {
			payloads: [][]byte{
				chatMessageEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_User,
					ChatMessageID:  firstMessageID,
					ConversationID: "conversation-a",
				}),
				chatMessageEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  secondMessageID,
					ConversationID: "conversation-b",
				}),
				chatMessageEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  thirdMessageID,
					ConversationID: "conversation-a",
				}),
			},
			expectedEvents: []domain.ChatMessageEvent{
				{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  thirdMessageID,
					ConversationID: "conversation-a",
				},
				{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  secondMessageID,
					ConversationID: "conversation-b",
				},
			},
		},
		"invalid-payload": {
			payloads: [][]byte{
				[]byte(`{"type"`),
			},
			expectedEvents: nil,
		},
		"ignore-unrelated-event-type": {
			payloads: [][]byte{
				chatMessageEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_TODO_CREATED,
					ChatMessageID:  firstMessageID,
					ConversationID: "global",
				}),
			},
			expectedEvents: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			subscriptionID := "chat-subscription-" + name
			client, topicName := setupPubSubServer(
				t,
				ctx,
				"chat-topic-"+name,
				subscriptionID,
			)

			receivedEvents := make([]domain.ChatMessageEvent, 0, len(tt.expectedEvents))
			gcs := usecases.NewMockGenerateChatSummary(t)
			for range tt.expectedEvents {
				gcs.EXPECT().
					Execute(mock.Anything, mock.Anything).
					Run(func(_ context.Context, event domain.ChatMessageEvent) {
						receivedEvents = append(receivedEvents, event)
					}).
					Return(nil).
					Once()
			}

			signalChan := make(chan struct{}, 10)
			subscriber := ChatEventSubscriber{
				Logger:              log.Default(),
				Client:              client,
				Interval:            5 * time.Second,
				BatchSize:           max(1, len(tt.payloads)),
				SubscriptionID:      subscriptionID,
				GenerateChatSummary: gcs,
				workerExecutionChan: signalChan,
			}

			cancel, doneChan := runChatSubscriber(t, ctx, subscriber)
			err := publishRawMessages(ctx, client, topicName, tt.payloads)
			assert.NoError(t, err)

			waitForBatchSignals(t, signalChan, 1, 1*time.Second)
			cancel()
			waitChatSubscriberStop(t, doneChan)

			assert.Equal(t, len(tt.expectedEvents), len(receivedEvents))

			expectedIndex := make(map[string]domain.ChatMessageEvent, len(tt.expectedEvents))
			for _, event := range tt.expectedEvents {
				expectedIndex[eventKey(event)] = event
			}

			for _, event := range receivedEvents {
				key := eventKey(event)
				expected, ok := expectedIndex[key]
				assert.True(t, ok, "unexpected event received: %s", key)
				if !ok {
					continue
				}
				assert.Equal(t, expected, event)
			}
		})
	}
}

// eventKey generates a deterministic key to assert expected summary event parameters.
func eventKey(event domain.ChatMessageEvent) string {
	return fmt.Sprintf(
		"%s|%s|%s|%s",
		event.ConversationID,
		event.ChatMessageID.String(),
		event.ChatRole,
		event.Type,
	)
}
