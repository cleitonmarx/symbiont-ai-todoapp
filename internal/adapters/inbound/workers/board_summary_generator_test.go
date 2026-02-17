package workers

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBoardSummaryGenerator_Run(t *testing.T) {
	tests := map[string]struct {
		batchSize       int
		interval        time.Duration
		publishCount    int
		expectedBatches int
		setExpectations func(*usecases.MockGenerateBoardSummary)
	}{
		"batch-full-triggers-processing": {
			batchSize:       5,
			interval:        300 * time.Millisecond,
			publishCount:    20,
			expectedBatches: 4,
			setExpectations: func(gbs *usecases.MockGenerateBoardSummary) {
				gbs.EXPECT().Execute(mock.Anything).Return(nil).Times(4)
			},
		},
		"interval-flush-triggers-processing": {
			batchSize:       10,
			interval:        100 * time.Millisecond,
			publishCount:    3,
			expectedBatches: 1,
			setExpectations: func(gbs *usecases.MockGenerateBoardSummary) {
				gbs.EXPECT().Execute(mock.Anything).Return(nil).Once()
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			client, topicName := setupPubSubServer(t, ctx, "test-topic-"+name, "test-subscription-"+name)

			gbs := usecases.NewMockGenerateBoardSummary(t)
			if tt.setExpectations != nil {
				tt.setExpectations(gbs)
			}

			signalChan := make(chan struct{})
			cancel, doneChan := run(t, ctx, BoardSummaryGenerator{
				Logger:               log.Default(),
				Client:               client,
				Interval:             tt.interval,
				BatchSize:            tt.batchSize,
				SubscriptionID:       "test-subscription-" + name,
				GenerateBoardSummary: gbs,
				workerExecutionChan:  signalChan,
			})

			var payloads [][]byte
			for range tt.publishCount {
				payloads = append(payloads, []byte("test message "))
			}
			err := publishMessages(ctx, client, topicName, payloads)
			assert.NoError(t, err)

			got := waitForBatchSignals(t, signalChan, tt.expectedBatches, 10*time.Second)
			assert.Equal(t, tt.expectedBatches, got)

			cancel()

			waitRunnableStop(t, doneChan)
		})
	}
}
