package workers

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestChatSummaryGenerator_Run(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	firstMessageID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	secondMessageID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	thirdMessageID := uuid.MustParse("323e4567-e89b-12d3-a456-426614174002")

	tests := map[string]struct {
		payloads       [][]byte
		expectedEvents []domain.ChatMessageEvent
	}{
		"coalesces-events-per-conversation": {
			payloads: [][]byte{
				chatEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  firstMessageID,
					ConversationID: conversationID,
				}),
				chatEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  secondMessageID,
					ConversationID: conversationID,
				}),
				chatEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  thirdMessageID,
					ConversationID: conversationID,
				}),
			},
			expectedEvents: []domain.ChatMessageEvent{
				{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  thirdMessageID,
					ConversationID: conversationID,
				},
			},
		},
		"calls-summary-once-per-conversation": {
			payloads: [][]byte{
				chatEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_User,
					ChatMessageID:  firstMessageID,
					ConversationID: conversationID,
				}),
				chatEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  secondMessageID,
					ConversationID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				}),
				chatEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  thirdMessageID,
					ConversationID: conversationID,
				}),
			},
			expectedEvents: []domain.ChatMessageEvent{
				{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  thirdMessageID,
					ConversationID: conversationID,
				},
				{
					Type:           domain.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       domain.ChatRole_Assistant,
					ChatMessageID:  secondMessageID,
					ConversationID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
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
				chatEventPayload(t, domain.ChatMessageEvent{
					Type:           domain.EventType_TODO_CREATED,
					ChatMessageID:  firstMessageID,
					ConversationID: conversationID,
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

			cancel, doneChan := run(t, ctx, ChatSummaryGenerator{
				Logger:              log.Default(),
				Client:              client,
				Interval:            1 * time.Second,
				BatchSize:           max(1, len(tt.payloads)),
				SubscriptionID:      subscriptionID,
				GenerateChatSummary: gcs,
				workerExecutionChan: signalChan,
			})

			err := publishMessages(ctx, client, topicName, tt.payloads)
			assert.NoError(t, err)

			waitForBatchSignals(t, signalChan, 1, 1*time.Second)

			cancel()

			waitRunnableStop(t, doneChan)

			assert.Equal(t, len(tt.expectedEvents), len(receivedEvents))

			expectedIndex := make(map[string]domain.ChatMessageEvent, len(tt.expectedEvents))
			for _, event := range tt.expectedEvents {
				expectedIndex[chatEventKey(event)] = event
			}

			for _, event := range receivedEvents {
				key := chatEventKey(event)
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
