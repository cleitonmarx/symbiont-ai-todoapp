package workers

import (
	"log"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMessageRelay_Run(t *testing.T) {
	md := usecases.NewMockRelayOutbox(t)

	md.EXPECT().Execute(mock.Anything).Return(assert.AnError).Once()
	md.EXPECT().Execute(mock.Anything).Return(nil).Once()

	signalChan := make(chan struct{})

	cancel, doneChan := run(t, t.Context(), MessageRelay{
		MessageDispatcher:   md,
		Logger:              log.Default(),
		Interval:            2 * time.Millisecond,
		workerExecutionChan: signalChan,
	})

	waitForBatchSignals(t, signalChan, 2, 1*time.Second)

	cancel()

	waitRunnableStop(t, doneChan)
}
