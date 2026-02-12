package graphql

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTodoGraphQLServer_Run(t *testing.T) {
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &TodoGraphQLServer{
		Port:   12345,
		Logger: log.Default(),
	}

	shutdownCh := make(chan error, 1)

	go func() {
		shutdownCh <- server.Run(cancelCtx)
	}()

	for range 10 {
		err := server.IsReady(cancelCtx)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-shutdownCh:
		if err != nil {
			assert.Fail(t, "server exited with error: %v", err)
		}
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		assert.Fail(t, "server did not shut down in time")

	}
}
