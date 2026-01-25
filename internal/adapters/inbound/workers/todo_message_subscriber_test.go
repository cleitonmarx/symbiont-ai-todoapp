package workers

import (
	"context"
	"log"
	"testing"
	"time"

	pubsubV2 "cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"cloud.google.com/go/pubsub/v2/pstest"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases/mocks"
	"github.com/stretchr/testify/assert"
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
func waitForBatchSignals(t *testing.T, signalChan chan struct{}, expectedBatches int, timeout time.Duration) {
	batchesProcessed := 0
	timeoutChan := time.After(timeout)

	for batchesProcessed < expectedBatches {
		select {
		case <-signalChan:
			batchesProcessed++
		case <-timeoutChan:
			t.Fatalf("timeout waiting for batch processing; got %d batches, expected %d", batchesProcessed, expectedBatches)
		}
	}
}

// TestTodoEventSubscriber_Run verifies that the TodoEventSubscriber correctly batches
// messages from Pub/Sub and triggers the GenerateBoardSummary use case when batch is full
// or interval expires.
func TestTodoEventSubscriber_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, topicName := setupPubSubServer(t, ctx, "test-topic", "test-subscription")

	// Setup mocks
	gbs := mocks.NewMockGenerateBoardSummary(t)
	gbs.EXPECT().Execute(ctx).Return(nil).Once()
	gbs.EXPECT().Execute(ctx).Return(assert.AnError).Once()

	signalChan := make(chan struct{})

	subscriber := TodoEventSubscriber{
		Logger:               log.Default(),
		Client:               client,
		Interval:             50 * time.Millisecond,
		BatchSize:            5,
		SubscriptionID:       "test-subscription",
		GenerateBoardSummary: gbs,
		workerExecutionChan:  signalChan,
	}

	// Run subscriber in background
	go func() {
		err := subscriber.Run(ctx)
		assert.NoError(t, err)
	}()

	// Publish 20 messages (4 batches of 5)
	publishMessages(ctx, client, topicName, 20)

	// Wait for 2 batch processing signals
	waitForBatchSignals(t, signalChan, 2, 5*time.Second)

	gbs.AssertExpectations(t)
	cancel()
}

// TestTodoEventSubscriber_BatchFlush verifies that messages are processed even when
// batch size is not reached if the interval timer expires.
func TestTodoEventSubscriber_BatchFlush(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, topicName := setupPubSubServer(t, ctx, "test-topic", "test-subscription")

	// Mock should be called exactly once (timer-based flush)
	gbs := mocks.NewMockGenerateBoardSummary(t)
	gbs.EXPECT().Execute(ctx).Return(nil).Once()

	signalChan := make(chan struct{})

	subscriber := TodoEventSubscriber{
		Logger:               log.Default(),
		Client:               client,
		Interval:             100 * time.Millisecond,
		BatchSize:            10,
		SubscriptionID:       "test-subscription",
		GenerateBoardSummary: gbs,
		workerExecutionChan:  signalChan,
	}

	go func() {
		err := subscriber.Run(ctx)
		assert.NoError(t, err)
	}()

	// Publish only 3 messages (less than BatchSize of 10)
	publishMessages(ctx, client, topicName, 3)

	// Wait for 1 signal (timer-based flush)
	waitForBatchSignals(t, signalChan, 1, 2*time.Second)

	gbs.AssertExpectations(t)
	cancel()
}
