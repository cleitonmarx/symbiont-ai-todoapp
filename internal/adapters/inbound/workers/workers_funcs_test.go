package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	pubsubV2 "cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"cloud.google.com/go/pubsub/v2/pstest"
	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const testPubSubProjectID = "test-project"

var (
	pubSubTestOnce   sync.Once
	pubSubTestServer *pstest.Server
	pubSubTestConn   *grpc.ClientConn
	pubSubTestClient *pubsubV2.Client
	pubSubTestErr    error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if pubSubTestClient != nil {
		pubSubTestClient.Close() //nolint:errcheck
	}
	if pubSubTestConn != nil {
		pubSubTestConn.Close() //nolint:errcheck
	}
	if pubSubTestServer != nil {
		pubSubTestServer.Close() //nolint:errcheck
	}
	os.Exit(code)
}

// setupPubSubServer creates a pstest server with topic and subscription.
func setupPubSubServer(t *testing.T, ctx context.Context, topicID, subscriptionID string) (*pubsubV2.Client, string) {
	pubSubTestOnce.Do(func() {
		pubSubTestServer = pstest.NewServer()
		pubSubTestConn, pubSubTestErr = grpc.NewClient(
			pubSubTestServer.Addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if pubSubTestErr != nil {
			return
		}
		pubSubTestClient, pubSubTestErr = pubsubV2.NewClient(
			ctx,
			testPubSubProjectID,
			option.WithGRPCConn(pubSubTestConn),
		)
	})
	assert.NoError(t, pubSubTestErr)

	// Create topic
	topicName := "projects/" + testPubSubProjectID + "/topics/" + topicID
	_, err := pubSubTestClient.TopicAdminClient.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName})
	if err != nil {
		_, err = pubSubTestClient.TopicAdminClient.GetTopic(ctx, &pubsubpb.GetTopicRequest{Topic: topicName})
	}
	assert.NoError(t, err)

	// Create subscription
	subName := "projects/" + testPubSubProjectID + "/subscriptions/" + subscriptionID
	_, err = pubSubTestClient.SubscriptionAdminClient.CreateSubscription(
		ctx,
		&pubsubpb.Subscription{
			Name:  subName,
			Topic: topicName,
		},
	)
	if err != nil {
		_, err = pubSubTestClient.SubscriptionAdminClient.GetSubscription(
			ctx,
			&pubsubpb.GetSubscriptionRequest{Subscription: subName},
		)
	}
	assert.NoError(t, err)

	return pubSubTestClient, topicName
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
func chatEventPayload(t *testing.T, event outbox.ChatMessageEvent) []byte {
	t.Helper()
	data, err := json.Marshal(event)
	assert.NoError(t, err)
	return data
}

// chatEventKey generates a deterministic key to assert expected summary event parameters.
func chatEventKey(event outbox.ChatMessageEvent) string {
	return fmt.Sprintf(
		"%s|%s|%s|%s",
		event.ConversationID,
		event.ChatMessageID.String(),
		event.ChatRole,
		event.Type,
	)
}
