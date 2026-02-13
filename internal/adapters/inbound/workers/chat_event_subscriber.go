package workers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
)

// ChatEventSubscriber consumes chat-message events from Pub/Sub
// and triggers conversation summary generation.
type ChatEventSubscriber struct {
	Logger              *log.Logger                  `resolve:""`
	Client              *pubsub.Client               `resolve:""`
	Interval            time.Duration                `config:"CHAT_SUMMARY_BATCH_INTERVAL" default:"3s"`
	BatchSize           int                          `config:"CHAT_SUMMARY_BATCH_SIZE" default:"50"`
	SubscriptionID      string                       `config:"CHAT_EVENTS_SUBSCRIPTION_ID"`
	GenerateChatSummary usecases.GenerateChatSummary `resolve:""`
	workerExecutionChan chan struct{}
}

// Run starts the chat event subscriber worker.
func (s ChatEventSubscriber) Run(ctx context.Context) error {
	s.Logger.Println("ChatEventSubscriber: running...")

	if s.BatchSize <= 0 {
		s.BatchSize = 50
	}
	if s.Interval <= 0 {
		s.Interval = 3 * time.Second
	}

	eventCh := make(chan *pubsub.Message, s.BatchSize*2)
	subscriberInitErrCh := make(chan error, 1)

	// 1. Receive messages in background (blocking call).
	go func() {
		err := s.Client.Subscriber(s.SubscriptionID).Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			select {
			case eventCh <- msg:
				// Ack later, after batching.
			case <-ctx.Done():
				msg.Nack()
			}
		})

		if err != nil {
			subscriberInitErrCh <- err
		}
	}()

	// 2. Batch + flush loop.
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	var batch []*pubsub.Message

	for {
		select {
		case <-ctx.Done():
			s.Logger.Println("ChatEventSubscriber: stopped")
			return nil

		case err := <-subscriberInitErrCh:
			return err

		case msg := <-eventCh:
			batch = append(batch, msg)
			if len(batch) >= s.BatchSize {
				s.flush(ctx, batch)
				batch = nil
			}

		case <-ticker.C:
			if len(batch) > 0 {
				s.flush(ctx, batch)
				batch = nil
			}
		}
	}
}

// chatSummaryConversationBatch groups chat-message events by conversation.
// It keeps all Pub/Sub messages for ack/nack handling and the latest chat event
// to avoid triggering summary generation multiple times for the same conversation.
type chatSummaryConversationBatch struct {
	LatestEvent domain.ChatMessageEvent
	Messages    []*pubsub.Message
}

// flush processes one batch of Pub/Sub messages.
func (s ChatEventSubscriber) flush(ctx context.Context, batch []*pubsub.Message) {
	s.Logger.Printf("ChatEventSubscriber: processing batch size=%d", len(batch))

	if s.workerExecutionChan != nil {
		s.workerExecutionChan <- struct{}{}
	}

	conversations := make(map[string]chatSummaryConversationBatch)
	for _, msg := range batch {
		var event domain.ChatMessageEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			s.Logger.Printf("ChatEventSubscriber: failed to decode event payload: %v", err)
			msg.Nack()
			continue
		}

		// Ignore unrelated events that may be delivered to this subscription.
		if event.Type != domain.EventType_CHAT_MESSAGE_SENT {
			msg.Ack()
			continue
		}

		conversationBatch, found := conversations[event.ConversationID]
		if !found {
			conversationBatch = chatSummaryConversationBatch{}
		}
		conversationBatch.LatestEvent = event
		conversationBatch.Messages = append(conversationBatch.Messages, msg)
		conversations[event.ConversationID] = conversationBatch
	}

	for _, conversationBatch := range conversations {
		err := s.GenerateChatSummary.Execute(ctx, conversationBatch.LatestEvent)
		if err != nil {
			for _, message := range conversationBatch.Messages {
				message.Nack()
			}
			if !errors.Is(err, context.Canceled) {
				s.Logger.Printf("ChatEventSubscriber: %v", err)
			}
			continue
		}

		for _, message := range conversationBatch.Messages {
			message.Ack()
		}
	}
}
