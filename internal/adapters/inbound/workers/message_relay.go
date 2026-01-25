package workers

import (
	"context"
	"log"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
)

// MessageRelay is a runnable that processes outbox events and publishes them to Pub/Sub.
type MessageRelay struct {
	MessageDispatcher   usecases.RelayOutbox `resolve:""`
	Logger              *log.Logger          `resolve:""`
	Interval            time.Duration        `config:"FETCH_OUTBOX_INTERVAL" default:"500ms"`
	workerExecutionChan chan struct{}
}

// Run starts the periodic processing of outbox events.
func (op MessageRelay) Run(ctx context.Context) error {
	op.Logger.Println("MessageRelay: running...")
	ticker := time.NewTicker(op.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := op.MessageDispatcher.Execute(ctx)
			if err != nil {
				op.Logger.Printf("error processing batch: %v", err)
			}
			if op.workerExecutionChan != nil {
				op.workerExecutionChan <- struct{}{}
			}
		case <-ctx.Done():
			op.Logger.Println("MessageRelay: stopping...")
			return nil
		}
	}
}
