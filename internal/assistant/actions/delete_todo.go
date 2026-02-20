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

// StatusMessage returns a status message about the action execution.
func (t TodoDeleterAction) StatusMessage() string {
	return "🗑️ Deleting the todo..."
}

// Definition returns the assistant action definition for TodoDeleterAction.
func (tdt TodoDeleterAction) Definition() domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name:        "delete_todo",
		Description: "Delete one todo by id.",
		Hints: domain.AssistantActionHints{
			UseWhen: "Use to delete one known todo by id.\n" +
				"- For multiple deletes, execute repeated delete_todo calls with each todo id.",
			AvoidWhen: "Do not use when id is missing or ambiguous; fetch first.",
			ArgRules:  "Required key: id (UUID). No extra keys.",
		},
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

	err := unmarshalActionInput(call.Input, &params)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", fmt.Sprintf("Failed to parse action input: %s", err.Error()), exampleArgs),
		}
	}

	err = tdt.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		return tdt.deleter.Delete(ctx, uow, params.ID)
	})
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("delete_todo_error", fmt.Sprintf("Failed to delete todo: %s", err.Error()), exampleArgs),
		}
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      fmt.Sprintf("todos[1]{id,deleted}\n%s,true", params.ID),
	}
}
