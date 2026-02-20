package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
)

// TodoCreatorAction is an assistant action for creating todos.
type TodoCreatorAction struct {
	uow          domain.UnitOfWork
	creator      usecases.TodoCreator
	timeProvider domain.CurrentTimeProvider
}

// NewTodoCreatorAction creates a new instance of TodoCreatorAction.
func NewTodoCreatorAction(uow domain.UnitOfWork, creator usecases.TodoCreator, timeProvider domain.CurrentTimeProvider) TodoCreatorAction {
	return TodoCreatorAction{
		uow:          uow,
		creator:      creator,
		timeProvider: timeProvider,
	}
}

// StatusMessage returns a status message about the action execution.
func (t TodoCreatorAction) StatusMessage() string {
	return "📝 Creating your todo..."
}

// Definition returns the assistant action definition for TodoCreatorAction.
func (tct TodoCreatorAction) Definition() domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name:        "create_todo",
		Description: "Create one todo item.",
		Hints: domain.AssistantActionHints{
			UseWhen: "Use for creating todo items. For planning requests or multi-task creation, create all planned tasks, not one generic task.\n" +
				"- For multi-task creation with a title prefix, apply the prefix to every created task title.\n" +
				"- For multi-task creation with a final target date/window, distribute due dates chronologically within that window and keep final milestone on the target date.",
			AvoidWhen: "Do not use for updates or deletes.",
			ArgRules:  "Required keys: title and due_date (YYYY-MM-DD). One call per task in batch create. Never stop batch creation after the first success. If user gave a title prefix, include it in every created title.",
		},
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"title": {
					Type:        "string",
					Description: "Todo title. REQUIRED.",
					Required:    true,
				},
				"due_date": {
					Type:        "string",
					Description: "Due date. REQUIRED. Use YYYY-MM-DD.",
					Required:    true,
				},
			},
		},
	}
}

// Execute executes TodoCreatorAction.
func (tct TodoCreatorAction) Execute(ctx context.Context, call domain.AssistantActionCall, conversationHistory []domain.AssistantMessage) domain.AssistantMessage {
	params := struct {
		Title   string `json:"title"`
		DueDate string `json:"due_date"`
	}{}
	exampleArgs := `{"title":"Pay rent","due_date":"2026-04-30"}`
	err := unmarshalActionInput(call.Input, &params)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", err.Error(), exampleArgs),
		}
	}

	now := tct.timeProvider.Now()
	dueDate, found := extractDateParam(params.DueDate, conversationHistory, now)
	if !found {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_due_date", "Due date cannot be empty. ISO 8601 string is required.", exampleArgs),
		}
	}

	var todo domain.Todo
	err = tct.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		td, err := tct.creator.Create(ctx, uow, params.Title, dueDate)
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
			Content:      newActionError("create_todo_error", err.Error(), exampleArgs),
		}
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      fmt.Sprintf("todos[1]{id,title,due_date,status}\n%s,%s,%s,%s", todo.ID, todo.Title, todo.DueDate.Format(time.DateOnly), todo.Status),
	}
}
