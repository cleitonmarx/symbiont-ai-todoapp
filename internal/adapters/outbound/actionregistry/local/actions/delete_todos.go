package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
	"github.com/google/uuid"
)

// DeleteTodosAction is an assistant action for deleting multiple todos.
type DeleteTodosAction struct {
	uow     transaction.UnitOfWork
	deleter todo.Deleter
}

// NewDeleteTodosAction creates a new instance of DeleteTodosAction.
func NewDeleteTodosAction(uow transaction.UnitOfWork, deleter todo.Deleter) DeleteTodosAction {
	return DeleteTodosAction{
		uow:     uow,
		deleter: deleter,
	}
}

// StatusMessage returns a status message about the action execution.
func (a DeleteTodosAction) StatusMessage() string {
	return "🗑️ Deleting todos..."
}

// Renderer returns the deterministic result renderer for deleted todos.
func (a DeleteTodosAction) Renderer() (assistant.ActionResultRenderer, bool) {
	return deleteTodosRenderer{}, true
}

// Definition returns the assistant action definition for DeleteTodosAction.
func (a DeleteTodosAction) Definition() assistant.ActionDefinition {
	return assistant.ActionDefinition{
		Name:        "delete_todos",
		Description: "Delete multiple todos.",
		Input: assistant.ActionInput{
			Type: "object",
			Fields: map[string]assistant.ActionField{
				"todos": {
					Type:        "array",
					Description: "List of todos to delete. Each item: {id,title}. REQUIRED.",
					Required:    true,
					Items: &assistant.ActionField{
						Type:        "object",
						Description: "Todo item to delete.",
						Fields: map[string]assistant.ActionField{
							"id": {
								Type:        "string",
								Description: "ID of the todo to delete. REQUIRED.",
								Required:    true,
							},
							"title": {
								Type:        "string",
								Description: "Title of the todo to delete (for confirmation only). REQUIRED.",
								Required:    true,
							},
						},
					},
				},
			},
		},

		Approval: assistant.ActionApproval{
			Required:    true,
			Title:       "Confirm deletion of todos",
			Description: "Deleting todos is irreversible. Please confirm.",
			PreviewFields: []string{
				"todos[].title",
			},
			Timeout: 2 * time.Minute,
		},
	}
}

type deleteTodoParams struct {
	Todos []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"todos"`
}

// Execute executes DeleteTodosAction.
func (a DeleteTodosAction) Execute(ctx context.Context, call assistant.ActionCall, _ []assistant.Message) assistant.Message {
	params := deleteTodoParams{}

	exampleArgs := `{"todos":[{"id":"<uuid>","title":"Sample Task"},{"id":"<uuid>","title":"Another Task"}]}`

	err := unmarshalActionInput(call.Input, &params)
	if err != nil {
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", err.Error(), exampleArgs),
		}
	}
	if len(params.Todos) == 0 {
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", "todos must not be empty.", exampleArgs),
		}
	}

	ids := make([]uuid.UUID, 0, len(params.Todos))
	for i, todo := range params.Todos {
		todoID, parseErr := uuid.Parse(todo.ID)
		if parseErr != nil {
			return assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_todo_id", fmt.Sprintf("id at index %d is invalid: %s", i, parseErr.Error()), exampleArgs),
			}
		}

		if strings.TrimSpace(todo.Title) == "" {
			return assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_title", fmt.Sprintf("title at index %d must not be empty.", i), exampleArgs),
			}
		}

		ids = append(ids, todoID)
	}

	err = a.uow.Execute(ctx, func(uowCtx context.Context, scope transaction.Scope) error {
		for i, id := range ids {
			deleteErr := a.deleter.Delete(uowCtx, scope, id)
			if deleteErr != nil {
				return fmt.Errorf("id at index %d: %w", i, deleteErr)
			}
		}
		return nil
	})
	if err != nil {
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("delete_todos_error", fmt.Sprintf("Failed to delete todos: %s", err.Error()), exampleArgs),
		}
	}

	return assistant.Message{
		Role:         assistant.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      formatDeletedRows(ids),
	}
}
