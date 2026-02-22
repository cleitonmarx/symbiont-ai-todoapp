package actions

import (
	"context"
	"fmt"

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
		Description: "Delete multiple todos by id.",
		Hints: domain.AssistantActionHints{
			UseWhen:   "Use for batch/multiple delete when ids are already known.",
			AvoidWhen: "Do not use when ids are missing or ambiguous; fetch first.",
			ArgRules:  "Required key: ids (array of UUID strings).",
		},
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"ids": {
					Type:        "array",
					Description: "List of todo UUIDs to delete. REQUIRED.",
					Required:    true,
				},
			},
		},
	}
}

// Execute executes BulkTodoDeleterAction.
func (a BulkTodoDeleterAction) Execute(ctx context.Context, call domain.AssistantActionCall, _ []domain.AssistantMessage) domain.AssistantMessage {
	params := struct {
		IDs []string `json:"ids"`
	}{}
	exampleArgs := `{"ids":["<uuid>","<uuid>"]}`

	err := unmarshalActionInput(call.Input, &params)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", err.Error(), exampleArgs),
		}
	}
	if len(params.IDs) == 0 {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", "ids must not be empty.", exampleArgs),
		}
	}

	ids := make([]uuid.UUID, 0, len(params.IDs))
	for i, id := range params.IDs {
		todoID, parseErr := uuid.Parse(id)
		if parseErr != nil {
			return domain.AssistantMessage{
				Role:         domain.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_todo_id", fmt.Sprintf("id at index %d is invalid: %s", i, parseErr.Error()), exampleArgs),
			}
		}
		ids = append(ids, todoID)
	}

	err = a.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		for i, id := range ids {
			deleteErr := a.deleter.Delete(ctx, uow, id)
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
		Content:      formatDeletedRows(params.IDs),
	}
}
