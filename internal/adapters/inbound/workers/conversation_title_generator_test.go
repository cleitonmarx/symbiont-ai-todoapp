package workers

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/chat"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestConversationTitleGenerator_Run(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	firstMessageID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	secondMessageID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	thirdMessageID := uuid.MustParse("323e4567-e89b-12d3-a456-426614174002")

	tests := map[string]struct {
		payloads       [][]byte
		expectedEvents []outbox.ChatMessageEvent
	}{
		"coalesces-events-per-conversation": {
			payloads: [][]byte{
				chatEventPayload(t, outbox.ChatMessageEvent{
					Type:           outbox.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       assistant.ChatRole_User,
					ChatMessageID:  firstMessageID,
					ConversationID: conversationID,
				}),
				chatEventPayload(t, outbox.ChatMessageEvent{
					Type:           outbox.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       assistant.ChatRole_Assistant,
					ChatMessageID:  secondMessageID,
					ConversationID: conversationID,
				}),
				chatEventPayload(t, outbox.ChatMessageEvent{
					Type:           outbox.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       assistant.ChatRole_Assistant,
					ChatMessageID:  thirdMessageID,
					ConversationID: conversationID,
				}),
			},
			expectedEvents: []outbox.ChatMessageEvent{
				{
					Type:           outbox.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       assistant.ChatRole_Assistant,
					ChatMessageID:  thirdMessageID,
					ConversationID: conversationID,
				},
			},
		},
		"calls-title-generator-once-per-conversation": {
			payloads: [][]byte{
				chatEventPayload(t, outbox.ChatMessageEvent{
					Type:           outbox.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       assistant.ChatRole_Assistant,
					ChatMessageID:  firstMessageID,
					ConversationID: conversationID,
				}),
				chatEventPayload(t, outbox.ChatMessageEvent{
					Type:           outbox.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       assistant.ChatRole_Assistant,
					ChatMessageID:  secondMessageID,
					ConversationID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				}),
				chatEventPayload(t, outbox.ChatMessageEvent{
					Type:           outbox.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       assistant.ChatRole_Assistant,
					ChatMessageID:  thirdMessageID,
					ConversationID: conversationID,
				}),
			},
			expectedEvents: []outbox.ChatMessageEvent{
				{
					Type:           outbox.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       assistant.ChatRole_Assistant,
					ChatMessageID:  thirdMessageID,
					ConversationID: conversationID,
				},
				{
					Type:           outbox.EventType_CHAT_MESSAGE_SENT,
					ChatRole:       assistant.ChatRole_Assistant,
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
				chatEventPayload(t, outbox.ChatMessageEvent{
					Type:           outbox.EventType_TODO_CREATED,
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
			subscriptionID := "chat-title-subscription-" + name
			client, topicName := setupPubSubServer(
				t,
				ctx,
				"chat-title-topic-"+name,
				subscriptionID,
			)

			receivedEvents := make([]outbox.ChatMessageEvent, 0, len(tt.expectedEvents))
			gct := chat.NewMockGenerateConversationTitle(t)
			for range tt.expectedEvents {
				gct.EXPECT().Execute(mock.Anything, mock.Anything).
					Run(func(ctx context.Context, event outbox.ChatMessageEvent) {
						receivedEvents = append(receivedEvents, event)
					}).
					Return(nil).
					Once()
			}

			signalChan := make(chan struct{}, 10)
			cancel, doneChan := run(t, ctx, ConversationTitleGenerator{
				Logger:                    log.Default(),
				Client:                    client,
				Interval:                  5 * time.Second,
				BatchSize:                 len(tt.payloads),
				SubscriptionID:            subscriptionID,
				GenerateConversationTitle: gct,
				workerExecutionChan:       signalChan,
			})

			err := publishMessages(ctx, client, topicName, tt.payloads)
			assert.NoError(t, err)

			waitForBatchSignals(t, signalChan, 1, 1*time.Second)
			cancel()
			waitRunnableStop(t, doneChan)

			assert.Equal(t, len(tt.expectedEvents), len(receivedEvents))

			expectedIndex := make(map[string]outbox.ChatMessageEvent, len(tt.expectedEvents))
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
