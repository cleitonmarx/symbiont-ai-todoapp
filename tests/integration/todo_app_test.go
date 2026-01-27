//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/app"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
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
			},
		},
		&InitDockerCompose{},
	)

	summaryQueue := make(usecases.CompletedSummaryQueue, 5)
	depend.Register(summaryQueue)

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownCh := todoApp.RunAsync(cancelCtx)

	err := todoApp.WaitForReadiness(cancelCtx, 1*time.Minute)
	if err != nil {
		cancel()
		t.Fatalf("TodoApp app failed to become ready: %v", err)
	}

	apiCli, err := gen.NewClientWithResponses("http://localhost:8080")
	assert.NoError(t, err, "failed to create TodoApp API client")
	t.Run("create-todos", func(t *testing.T) {
		for i := range 5 {
			createResp, err := apiCli.CreateTodoWithResponse(cancelCtx, gen.CreateTodoJSONRequestBody{
				Title:   fmt.Sprintf("Test Todo %d", i+1),
				DueDate: types.Date{Time: time.Now().Add(24 * time.Hour)},
			})
			assert.NoError(t, err, "failed to call CreateTodo endpoint")
			assert.NotNil(t, createResp.JSON201, "expected non-nil response for CreateTodo")
		}
	})

	var todos []gen.Todo
	t.Run("list-created-todos", func(t *testing.T) {
		resp, err := apiCli.ListTodosWithResponse(cancelCtx, &gen.ListTodosParams{
			Page:     1,
			Pagesize: 10,
		})

		assert.NoError(t, err, "failed to call ListTodos endpoint")
		assert.NotNil(t, resp.JSON200, "expected non-nil response for ListTodos")
		assert.Equal(t, 5, len(resp.JSON200.Items), "expected 5 todos in the list")

		todos = resp.JSON200.Items
	})

	t.Run("update-todos-and-check-emails", func(t *testing.T) {
		statusDone := gen.DONE
		for _, todo := range todos {
			updateResp, err := apiCli.UpdateTodoWithResponse(cancelCtx, todo.Id, gen.UpdateTodoJSONRequestBody{
				Status: &statusDone,
			})
			assert.NoError(t, err, "failed to call UpdateTodo endpoint")
			assert.NotNil(t, updateResp.JSON200, "expected non-nil response for UpdateTodo")
			assert.Equal(t, gen.DONE, updateResp.JSON200.Status, "expected todo status to be 'completed'")
		}
	})

	t.Run("check-board-summary-generated", func(t *testing.T) {
		select {
		case summary := <-summaryQueue:
			assert.True(
				t,
				summary.Content.Counts.Done >= 1 ||
					summary.Content.Counts.Open >= 1,
				"expected board summary to have at least one done or open todo",
			)
		case <-time.After(20 * time.Second):
			t.Fatalf("Timed out waiting for board summary in queue")
		}
	})

	t.Run("delete-todos", func(t *testing.T) {
		for _, todo := range todos {
			deleteResp, err := apiCli.DeleteTodoWithResponse(cancelCtx, todo.Id)
			assert.NoError(t, err, "failed to call DeleteTodo endpoint")
			assert.Equal(t, 204, deleteResp.StatusCode(), "expected 204 No Content response for DeleteTodo")
		}

		// Verify todos are deleted
		listResp, err := apiCli.ListTodosWithResponse(cancelCtx, &gen.ListTodosParams{
			Page:     1,
			Pagesize: 10,
		})
		assert.NoError(t, err, "failed to call ListTodos endpoint after deletions")
		assert.NotNil(t, listResp.JSON200, "expected non-nil response for ListTodos after deletions")
		assert.Equal(t, 0, len(listResp.JSON200.Items), "expected 0 todos in the list after deletions")
	})

	// Shutdown the app
	cancel()

	select {
	case <-time.After(10 * time.Second):
		t.Fatalf("TodoMailer app did not shut down in time")
	case err = <-shutdownCh:
		if err != nil {
			t.Fatalf("TodoMailer app shutdown with error: %v", err)
		} else {
			t.Logf("TodoMailer app shut down gracefully")
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
