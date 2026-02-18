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

// StatusMessage returns a status message about the tool execution.
func (t TodoCreatorAction) StatusMessage() string {
	return "📝 Creating your todo..."
}

// Definition returns the assistant action definition for TodoCreatorAction.
func (tct TodoCreatorAction) Definition() domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name:        "create_todo",
		Description: "Create exactly one todo. Required keys: title (string) and due_date (YYYY-MM-DD). No extra keys. For batch creation requests, call this tool once per task until all tasks are saved. Valid: {\"title\":\"Pay rent\",\"due_date\":\"2026-04-30\"}. Invalid: {\"title\":\"Pay rent\",\"due\":\"tomorrow\",\"priority\":\"high\"}.",
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

	err := unmarshalToolArguments(call.Input, &params)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	now := tct.timeProvider.Now()
	dueDate, found := extractDateParam(params.DueDate, conversationHistory, now)
	if !found {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"invalid_due_date","details":"Due date cannot be empty. ISO 8601 string is required.", "example":%s}`, exampleArgs),
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
			Content:      fmt.Sprintf(`{"error":"create_todo_error","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      fmt.Sprintf(`{"message":"Your todo was created successfully! todo: {"id":"%s", "title":"%s", "due_date":"%s", "status":"%s"}"}`, todo.ID, todo.Title, todo.DueDate.Format(time.DateOnly), todo.Status),
	}
}
