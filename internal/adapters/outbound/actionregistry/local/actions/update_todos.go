package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	todouc "github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
	"github.com/google/uuid"
)

// UpdateTodosAction is an assistant action for updating multiple todos.
type UpdateTodosAction struct {
	uow     transaction.UnitOfWork
	updater todouc.Updater
}

// NewUpdateTodosAction creates a new instance of UpdateTodosAction.
func NewUpdateTodosAction(uow transaction.UnitOfWork, updater todouc.Updater) UpdateTodosAction {
	return UpdateTodosAction{
		uow:     uow,
		updater: updater,
	}
}

// StatusMessage returns a status message about the action execution.
func (a UpdateTodosAction) StatusMessage() string {
	return "✏️ Updating your todos..."
}

// Renderer returns the deterministic result renderer for todo updates.
func (a UpdateTodosAction) Renderer() (assistant.ActionResultRenderer, bool) {
	return updateTodosRenderer{}, true
}

// Definition returns the assistant action definition for UpdateTodosAction.
func (a UpdateTodosAction) Definition() assistant.ActionDefinition {
	return assistant.ActionDefinition{
		Name:        "update_todos",
		Description: "Update title and/or status for multiple todos.",
		Input: assistant.ActionInput{
			Type: "object",
			Fields: map[string]assistant.ActionField{
				"todos": {
					Type:        "array",
					Description: "List of todo updates. Each item: {id,title?,status?}. REQUIRED.",
					Required:    true,
					Items: &assistant.ActionField{
						Type:        "object",
						Description: "Todo item to update.",
						Required:    true,
						Fields: map[string]assistant.ActionField{
							"id": {
								Type:        "string",
								Description: "ID of the todo to update. REQUIRED.",
								Required:    true,
							},
							"title": {
								Type:        "string",
								Description: "New title for the todo. Optional but at least one of title or status must be present.",
								Required:    false,
							},
							"status": {
								Type:        "string",
								Description: "New status for the todo. Allowed values: OPEN or DONE. Optional but at least one of title or status must be present.",
								Required:    false,
								Enum:        []any{todo.Status_OPEN, todo.Status_DONE},
							},
						},
					},
				},
			},
		},
		Approval: assistant.ActionApproval{
			Required:    true,
			Title:       "Confirm update of todos",
			Description: "Updating todos will modify existing items. Please confirm.",
			PreviewFields: []string{
				"todos[].id",
				"todos[].title",
				"todos[].status",
			},
			Timeout: 2 * time.Minute,
		},
	}
}

// Execute executes UpdateTodosAction.
func (a UpdateTodosAction) Execute(ctx context.Context, call assistant.ActionCall, _ []assistant.Message) assistant.Message {
	params := struct {
		Todos []struct {
			ID     string  `json:"id"`
			Title  *string `json:"title"`
			Status *string `json:"status"`
		} `json:"todos"`
	}{}
	exampleArgs := `{"todos":[{"id":"<uuid>","title":"Pay rent (done)","status":"DONE"},{"id":"<uuid>","title":"Buy groceries"}]}`

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

	type updateItem struct {
		ID     uuid.UUID
		Title  *string
		Status *todo.Status
	}
	items := make([]updateItem, 0, len(params.Todos))
	for i, td := range params.Todos {
		todoID, parseErr := uuid.Parse(td.ID)
		if parseErr != nil {
			return assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_todo_id", fmt.Sprintf("todo at index %d has invalid id: %s", i, parseErr.Error()), exampleArgs),
			}
		}

		var statusPtr *todo.Status
		if td.Status != nil {
			status := todo.Status(*td.Status)
			if status != todo.Status_OPEN && status != todo.Status_DONE {
				return assistant.Message{
					Role:         assistant.ChatRole_Tool,
					ActionCallID: &call.ID,
					Content:      newActionError("invalid_status", fmt.Sprintf("todo at index %d has invalid status: %s", i, *td.Status), exampleArgs),
				}
			}
			statusPtr = &status
		}

		items = append(items, updateItem{
			ID:     todoID,
			Title:  td.Title,
			Status: statusPtr,
		})
	}

	todos := make([]todo.Todo, 0, len(items))
	err = a.uow.Execute(ctx, func(uowCtx context.Context, scope transaction.Scope) error {
		for i, item := range items {
			todo, updateErr := a.updater.Update(uowCtx, scope, item.ID, item.Title, item.Status, nil)
			if updateErr != nil {
				return fmt.Errorf("todo at index %d: %w", i, updateErr)
			}
			todos = append(todos, todo)
		}
		return nil
	})
	if err != nil {
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("update_todos_error", err.Error(), exampleArgs),
		}
	}

	return assistant.Message{
		Role:         assistant.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      formatTodosRows(todos),
	}
}
