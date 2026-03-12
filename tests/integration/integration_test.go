//go:build integration

package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/graphql"
	gqlmodels "github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/graphql/gen"
	rest "github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/app"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/board"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/chat"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
)

// boardSummaryQueue is used to receive completed board summaries for verification in tests.
var (
	boardSummaryQueue        board.CompletedBoardSummaryChannel
	conversationSummaryQueue chat.CompletedConversationSummaryChannel
	conversationTitleQueue   chat.CompletedConversationTitleUpdateChannel
	restCli                  *rest.ClientWithResponses
)

func TestMain(m *testing.M) {
	todoApp := app.NewMonolithic(
		&InitEnvVars{
			envVars: map[string]string{
				"VAULT_ADDR":                                 "http://localhost:8200",
				"VAULT_TOKEN":                                "root-token",
				"VAULT_MOUNT_PATH":                           "secret",
				"VAULT_SECRET_PATH":                          "todoapp",
				"OTEL_EXPORTER_OTLP_ENDPOINT":                "http://localhost:4318",
				"DB_HOST":                                    "localhost",
				"DB_PORT":                                    "5432",
				"DB_NAME":                                    "todoappdb",
				"PUBSUB_EMULATOR_HOST":                       "localhost:8681",
				"PUBSUB_PROJECT_ID":                          "local-dev",
				"PUBSUB_TOPIC_ID":                            "Todo",
				"TODO_EVENTS_SUBSCRIPTION_ID":                "todo_summary_generator",
				"CHAT_EVENTS_SUBSCRIPTION_ID":                "chat_message_summary_generator",
				"CHAT_TITLE_EVENTS_SUBSCRIPTION_ID":          "chat_message_title_generator",
				"LLM_EMBEDDING_MODEL_HOST":                   "http://localhost:12434",
				"LLM_MODEL_HOST":                             "http://localhost:12434",
				"LLM_CHAT_SUMMARY_MODEL":                     "qwen3:4B-F16",
				"LLM_CHAT_TITLE_MODEL":                       "qwen3:4B-F16",
				"LLM_SUMMARY_MODEL":                          "qwen3:4B-F16",
				"LLM_EMBEDDING_MODEL":                        "embeddinggemma:300M-Q8_0",
				"MCP_GATEWAY_ENDPOINT":                       "http://localhost:8811",
				"ACTION_APPROVAL_EVENTS_SUBSCRIPTION_PREFIX": "action_approval_dispatcher",
			},
		},
		&InitDockerCompose{},
	)

	boardSummaryQueue = make(board.CompletedBoardSummaryChannel)
	depend.Register(boardSummaryQueue)

	conversationSummaryQueue = make(chat.CompletedConversationSummaryChannel)
	depend.Register(conversationSummaryQueue)

	conversationTitleQueue = make(chat.CompletedConversationTitleUpdateChannel)
	depend.Register(conversationTitleQueue)

	var err error

	restCli, err = rest.NewClientWithResponses("http://localhost:8080")
	if err != nil {
		log.Fatalf("failed to create REST client: %v", err)
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownCh := todoApp.RunAsync(cancelCtx)

	err = todoApp.WaitForReadiness(cancelCtx, 10*time.Minute)
	if err != nil {
		cancel()
		log.Fatalf("TodoApp app failed to become ready: %v", err)
	}

	// Run tests
	code := m.Run()

	// Shutdown the app
	cancel()

	select {
	case <-time.After(1 * time.Minute):
		log.Fatalf("TodoApp app did not shut down in time")
	case err = <-shutdownCh:
		if err != nil {
			log.Fatalf("TodoApp app shutdown with error: %v", err)
		} else {
			log.Printf("TodoApp app shut down gracefully")
		}
	}

	os.Exit(code)
}

func TestTodoApp_TodoRestAPI(t *testing.T) {
	t.Run("create-todo", func(t *testing.T) {
		createResp, err := restCli.CreateTodoWithResponse(t.Context(), rest.CreateTodoJSONRequestBody{
			Title:   "Integration Test Todo",
			DueDate: types.Date{Time: time.Now().Add(24 * time.Hour)},
		})
		require.NoError(t, err, "failed to call CreateTodo endpoint")
		require.NotNil(t, createResp.JSON201, "expected non-nil response for CreateTodo")

	})

	var todos []rest.Todo
	t.Run("list-created-todo", func(t *testing.T) {
		resp, err := restCli.ListTodosWithResponse(t.Context(), &rest.ListTodosParams{
			Page:     1,
			PageSize: 10,
		})

		require.NoError(t, err, "failed to call ListTodos endpoint")
		require.NotNil(t, resp.JSON200, "expected non-nil response for ListTodos")
		require.Equal(t, 1, len(resp.JSON200.Items), "expected 1 todo in the list")

		todos = resp.JSON200.Items
	})

	t.Run("update-todos", func(t *testing.T) {
		for _, todo := range todos {
			deadline := time.Now().Add(48 * time.Hour)
			dueDate := time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 0, 0, 0, 0, time.UTC)
			updateResp, err := restCli.UpdateTodoWithResponse(t.Context(), todo.Id, rest.UpdateTodoJSONRequestBody{
				DueDate: &types.Date{Time: dueDate},
			})
			require.NoError(t, err, "failed to call UpdateTodo endpoint")
			require.NotNil(t, updateResp.JSON200, "expected non-nil response for UpdateTodo")
			require.Equal(t, dueDate, updateResp.JSON200.DueDate.Time, "expected todo status to be 'completed'")
		}
	})

	t.Run("check-board-summary-generated", func(t *testing.T) {
		select {
		case summary := <-boardSummaryQueue:
			require.Equal(t, 1, summary.Content.Counts.Open,
				"expected board summary to have at least one open todo",
			)
		case <-time.After(1 * time.Minute):
			t.Fatalf("Timed out waiting for board summary in queue")
		}
	})

	t.Run("delete-todos", func(t *testing.T) {
		for _, todo := range todos {
			deleteResp, err := restCli.DeleteTodoWithResponse(t.Context(), todo.Id)
			require.NoError(t, err, "failed to call DeleteTodo endpoint")
			require.Equal(t, 204, deleteResp.StatusCode(), "expected 204 No Content response for DeleteTodo")
		}

		// Verify todos are deleted
		listResp, err := restCli.ListTodosWithResponse(t.Context(), &rest.ListTodosParams{
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err, "failed to call ListTodos endpoint after deletions")
		require.NotNil(t, listResp.JSON200, "expected non-nil response for ListTodos after deletions")
		require.Equal(t, 0, len(listResp.JSON200.Items), "expected 0 todos in the list after deletions")
	})
}

func TestTodoApp_GraphQLAPI(t *testing.T) {
	cli := graphql.NewClient("http://localhost:8085/v1/query")

	t.Run("create-todos", func(t *testing.T) {
		for range 2 {
			createResp, err := restCli.CreateTodoWithResponse(t.Context(), rest.CreateTodoJSONRequestBody{
				Title:   "Integration Test Todo",
				DueDate: types.Date{Time: time.Now().Add(24 * time.Hour)},
			})
			require.NoError(t, err, "failed to call CreateTodo endpoint")
			require.NotNil(t, createResp.JSON201, "expected non-nil response for CreateTodo")
		}
	})

	var todos []*gqlmodels.Todo
	t.Run("list-created-todos", func(t *testing.T) {
		listResp, err := cli.ListTodos(t.Context(), nil, 1, 10)
		require.NoError(t, err, "failed to call ListTodos GraphQL query")
		require.NotNil(t, listResp, "expected non-nil response for ListTodos GraphQL query")
		require.Equal(t, 2, len(listResp.Items), "expected 2 todos in the list")

		todos = listResp.Items
	})

	t.Run("update-todos", func(t *testing.T) {
		var updateParams []gqlmodels.UpdateTodoParams
		for _, todo := range todos {
			updateParams = append(updateParams, gqlmodels.UpdateTodoParams{
				ID:     todo.ID,
				Status: common.Ptr(gqlmodels.TodoStatusDone),
			})
		}

		updateResp, err := cli.UpdateTodos(t.Context(), updateParams)
		require.NoError(t, err, "failed to call UpdateTodos GraphQL mutation")
		require.NotNil(t, updateResp, "expected non-nil response for UpdateTodos GraphQL mutation")
		require.Equal(t, 2, len(updateResp), "expected 2 todos in the update response")
	})

	t.Run("delete-todos", func(t *testing.T) {
		var ids []uuid.UUID
		for _, todo := range todos {
			ids = append(ids, todo.ID)
		}

		deleteResp, err := cli.DeleteTodos(t.Context(), ids)
		require.NoError(t, err, "failed to call DeleteTodos GraphQL mutation")
		require.NotNil(t, deleteResp, "expected non-nil response for DeleteTodos GraphQL mutation")
		require.Equal(t, 2, len(deleteResp), "expected 2 results in the delete response")
		for _, deleted := range deleteResp {
			require.True(t, deleted, "expected todo to be deleted successfully")
		}

		// Verify todos are deleted
		listResp, err := cli.ListTodos(t.Context(), nil, 1, 10)
		require.NoError(t, err, "failed to call ListTodos GraphQL query after deletions")
		require.NotNil(t, listResp, "expected non-nil response for ListTodos GraphQL query after deletions")
		require.Equal(t, 0, len(listResp.Items), "expected 0 todos in the list after deletions")
	})
}

func TestTodoApp_ChatRestAPI(t *testing.T) {
	var (
		model            string
		conversationID   uuid.UUID
		createTodoPrompt = "Create one todo named \"Integration Test Todo\" due tomorrow."
	)

	t.Run("fetch-available-models", func(t *testing.T) {
		expectedModel := rest.ModelInfo{Id: "docker.io/ai/qwen3:4B-F16", Name: "qwen3:4B-F16"}
		modelsResp, err := restCli.ListAvailableModelsWithResponse(t.Context())
		require.NoError(t, err, "failed to call GetAvailableModels endpoint")
		require.NotNil(t, modelsResp.JSON200, "expected non-nil response for GetAvailableModels")
		require.Greater(t, len(modelsResp.JSON200.Models), 0, "expected at least one available model")
		require.Contains(t, modelsResp.JSON200.Models, expectedModel, "expected available models to include qwen3:4B-F16")
		i := slices.Index(modelsResp.JSON200.Models, expectedModel)
		model = modelsResp.JSON200.Models[i].Id
	})

	t.Run("create-todo", func(t *testing.T) {
		chatResp, err := restCli.StreamChat(t.Context(), rest.StreamChatJSONRequestBody{
			Model:   model,
			Message: createTodoPrompt,
		})
		require.NoError(t, err, "failed to call StreamChat endpoint")
		defer chatResp.Body.Close() //nolint:errcheck
		require.Equal(t, 200, chatResp.StatusCode, "expected 200 OK response for StreamChat")

		deltaText, actionStartedText, actionCompletedCount, cID := readChatEventsText(t, chatResp.Body)
		conversationID = cID

		fmt.Println("Chat response:", deltaText)
		require.Contains(t, actionStartedText, "📝 Creating your todos...")
		require.GreaterOrEqual(t, actionCompletedCount, 1)
		require.Contains(t, deltaText, "Integration Test Todo", "expected chat response to contain created todo title")
	})

	var lastConversation rest.Conversation
	t.Run("auto-rename-conversation-title", func(t *testing.T) {
		resp, err := restCli.ListConversationsWithResponse(t.Context(), &rest.ListConversationsParams{
			Page:     1,
			PageSize: 2,
		})
		require.NoError(t, err, "failed to call ListConversations endpoint")
		require.Equal(t, http.StatusOK, resp.StatusCode(), "expected 200 OK response for ListConversations")
		require.NotNil(t, resp.JSON200, "expected non-nil response for ListConversations")
		require.Len(t, resp.JSON200.Conversations, 1, "expected 1 conversation in the list")

		autoTitle := assistant.GenerateAutoConversationTitle(createTodoPrompt)

		for _, conv := range resp.JSON200.Conversations {
			if conv.Id == conversationID {
				lastConversation = conv
				break
			}
		}

		require.Equal(t, autoTitle, lastConversation.Title, "expected conversation title to be auto-generated based on the initial user message")
		require.Equal(t, rest.ConversationTitleSourceAuto, lastConversation.TitleSource, "expected conversation title source to be 'auto'")
	})

	t.Run("chat-fetch-todo", func(t *testing.T) {
		chatResp, err := restCli.StreamChat(t.Context(), rest.StreamChatJSONRequestBody{
			ConversationId: &conversationID,
			Model:          model,
			Message:        "Fetch the todo \"Integration Test Todo\".",
		})
		require.NoError(t, err, "failed to call StreamChat endpoint")
		defer chatResp.Body.Close() //nolint:errcheck
		require.Equal(t, 200, chatResp.StatusCode, "expected 200 OK response for StreamChat")

		deltaText, actionStartedText, actionCompletedCount, _ := readChatEventsText(t, chatResp.Body)

		fmt.Println("Chat response:", deltaText)
		require.Contains(t, actionStartedText, "🔎 Fetching todos...")
		require.GreaterOrEqual(t, actionCompletedCount, 1)
		require.Contains(t, deltaText, "Integration Test Todo", "expected chat response to contain created todo title")
		require.Contains(t, strings.ToLower(deltaText), "open", "expected chat response to contain created todo status")

	})

	t.Run("mark-todo-done", func(t *testing.T) {
		chatResp, err := restCli.StreamChat(t.Context(), rest.StreamChatJSONRequestBody{
			ConversationId: &conversationID,
			Model:          model,
			Message:        "Mark my todo \"Integration Test Todo\" as done.",
		})
		require.NoError(t, err, "failed to call StreamChat endpoint")
		defer chatResp.Body.Close() //nolint:errcheck
		require.Equal(t, 200, chatResp.StatusCode, "expected 200 OK response for StreamChat")

		scanner := newSSEScanner(chatResp.Body)

		approvalRequest := readChatApprovalRequiredEvent(t, scanner)
		require.Equal(t, "update_todos", approvalRequest.Name, "expected action approval request for 'update_todos' action")
		fmt.Printf("\nReceived action approval request: %+v\n", approvalRequest)

		approvalResp, err := restCli.SubmitActionApprovalWithResponse(t.Context(), rest.SubmitActionApprovalRequest{
			ActionCallId:   approvalRequest.ActionCallID,
			ActionName:     &approvalRequest.Name,
			ConversationId: approvalRequest.ConversationID,
			Reason:         common.Ptr("approved by integration test"),
			Status:         rest.ActionApprovalStatusAPPROVED,
			TurnId:         approvalRequest.TurnID,
		})
		require.NoError(t, err, "failed to submit action approval")
		require.NotNil(t, approvalResp, "expected non-nil response for SubmitActionApproval")
		require.Equal(t, http.StatusAccepted, approvalResp.StatusCode(), "expected 202 Accepted for SubmitActionApproval")

		approvalResolved := readChatApprovalResolvedEvent(t, scanner)
		require.Equal(t, approvalRequest.ActionCallID, approvalResolved.ActionCallID, "expected resolved action_call_id to match required event")
		require.Equal(t, approvalRequest.ConversationID, approvalResolved.ConversationID, "expected resolved conversation_id to match required event")
		require.Equal(t, approvalRequest.TurnID, approvalResolved.TurnID, "expected resolved turn_id to match required event")
		require.Equal(t, assistant.ChatMessageApprovalStatus_Approved, approvalResolved.Status, "expected resolved status to be APPROVED")
		fmt.Printf("\nReceived action approval resolved event: %+v\n", approvalResolved)

		deltaText, actionStartedText, actionCompletedCount, _ := readChatEventsTextFromScanner(t, scanner)

		fmt.Println("Chat response:", deltaText)
		require.Contains(t, actionStartedText, "✏️ Updating your todos...")
		require.GreaterOrEqual(t, actionCompletedCount, 1)
		require.Contains(t, deltaText, "Integration Test Todo", "expected chat response to contain created todo title")
		require.Contains(t, strings.ToLower(deltaText), "done", "expected chat response to contain created todo status")
	})

	t.Run("delete-todo-with-approval", func(t *testing.T) {
		chatResp, err := restCli.StreamChat(t.Context(), rest.StreamChatJSONRequestBody{
			ConversationId: &conversationID,
			Model:          model,
			Message:        "Delete the todo \"Integration Test Todo\".",
		})
		require.NoError(t, err, "failed to call StreamChat endpoint")
		defer chatResp.Body.Close() //nolint:errcheck
		require.Equal(t, 200, chatResp.StatusCode, "expected 200 OK response for StreamChat")

		scanner := newSSEScanner(chatResp.Body)

		approvalRequest := readChatApprovalRequiredEvent(t, scanner)
		require.Equal(t, "delete_todos", approvalRequest.Name, "expected action approval request for 'delete_todos' action")
		fmt.Printf("\nReceived action approval request: %+v\n", approvalRequest)

		approvalResp, err := restCli.SubmitActionApprovalWithResponse(t.Context(), rest.SubmitActionApprovalRequest{
			ActionCallId:   approvalRequest.ActionCallID,
			ActionName:     &approvalRequest.Name,
			ConversationId: approvalRequest.ConversationID,
			Reason:         common.Ptr("approved by integration test"),
			Status:         rest.ActionApprovalStatusAPPROVED,
			TurnId:         approvalRequest.TurnID,
		})

		require.NoError(t, err, "failed to submit action approval")
		require.NotNil(t, approvalResp, "expected non-nil response for SubmitActionApproval")
		require.Equal(t, http.StatusAccepted, approvalResp.StatusCode(), "expected 202 Accepted for SubmitActionApproval")

		approvalResolved := readChatApprovalResolvedEvent(t, scanner)
		require.Equal(t, approvalRequest.ActionCallID, approvalResolved.ActionCallID, "expected resolved action_call_id to match required event")
		require.Equal(t, approvalRequest.ConversationID, approvalResolved.ConversationID, "expected resolved conversation_id to match required event")
		require.Equal(t, approvalRequest.TurnID, approvalResolved.TurnID, "expected resolved turn_id to match required event")
		require.Equal(t, assistant.ChatMessageApprovalStatus_Approved, approvalResolved.Status, "expected resolved status to be APPROVED")
		fmt.Printf("\nReceived action approval resolved event: %+v\n", approvalResolved)

		deltaText, actionStartedText, actionCompletedCount, _ := readChatEventsTextFromScanner(t, scanner)
		fmt.Println("Chat response:", deltaText)
		require.Contains(t, actionStartedText, "🗑️ Deleting todos...")
		require.GreaterOrEqual(t, actionCompletedCount, 1)
	})

	t.Run("check-conversation-summary-generated", func(t *testing.T) {
		select {
		case summary := <-conversationSummaryQueue:
			require.Contains(t, summary.CurrentStateSummary, "Integration Test Todo")
		case <-time.After(1 * time.Minute):
			t.Fatalf("Timed out waiting for conversation summary in queue")
		}
	})

	t.Run("check-conversation-title-update-generated", func(t *testing.T) {
		select {
		case titleUpdate := <-conversationTitleQueue:
			updatedTitleSource := rest.ConversationTitleSource(titleUpdate.TitleSource)
			require.NotEqual(t, lastConversation.Title, titleUpdate.Title, "expected conversation title to be updated from the initial auto-generated title")
			require.Equal(t, rest.ConversationTitleSourceLlm, updatedTitleSource, "expected conversation title source to be 'llm'")
			fmt.Printf("Conversation title updated to: %s, source: %s\n", titleUpdate.Title, titleUpdate.TitleSource)
		case <-time.After(1 * time.Minute):
			t.Fatalf("Timed out waiting for conversation title update in queue")
		}
	})
}

func TestTodoApp_ConversationRestAPI(t *testing.T) {
	var conversationID uuid.UUID
	t.Run("list-conversations", func(t *testing.T) {
		listResp, err := restCli.ListConversationsWithResponse(t.Context(), &rest.ListConversationsParams{
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err, "failed to call ListConversations endpoint")
		require.NotNil(t, listResp.JSON200, "expected non-nil response for ListConversations")
		require.Len(t, listResp.JSON200.Conversations, 1, "expected 1 conversation in the list")
		conversationID = uuid.UUID(listResp.JSON200.Conversations[0].Id)
	})

	t.Run("update-conversation", func(t *testing.T) {
		newTitle := "Updated Conversation Title"
		updateResp, err := restCli.UpdateConversationWithResponse(t.Context(), conversationID, rest.UpdateConversationJSONRequestBody{
			Title: newTitle,
		})
		require.NoError(t, err, "failed to call UpdateConversation endpoint")
		require.NotNil(t, updateResp.JSON200, "expected non-nil response for UpdateConversation")
		require.Equal(t, newTitle, updateResp.JSON200.Title, "expected conversation title to be updated")
		require.Equal(t, rest.ConversationTitleSourceUser, updateResp.JSON200.TitleSource, "expected conversation title source to be 'user'")
	})

	t.Run("list-messages-in-conversation", func(t *testing.T) {
		messagesResp, err := restCli.ListChatMessagesWithResponse(t.Context(), &rest.ListChatMessagesParams{
			ConversationId: conversationID,
			Page:           1,
			PageSize:       100,
		})
		require.NoError(t, err, "failed to call ListChatMessages endpoint")
		require.NotNil(t, messagesResp.JSON200, "expected non-nil response for ListChatMessages")
		require.Len(t, messagesResp.JSON200.Messages, 8, "expected 8 messages in the conversation (4 user messages + 4 action calls)")
	})

	t.Run("delete-conversation", func(t *testing.T) {
		deleteResp, err := restCli.DeleteConversationWithResponse(t.Context(), conversationID)
		require.NoError(t, err, "failed to call DeleteConversation endpoint")
		require.Equal(t, http.StatusNoContent, deleteResp.StatusCode(), "expected 204 No Content response for DeleteConversation")

		// Verify conversation is deleted
		listResp, err := restCli.ListConversationsWithResponse(t.Context(), &rest.ListConversationsParams{
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err, "failed to call ListConversations endpoint after deletion")
		require.Len(t, listResp.JSON200.Conversations, 0, "expected 0 conversations in the list after deletion")

		// Verify messages are deleted
		messagesResp, err := restCli.ListChatMessagesWithResponse(t.Context(), &rest.ListChatMessagesParams{
			ConversationId: conversationID,
			Page:           1,
			PageSize:       100,
		})
		require.NoError(t, err, "failed to call ListChatMessages endpoint after conversation deletion")
		require.NotNil(t, messagesResp.JSON200, "expected non-nil response for ListChatMessages after conversation deletion")
		require.Len(t, messagesResp.JSON200.Messages, 0, "expected 0 messages in the conversation after deletion")
	})
}

func TestToodApp_MCPGatewayIntegration(t *testing.T) {
	t.Run("mcp-fetch-web-page-with-approval", func(t *testing.T) {
		chatResp, err := restCli.StreamChat(t.Context(), rest.StreamChatJSONRequestBody{
			Model:   "qwen3:4B-F16",
			Message: "Open the external webpage https://duckduckgo.com/ and return only the page title.",
		})

		require.NoError(t, err, "failed to call StreamChat endpoint")
		defer chatResp.Body.Close() //nolint:errcheck
		require.Equal(t, 200, chatResp.StatusCode, "expected 200 OK response for StreamChat")

		scanner := newSSEScanner(chatResp.Body)

		approvalRequest := readChatApprovalRequiredEvent(t, scanner)
		require.Equal(t, "fetch_content", approvalRequest.Name, "expected action approval request for 'fetch_content' action")
		fmt.Printf("\nReceived action approval request: %+v\n", approvalRequest)

		approvalResp, err := restCli.SubmitActionApprovalWithResponse(t.Context(), rest.SubmitActionApprovalRequest{
			ActionCallId:   approvalRequest.ActionCallID,
			ActionName:     &approvalRequest.Name,
			ConversationId: approvalRequest.ConversationID,
			Reason:         common.Ptr("approved by integration test"),
			Status:         rest.ActionApprovalStatusAPPROVED,
			TurnId:         approvalRequest.TurnID,
		})

		require.NoError(t, err, "failed to submit action approval")
		require.NotNil(t, approvalResp, "expected non-nil response for SubmitActionApproval")
		require.Equal(t, http.StatusAccepted, approvalResp.StatusCode(), "expected 202 Accepted for SubmitActionApproval")

		deltaText, actionStartedText, actionCompletedCount, _ := readChatEventsTextFromScanner(t, scanner)

		fmt.Println("Chat response:", deltaText)
		require.Contains(t, actionStartedText, "📄 Fetching page content...")
		require.GreaterOrEqual(t, actionCompletedCount, 1)
		require.Contains(t, deltaText, "DuckDuckGo", "expected chat response to contain content fetched from the web page")
	})
}

func readChatEventsText(t *testing.T, reader io.Reader) (string, string, int, uuid.UUID) {
	t.Helper()
	return readChatEventsTextFromScanner(t, newSSEScanner(reader))
}

func readChatEventsTextFromScanner(t *testing.T, scanner *bufio.Scanner) (string, string, int, uuid.UUID) {
	t.Helper()

	var (
		isDelta              = false
		isActionStarted      = false
		isActionCompleted    = false
		isMeta               = false
		actionStartedText    = strings.Builder{}
		actionCompletedCount = 0
		deltaText            = strings.Builder{}
		conversationID       uuid.UUID
	)
	for scanner.Scan() {
		line := scanner.Text()

		if isMeta {
			isMeta = false
			dataLine := scanner.Text()
			dataPayload := strings.TrimSpace(strings.TrimPrefix(dataLine, "data:"))
			var metaEvent assistant.TurnStarted
			err := json.Unmarshal([]byte(dataPayload), &metaEvent)
			require.NoError(t, err, "failed to unmarshal chat meta event payload")
			conversationID = metaEvent.ConversationID
		}
		if strings.HasPrefix(line, "event: turn_started") {
			isMeta = true
		}

		if isDelta {
			isDelta = false
			dataLine := scanner.Text()
			dataPayload := strings.TrimSpace(strings.TrimPrefix(dataLine, "data:"))
			var delta assistant.MessageDelta
			err := json.Unmarshal([]byte(dataPayload), &delta)
			require.NoError(t, err, "failed to unmarshal chat delta payload")
			deltaText.WriteString(delta.Text)
		}

		if strings.HasPrefix(line, "event: message_delta") {
			isDelta = true
		}

		if isActionStarted {
			isActionStarted = false
			dataline := scanner.Text()
			dataPayload := strings.TrimSpace(strings.TrimPrefix(dataline, "data:"))
			var actionStarted assistant.ActionCall
			err := json.Unmarshal([]byte(dataPayload), &actionStarted)
			require.NoError(t, err, "failed to unmarshal chat action started payload")
			actionStartedText.WriteString(actionStarted.Text)
		}

		if strings.HasPrefix(line, "event: action_started") {
			isActionStarted = true
		}

		if isActionCompleted {
			isActionCompleted = false
			dataline := scanner.Text()
			dataPayload := strings.TrimSpace(strings.TrimPrefix(dataline, "data:"))
			var actionCompleted assistant.ActionCompleted
			err := json.Unmarshal([]byte(dataPayload), &actionCompleted)
			require.NoError(t, err, "failed to unmarshal chat action completed payload")
			actionCompletedCount++
		}

		if strings.HasPrefix(line, "event: action_completed") {
			isActionCompleted = true
		}
	}

	require.NoError(t, scanner.Err(), "failed to scan chat stream")

	return deltaText.String(), actionStartedText.String(), actionCompletedCount, conversationID
}

func readChatApprovalRequiredEvent(t *testing.T, scanner *bufio.Scanner) assistant.ActionApprovalRequired {
	t.Helper()

	dataPayload := readFirstSSEEventData(t, scanner, "action_approval_required")
	var event assistant.ActionApprovalRequired
	err := json.Unmarshal([]byte(dataPayload), &event)
	require.NoError(t, err, "failed to unmarshal chat action approval required payload")
	return event
}

func readChatApprovalResolvedEvent(t *testing.T, scanner *bufio.Scanner) assistant.ActionApprovalResolved {
	t.Helper()

	dataPayload := readFirstSSEEventData(t, scanner, "action_approval_resolved")
	var event assistant.ActionApprovalResolved
	err := json.Unmarshal([]byte(dataPayload), &event)
	require.NoError(t, err, "failed to unmarshal chat action approval resolved payload")
	return event
}

func readFirstSSEEventData(t *testing.T, scanner *bufio.Scanner, eventName string) string {
	t.Helper()

	eventPrefix := "event: " + eventName
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, eventPrefix) {
			continue
		}

		require.True(t, scanner.Scan(), "missing SSE data line for event %s", eventName)
		dataLine := scanner.Text()
		return strings.TrimSpace(strings.TrimPrefix(dataLine, "data:"))
	}

	require.NoError(t, scanner.Err(), "failed to scan SSE stream for event %s", eventName)
	t.Fatalf("event %s not found in SSE stream", eventName)
	return ""
}

func newSSEScanner(reader io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(reader)
	const maxTokenSize = 1024 * 1024 // 1 MiB
	scanner.Buffer(make([]byte, 0, 64*1024), maxTokenSize)
	return scanner
}
