//--go:build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/app"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
)

func TestTodoApp_Integration(t *testing.T) {
	todoApp := app.NewTodoApp(
		&initEnvVars{
			envVars: map[string]string{
				"VAULT_ADDR":                  "http://localhost:8200",
				"VAULT_TOKEN":                 "root-token",
				"VAULT_MOUNT_PATH":            "secret",
				"VAULT_SECRET_PATH":           "todoapp",
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4318",
				"DB_HOST":                     "localhost",
				"DB_PORT":                     "5432",
				"DB_NAME":                     "todoappdb",
				"EMAIL_SENDER_INTERVAL":       "1s",
				"PUBSUB_EMULATOR_HOST":        "localhost:8681",
				"PUBSUB_PROJECT_ID":           "local-dev",
				"PUBSUB_TOPIC_ID":             "Todo",
				"PUBSUB_SUBSCRIPTION_ID":      "todo_summary_generator",
				"LLM_MODEL_HOST":              "http://localhost:12434",
				"LLM_MODEL":                   "ai/gpt-oss",
				"LLM_EMBEDDING_MODEL":         "ai/qwen3-embedding",
			},
		},
		&InitDockerCompose{},
	)

	summaryQueue := make(usecases.CompletedSummaryQueue, 1)
	depend.Register(summaryQueue)

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownCh := todoApp.RunAsync(cancelCtx)

	err := todoApp.WaitForReadiness(cancelCtx, 10*time.Minute)
	if err != nil {
		cancel()
		t.Fatalf("TodoApp app failed to become ready: %v", err)
	}

	apiCli, err := gen.NewClientWithResponses("http://localhost:8080")
	require.NoError(t, err, "failed to create TodoApp API client")
	t.Run("create-todo", func(t *testing.T) {
		createResp, err := apiCli.CreateTodoWithResponse(cancelCtx, gen.CreateTodoJSONRequestBody{
			Title:   "Integration Test Todo",
			DueDate: types.Date{Time: time.Now().Add(24 * time.Hour)},
		})
		require.NoError(t, err, "failed to call CreateTodo endpoint")
		require.NotNil(t, createResp.JSON201, "expected non-nil response for CreateTodo")

	})

	var todos []gen.Todo
	t.Run("list-created-todo", func(t *testing.T) {
		resp, err := apiCli.ListTodosWithResponse(cancelCtx, &gen.ListTodosParams{
			Page:     1,
			Pagesize: 10,
		})

		require.NoError(t, err, "failed to call ListTodos endpoint")
		require.NotNil(t, resp.JSON200, "expected non-nil response for ListTodos")
		require.Equal(t, 1, len(resp.JSON200.Items), "expected 1 todo in the list")

		todos = resp.JSON200.Items
	})

	t.Run("update-todos", func(t *testing.T) {
		for _, todo := range todos {
			updateResp, err := apiCli.UpdateTodoWithResponse(cancelCtx, todo.Id, gen.UpdateTodoJSONRequestBody{
				Status: common.Ptr(gen.DONE),
			})
			require.NoError(t, err, "failed to call UpdateTodo endpoint")
			require.NotNil(t, updateResp.JSON200, "expected non-nil response for UpdateTodo")
			require.Equal(t, gen.DONE, updateResp.JSON200.Status, "expected todo status to be 'completed'")
		}
	})

	t.Run("check-board-summary-generated", func(t *testing.T) {
		select {
		case summary := <-summaryQueue:
			require.Equal(t, 1, summary.Content.Counts.Done,
				"expected board summary to have at least one done or open todo",
			)
		case <-time.After(5 * time.Minute):
			t.Fatalf("Timed out waiting for board summary in queue")
		}
	})

	t.Run("delete-todos", func(t *testing.T) {
		for _, todo := range todos {
			deleteResp, err := apiCli.DeleteTodoWithResponse(cancelCtx, todo.Id)
			require.NoError(t, err, "failed to call DeleteTodo endpoint")
			require.Equal(t, 204, deleteResp.StatusCode(), "expected 204 No Content response for DeleteTodo")
		}

		// Verify todos are deleted
		listResp, err := apiCli.ListTodosWithResponse(cancelCtx, &gen.ListTodosParams{
			Page:     1,
			Pagesize: 10,
		})
		require.NoError(t, err, "failed to call ListTodos endpoint after deletions")
		require.NotNil(t, listResp.JSON200, "expected non-nil response for ListTodos after deletions")
		require.Equal(t, 0, len(listResp.JSON200.Items), "expected 0 todos in the list after deletions")
	})

	// Shutdown the app
	cancel()

	select {
	case <-time.After(1 * time.Minute):
		t.Fatalf("TodoApp app did not shut down in time")
	case err = <-shutdownCh:
		if err != nil {
			t.Fatalf("TodoApp app shutdown with error: %v", err)
		} else {
			t.Logf("TodoApp app shut down gracefully")
		}
	}
}

type initEnvVars struct {
	envVars map[string]string
}

func (i *initEnvVars) Initialize(ctx context.Context) (context.Context, error) {
	for key, value := range i.envVars {
		os.Setenv(key, value) //nolint:errcheck
	}
	return ctx, nil
}

func (i *initEnvVars) Close() {
	for key := range i.envVars {
		os.Unsetenv(key) //nolint:errcheck
	}
}
