package pubsub

import (
	"context"
	"testing"

	pubsubV2 "cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/pstest"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestInitClient_Initialize(t *testing.T) {
	server := pstest.NewServer()
	defer server.Close() //nolint:errcheck

	conn, err := grpc.NewClient(server.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	ctx := context.Background()
	client, err := pubsubV2.NewClient(
		ctx,
		"test-project",
		option.WithGRPCConn(conn),
	)
	assert.NoError(t, err)

	init := &InitClient{
		client: client,
	}

	_, err = init.Initialize(ctx)
	assert.NoError(t, err)

	_, err = depend.Resolve[*pubsubV2.Client]()
	assert.NoError(t, err)

	init.Close()
}
