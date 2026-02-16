package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	pubsubV2 "cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"cloud.google.com/go/pubsub/v2/pstest"
	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
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

// publishMessages sends many payloads to the same Pub/Sub topic.
func publishMessages(ctx context.Context, client *pubsubV2.Client, topicName string, payloads [][]byte) error {
	for _, payload := range payloads {
		result := client.Publisher(topicName).Publish(ctx, &pubsubV2.Message{
			Data: payload,
		})
		_, err := result.Get(ctx) // Wait for the publish result to ensure message is sent
		if err != nil {
			return err
		}
	}
	return nil
}

// run starts the runnable and returns a cancel function and done channel.
func run(
	t *testing.T,
	ctx context.Context,
	subscriber symbiont.Runnable,
) (context.CancelFunc, chan struct{}) {
	t.Helper()

	runCtx, cancel := context.WithCancel(ctx)
	doneChan := make(chan struct{}, 1)

	go func() {
		err := subscriber.Run(runCtx)
		assert.NoError(t, err)
		doneChan <- struct{}{}
	}()

	return cancel, doneChan
}

// waitRunnableStop waits until the runnable goroutine exits.
func waitRunnableStop(t *testing.T, doneChan chan struct{}) {
	t.Helper()

	select {
	case <-doneChan:
	case <-time.After(1 * time.Second):
		t.Fatal("runnable did not shut down in time")
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

// chatEventPayload marshals a ChatMessageEvent into JSON bytes for Pub/Sub publishing.
func chatEventPayload(t *testing.T, event domain.ChatMessageEvent) []byte {
	t.Helper()
	data, err := json.Marshal(event)
	assert.NoError(t, err)
	return data
}

// chatEventKey generates a deterministic key to assert expected summary event parameters.
func chatEventKey(event domain.ChatMessageEvent) string {
	return fmt.Sprintf(
		"%s|%s|%s|%s",
		event.ConversationID,
		event.ChatMessageID.String(),
		event.ChatRole,
		event.Type,
	)
}
