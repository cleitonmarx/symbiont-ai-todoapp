package actions

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/google/uuid"
)

// BulkTodoUpdaterAction is an assistant action for updating multiple todos.
type BulkTodoUpdaterAction struct {
	uow     domain.UnitOfWork
	updater usecases.TodoUpdater
}

// NewBulkTodoUpdaterAction creates a new instance of BulkTodoUpdaterAction.
func NewBulkTodoUpdaterAction(uow domain.UnitOfWork, updater usecases.TodoUpdater) BulkTodoUpdaterAction {
	return BulkTodoUpdaterAction{
		uow:     uow,
		updater: updater,
	}
}

// StatusMessage returns a status message about the action execution.
func (a BulkTodoUpdaterAction) StatusMessage() string {
	return "✏️ Updating your todos..."
}

// Definition returns the assistant action definition for BulkTodoUpdaterAction.
func (a BulkTodoUpdaterAction) Definition() domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name:        "update_todos",
		Description: "Update title and/or status for multiple todos.",
		Hints: domain.AssistantActionHints{
			UseWhen:   "Use for batch updates across multiple existing todos (plural updates).",
			AvoidWhen: "Do not use for due date-only changes or single-item updates.",
			ArgRules:  "Required key: todos. Each item requires id <UUID> and may include title and/or status (OPEN or DONE). Never place title text in id.",
		},
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"todos": {
					Type:        "array",
					Description: "List of todo updates. Each item: {id,title?,status?}. REQUIRED.",
					Required:    true,
					Items: &domain.AssistantActionField{
						Type:        "object",
						Description: "Todo item to update.",
						Required:    true,
						Fields: map[string]domain.AssistantActionField{
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
								Enum:        []any{domain.TodoStatus_OPEN, domain.TodoStatus_DONE},
							},
						},
					},
				},
			},
		},
	}
}

// Execute executes BulkTodoUpdaterAction.
func (a BulkTodoUpdaterAction) Execute(ctx context.Context, call domain.AssistantActionCall, _ []domain.AssistantMessage) domain.AssistantMessage {
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

	type updateItem struct {
		ID     uuid.UUID
		Title  *string
		Status *domain.TodoStatus
	}
	items := make([]updateItem, 0, len(params.Todos))
	for i, todo := range params.Todos {
		todoID, parseErr := uuid.Parse(todo.ID)
		if parseErr != nil {
			return domain.AssistantMessage{
				Role:         domain.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_todo_id", fmt.Sprintf("todo at index %d has invalid id: %s", i, parseErr.Error()), exampleArgs),
			}
		}

		var statusPtr *domain.TodoStatus
		if todo.Status != nil {
			status := domain.TodoStatus(*todo.Status)
			if status != domain.TodoStatus_OPEN && status != domain.TodoStatus_DONE {
				return domain.AssistantMessage{
					Role:         domain.ChatRole_Tool,
					ActionCallID: &call.ID,
					Content:      newActionError("invalid_status", fmt.Sprintf("todo at index %d has invalid status: %s", i, *todo.Status), exampleArgs),
				}
			}
			statusPtr = &status
		}

		items = append(items, updateItem{
			ID:     todoID,
			Title:  todo.Title,
			Status: statusPtr,
		})
	}

	todos := make([]domain.Todo, 0, len(items))
	err = a.uow.Execute(ctx, func(uowCtx context.Context, uow domain.UnitOfWork) error {
		for i, item := range items {
			todo, updateErr := a.updater.Update(uowCtx, uow, item.ID, item.Title, item.Status, nil)
			if updateErr != nil {
				return fmt.Errorf("todo at index %d: %w", i, updateErr)
			}
			todos = append(todos, todo)
		}
		return nil
	})
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("update_todos_error", err.Error(), exampleArgs),
		}
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      formatTodosRows(todos),
	}
}
