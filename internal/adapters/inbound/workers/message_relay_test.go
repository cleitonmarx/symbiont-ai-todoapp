package workers

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMessageRelay_Run(t *testing.T) {
	md := mocks.NewMockRelayOutbox(t)

	md.EXPECT().Execute(mock.Anything).Return(assert.AnError).Once()
	md.EXPECT().Execute(mock.Anything).Return(nil).Once()

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan struct{})

	mr := MessageRelay{
		MessageDispatcher:   md,
		Logger:              log.Default(),
		Interval:            2 * time.Millisecond,
		workerExecutionChan: signalChan,
	}

	go func() {
		err := mr.Run(cancelCtx)
		assert.NoError(t, err)
	}()

	for range 2 {
		select {
		case <-signalChan:
			// Received signal that a batch was processed
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for message relay to process batch")
		}
	}

	cancel()
}
