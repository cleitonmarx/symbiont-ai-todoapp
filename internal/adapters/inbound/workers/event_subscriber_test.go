package workers

import (
	"context"
	"log"
	"testing"
	"time"

	pubsubV2 "cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"cloud.google.com/go/pubsub/v2/pstest"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// setupPubSubServer creates a pstest server with topic and subscription.
func setupPubSubServer(t *testing.T, ctx context.Context, topicID, subscriptionID string) (*pubsubV2.Client, string) {
	server := pstest.NewServer()
	t.Cleanup(func() {
		server.Close() //nolint:errcheck
	})

	conn, err := grpc.NewClient(server.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	t.Cleanup(func() {
		conn.Close() //nolint:errcheck
	})

	projectID := "test-project"
	client, err := pubsubV2.NewClient(ctx, projectID, option.WithGRPCConn(conn))
	assert.NoError(t, err)
	t.Cleanup(func() {
		client.Close() //nolint:errcheck
	})

	// Create topic
	topicName := "projects/" + projectID + "/topics/" + topicID
	topic, err := client.TopicAdminClient.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName})
	assert.NoError(t, err)

	// Create subscription
	subName := "projects/" + projectID + "/subscriptions/" + subscriptionID
	_, err = client.SubscriptionAdminClient.CreateSubscription(
		ctx,
		&pubsubpb.Subscription{
			Name:  subName,
			Topic: topic.GetName(),
		},
	)
	assert.NoError(t, err)

	return client, topicName
}

// publishMessages sends count messages to the given topic.
func publishMessages(ctx context.Context, client *pubsubV2.Client, topicName string, count int) {
	for range count {
		client.Publisher(topicName).Publish(ctx, &pubsubV2.Message{
			Data: []byte("test message"),
		}) //nolint:errcheck

	}
}

// waitForBatchSignals waits for the expected number of batch processing signals or timeout.
func waitForBatchSignals(t *testing.T, signalChan chan struct{}, expectedBatches int, timeout time.Duration) int {
	batchesProcessed := 0

	for batchesProcessed < expectedBatches {
		select {
		case <-signalChan:
			batchesProcessed++
		case <-time.After(timeout):
			t.Fatalf("timeout waiting for batch processing; got %d batches, expected %d", batchesProcessed, expectedBatches)
		}
	}
	return batchesProcessed
}

// TestTodoEventSubscriber_Run verifies that the TodoEventSubscriber correctly batches
// messages from Pub/Sub and triggers the GenerateBoardSummary use case when batch is full
// or interval expires.
func TestTodoEventSubscriber_Run(t *testing.T) {
	tests := map[string]struct {
		batchSize       int
		interval        time.Duration
		publishCount    int
		expectedBatches int
		setExpectations func(*usecases.MockGenerateBoardSummary)
	}{
		"batch-full-triggers-processing": {
			batchSize:       5,
			interval:        50 * time.Millisecond,
			publishCount:    20,
			expectedBatches: 2,
			setExpectations: func(gbs *usecases.MockGenerateBoardSummary) {
				gbs.EXPECT().Execute(mock.Anything).Return(nil)
				gbs.EXPECT().Execute(mock.Anything).Return(assert.AnError)
			},
		},
		"interval-flush-triggers-processing": {
			batchSize:       10,
			interval:        100 * time.Millisecond,
			publishCount:    3,
			expectedBatches: 1,
			setExpectations: func(gbs *usecases.MockGenerateBoardSummary) {
				gbs.EXPECT().Execute(mock.Anything).Return(nil)
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
			doneChan := make(chan struct{})

			subscriber := TodoEventSubscriber{
				Logger:               log.Default(),
				Client:               client,
				Interval:             tt.interval,
				BatchSize:            tt.batchSize,
				SubscriptionID:       "test-subscription-" + name,
				GenerateBoardSummary: gbs,
				workerExecutionChan:  signalChan,
			}

			go func() {
				err := subscriber.Run(ctx)
				assert.NoError(t, err)
				doneChan <- struct{}{}

			}()

			publishMessages(ctx, client, topicName, tt.publishCount)
			got := waitForBatchSignals(t, signalChan, tt.expectedBatches, 1*time.Second)
			assert.Equal(t, tt.expectedBatches, got)

			cancel()

			select {
			case <-doneChan:
				// success
			case <-time.After(1 * time.Second):
				t.Fatal("subscriber did not shut down in time")
			}
		})
	}

}
