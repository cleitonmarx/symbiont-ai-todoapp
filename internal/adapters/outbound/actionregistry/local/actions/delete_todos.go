package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/google/uuid"
)

// BulkTodoDeleterAction is an assistant action for deleting multiple todos.
type BulkTodoDeleterAction struct {
	uow     domain.UnitOfWork
	deleter usecases.TodoDeleter
}

// NewBulkTodoDeleterAction creates a new instance of BulkTodoDeleterAction.
func NewBulkTodoDeleterAction(uow domain.UnitOfWork, deleter usecases.TodoDeleter) BulkTodoDeleterAction {
	return BulkTodoDeleterAction{
		uow:     uow,
		deleter: deleter,
	}
}

// StatusMessage returns a status message about the action execution.
func (a BulkTodoDeleterAction) StatusMessage() string {
	return "🗑️ Deleting todos..."
}

// Definition returns the assistant action definition for BulkTodoDeleterAction.
func (a BulkTodoDeleterAction) Definition() domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name:        "delete_todos",
		Description: "Delete multiple todos.",
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"todos": {
					Type:        "array",
					Description: "List of todos to delete. Each item: {id,title}. REQUIRED.",
					Required:    true,
					Items: &domain.AssistantActionField{
						Type:        "object",
						Description: "Todo item to delete.",
						Fields: map[string]domain.AssistantActionField{
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

		Approval: domain.AssistantActionApproval{
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

// Execute executes BulkTodoDeleterAction.
func (a BulkTodoDeleterAction) Execute(ctx context.Context, call domain.AssistantActionCall, _ []domain.AssistantMessage) domain.AssistantMessage {
	params := deleteTodoParams{}

	exampleArgs := `{"todos":[{"id":"<uuid>","title":"Sample Task"},{"id":"<uuid>","title":"Another Task"}]}`

	err := unmarshalActionInput(call.Input, &params)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", err.Error(), exampleArgs),
		}
	}
	if len(params.Todos) == 0 {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", "todos must not be empty.", exampleArgs),
		}
	}

	ids := make([]uuid.UUID, 0, len(params.Todos))
	for i, todo := range params.Todos {
		todoID, parseErr := uuid.Parse(todo.ID)
		if parseErr != nil {
			return domain.AssistantMessage{
				Role:         domain.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_todo_id", fmt.Sprintf("id at index %d is invalid: %s", i, parseErr.Error()), exampleArgs),
			}
		}

		if strings.TrimSpace(todo.Title) == "" {
			return domain.AssistantMessage{
				Role:         domain.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_title", fmt.Sprintf("title at index %d must not be empty.", i), exampleArgs),
			}
		}

		ids = append(ids, todoID)
	}

	err = a.uow.Execute(ctx, func(uowCtx context.Context, uow domain.UnitOfWork) error {
		for i, id := range ids {
			deleteErr := a.deleter.Delete(uowCtx, uow, id)
			if deleteErr != nil {
				return fmt.Errorf("id at index %d: %w", i, deleteErr)
			}
		}
		return nil
	})
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("delete_todos_error", fmt.Sprintf("Failed to delete todos: %s", err.Error()), exampleArgs),
		}
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      formatDeletedRows(ids),
	}
}
