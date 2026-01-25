package workers

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
)

// TodoEventSubscriber consumes Todo domain events from Pub/Sub
// and triggers AI summary generation.
type TodoEventSubscriber struct {
	Logger               *log.Logger                   `resolve:""`
	Client               *pubsub.Client                `resolve:""`
	Interval             time.Duration                 `config:"SUMMARY_BATCH_INTERVAL" default:"3s"`
	BatchSize            int                           `config:"SUMMARY_BATCH_SIZE" default:"20"`
	SubscriptionID       string                        `config:"PUBSUB_SUBSCRIPTION_ID"`
	GenerateBoardSummary usecases.GenerateBoardSummary `resolve:""`
	workerExecutionChan  chan struct{}
}

// Run starts the subscriber worker.
func (s TodoEventSubscriber) Run(ctx context.Context) error {
	s.Logger.Println("TodoEventSubscriber: running...")

	eventCh := make(chan *pubsub.Message, s.BatchSize*2)
	subscriberInitErrCh := make(chan error, 1)

	// 1. Receive messages in background (blocking call)
	go func() {
		err := s.Client.Subscriber(s.SubscriptionID).Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			select {
			case eventCh <- msg:
				// Ack later, after batching
			case <-ctx.Done():
				msg.Nack()
			}
		})

		if err != nil {
			subscriberInitErrCh <- err
		}
	}()

	// 2. Batch + flush loop
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	var batch []*pubsub.Message

	for {
		select {
		case <-ctx.Done():
			s.Logger.Println("TodoEventSubscriber: stopping...")
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

func (s TodoEventSubscriber) flush(ctx context.Context, batch []*pubsub.Message) {
	s.Logger.Printf("TodoEventSubscriber: processing batch size=%d", len(batch))

	if s.workerExecutionChan != nil {
		s.workerExecutionChan <- struct{}{}
	}

	// Generate board-level summary once per batch
	if err := s.GenerateBoardSummary.Execute(ctx); err != nil {
		s.Logger.Printf("AI summary generation failed: %v", err)
		return
	}

	// Ack messages only after successful enqueue/processing
	for _, msg := range batch {
		msg.Ack()
	}
}
