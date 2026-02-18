package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/google/uuid"
)

// TodoUpdaterAction is an assistant action for updating todos.
type TodoUpdaterAction struct {
	uow     domain.UnitOfWork
	updater usecases.TodoUpdater
}

// NewTodoUpdaterAction creates a new instance of TodoUpdaterAction.
func NewTodoUpdaterAction(uow domain.UnitOfWork, updater usecases.TodoUpdater) TodoUpdaterAction {
	return TodoUpdaterAction{
		uow:     uow,
		updater: updater,
	}
}

// StatusMessage returns a status message about the tool execution.
func (t TodoUpdaterAction) StatusMessage() string {
	return "✏️ Updating your todo..."
}

// Definition returns the assistant action definition for TodoUpdaterAction.
func (tut TodoUpdaterAction) Definition() domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name:        "update_todo",
		Description: "Update metadata for exactly one existing todo. Required key: id (UUID). Optional keys: title and status. Use this tool only for title/status changes (never due date). status must be OPEN or DONE. No extra keys. Valid: {\"id\":\"<uuid>\",\"status\":\"DONE\"}. Invalid: {\"id\":\"<uuid>\",\"due_date\":\"2026-04-30\"}.",
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"id": {
					Type:        "string",
					Description: "Todo UUID. REQUIRED.",
					Required:    true,
				},
				"title": {
					Type:        "string",
					Description: "New title (optional).",
					Required:    false,
				},
				"status": {
					Type:        "string",
					Description: "OPEN or DONE (optional).",
					Required:    false,
				},
			},
		},
	}
}

// Execute executes TodoUpdaterAction.
func (tut TodoUpdaterAction) Execute(ctx context.Context, call domain.AssistantActionCall, _ []domain.AssistantMessage) domain.AssistantMessage {
	params := struct {
		ID     string  `json:"id"`
		Title  *string `json:"title"`
		Status *string `json:"status"`
	}{}

	exampleArgs := `{"id":"<uuid>","status":"DONE", "title":"New title"}`

	err := unmarshalToolArguments(call.Input, &params)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	todoID, err := uuid.Parse(params.ID)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"invalid_todo_id","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	var todo domain.Todo
	err = tut.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		td, err := tut.updater.Update(ctx, uow, todoID, params.Title, (*domain.TodoStatus)(params.Status), nil)
		if err != nil {
			return err
		}
		todo = td
		return nil
	})

	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"update_todo_error","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      fmt.Sprintf(`{"message":"Your todo was updated successfully! todo: {"id":"%s", "title":"%s", "due_date":"%s", "status":"%s"}"}`, todo.ID, todo.Title, todo.DueDate.Format(time.DateOnly), todo.Status),
	}
}
