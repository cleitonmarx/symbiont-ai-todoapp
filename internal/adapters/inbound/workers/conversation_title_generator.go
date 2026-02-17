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
	"github.com/google/uuid"
)

// ConversationTitleGenerator is a runnable that consumes chat-message events and asynchronously generates conversation titles.
type ConversationTitleGenerator struct {
	Logger                    *log.Logger                        `resolve:""`
	Client                    *pubsub.Client                     `resolve:""`
	GenerateConversationTitle usecases.GenerateConversationTitle `resolve:""`
	Interval                  time.Duration                      `config:"CHAT_TITLE_BATCH_INTERVAL" default:"3s"`
	BatchSize                 int                                `config:"CHAT_TITLE_BATCH_SIZE" default:"50"`
	SubscriptionID            string                             `config:"CHAT_TITLE_EVENTS_SUBSCRIPTION_ID"`
	workerExecutionChan       chan struct{}
}

// Run starts the conversation title generator worker.
func (s ConversationTitleGenerator) Run(ctx context.Context) error {
	s.Logger.Println("ConversationTitleGenerator: running...")

	if s.BatchSize <= 0 {
		s.BatchSize = 50
	}
	if s.Interval <= 0 {
		s.Interval = 3 * time.Second
	}

	eventCh := make(chan *pubsub.Message, s.BatchSize*2)
	subscriberInitErrCh := make(chan error, 1)

	go func() {
		err := s.Client.Subscriber(s.SubscriptionID).Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			select {
			case eventCh <- msg:
			case <-ctx.Done():
				msg.Nack()
			}
		})

		if err != nil {
			subscriberInitErrCh <- err
		}
	}()

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	var batch []*pubsub.Message

	for {
		select {
		case <-ctx.Done():
			s.Logger.Println("ConversationTitleGenerator: stopped")
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

// flush processes a batch of Pub/Sub messages, grouping them by conversation
// and invoking the title generator use case.
func (s ConversationTitleGenerator) flush(ctx context.Context, batch []*pubsub.Message) {
	s.Logger.Printf("ConversationTitleGenerator: processing batch size=%d", len(batch))

	if s.workerExecutionChan != nil {
		s.workerExecutionChan <- struct{}{}
	}

	conversations := make(map[uuid.UUID]conversationTitleGeneratorBatch)
	for _, msg := range batch {
		var event domain.ChatMessageEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			s.Logger.Printf("ConversationTitleGenerator: failed to decode event payload: %v", err)
			msg.Nack()
			continue
		}

		if event.Type != domain.EventType_CHAT_MESSAGE_SENT {
			msg.Ack()
			continue
		}

		conversationBatch, found := conversations[event.ConversationID]
		if !found {
			conversationBatch = conversationTitleGeneratorBatch{}
		}
		conversationBatch.LatestEvent = event
		conversationBatch.Messages = append(conversationBatch.Messages, msg)
		conversations[event.ConversationID] = conversationBatch
	}

	for _, conversationBatch := range conversations {
		err := s.GenerateConversationTitle.Execute(ctx, conversationBatch.LatestEvent)
		if err != nil {
			for _, message := range conversationBatch.Messages {
				message.Nack()
			}
			if !errors.Is(err, context.Canceled) {
				s.Logger.Printf("ConversationTitleGenerator: %v", err)
			}
			continue
		}

		for _, message := range conversationBatch.Messages {
			message.Ack()
		}
	}
}

// conversationTitleGeneratorBatch represents a batch of chat message events for a single conversation,
// along with the latest event for that conversation.
type conversationTitleGeneratorBatch struct {
	LatestEvent domain.ChatMessageEvent
	Messages    []*pubsub.Message
}
