package actions

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/google/uuid"
)

type TodoDeleterAction struct {
	uow     domain.UnitOfWork
	deleter usecases.TodoDeleter
}

// NewTodoDeleterAction creates a new instance of TodoDeleterAction.
func NewTodoDeleterAction(uow domain.UnitOfWork, deleter usecases.TodoDeleter) TodoDeleterAction {
	return TodoDeleterAction{
		uow:     uow,
		deleter: deleter,
	}
}

// StatusMessage returns a status message about the tool execution.
func (t TodoDeleterAction) StatusMessage() string {
	return "🗑️ Deleting the todo..."
}

// Definition returns the assistant action definition for TodoDeleterAction.
func (tdt TodoDeleterAction) Definition() domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name:        "delete_todo",
		Description: "Delete exactly one todo by id. Required key: id (UUID string). No extra keys. If id is unknown, call fetch_todos first. Valid: {\"id\":\"<uuid>\"}. Invalid: {\"id\":\"<uuid>\",\"confirm\":true}.",
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"id": {
					Type:        "string",
					Description: "Todo UUID. REQUIRED.",
					Required:    true,
				},
			},
		},
	}
}

// Execute executes TodoDeleterAction.
func (tdt TodoDeleterAction) Execute(ctx context.Context, call domain.AssistantActionCall, _ []domain.AssistantMessage) domain.AssistantMessage {
	params := struct {
		ID uuid.UUID `json:"id"`
	}{}

	exampleArgs := `{"id":"<uuid>"}`

	err := unmarshalToolArguments(call.Input, &params)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	err = tdt.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		return tdt.deleter.Delete(ctx, uow, params.ID)
	})
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"delete_todo_error","details":"%s"}`, err.Error()),
		}
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      `{"message":"The todo was deleted successfully!"}`,
	}
}
