package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
)

// BulkTodoCreatorAction is an assistant action for creating multiple todos.
type BulkTodoCreatorAction struct {
	uow          domain.UnitOfWork
	creator      usecases.TodoCreator
	timeProvider domain.CurrentTimeProvider
}

// NewBulkTodoCreatorAction creates a new instance of BulkTodoCreatorAction.
func NewBulkTodoCreatorAction(uow domain.UnitOfWork, creator usecases.TodoCreator, timeProvider domain.CurrentTimeProvider) BulkTodoCreatorAction {
	return BulkTodoCreatorAction{
		uow:          uow,
		creator:      creator,
		timeProvider: timeProvider,
	}
}

// StatusMessage returns a status message about the action execution.
func (a BulkTodoCreatorAction) StatusMessage() string {
	return "📝 Creating your todos..."
}

// Definition returns the assistant action definition for BulkTodoCreatorAction.
func (a BulkTodoCreatorAction) Definition() domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name:        "create_todos",
		Description: "Create multiple todo items in one call (batch).",
		Hints: domain.AssistantActionHints{
			UseWhen: "Use for multi-task creation requests: tasks (plural), checklist, roadmap, steps, or plan for a goal (e.g., plan a trip).\n" +
				"- If user gives a title prefix, apply it to every created task title.\n" +
				"- If user gives a target date/window, distribute due dates chronologically up to that date.",
			AvoidWhen: "Do not use for single-task creation, updates, or deletes.",
			ArgRules:  "Required key: todos. Each item requires title and due_date (YYYY-MM-DD).",
		},
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"todos": {
					Type:        "array",
					Description: "List of todos to create. Each item: {title, due_date}. REQUIRED.",
					Required:    true,
					Items: &domain.AssistantActionField{
						Type:        "object",
						Description: "Todo item to create.",
						Fields: map[string]domain.AssistantActionField{
							"title": {
								Type:        "string",
								Description: "Title of the todo. REQUIRED.",
								Required:    true,
							},
							"due_date": {
								Type:        "string",
								Description: "Due date in YYYY-MM-DD format. REQUIRED.",
								Required:    true,
								Format:      "date",
							},
						},
					},
				},
			},
		},
	}
}

// Execute executes BulkTodoCreatorAction.
func (a BulkTodoCreatorAction) Execute(ctx context.Context, call domain.AssistantActionCall, conversationHistory []domain.AssistantMessage) domain.AssistantMessage {
	params := struct {
		Todos []struct {
			Title   string `json:"title"`
			DueDate string `json:"due_date"`
		} `json:"todos"`
	}{}
	exampleArgs := `{"todos":[{"title":"Pay rent","due_date":"2026-04-30"},{"title":"Buy groceries","due_date":"2026-05-01"}]}`

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

	now := a.timeProvider.Now()
	type createItem struct {
		Title   string
		DueDate time.Time
	}
	items := make([]createItem, 0, len(params.Todos))
	for i, todo := range params.Todos {
		title := strings.TrimSpace(todo.Title)
		if title == "" {
			return domain.AssistantMessage{
				Role:         domain.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_title", fmt.Sprintf("todo at index %d has an empty title.", i), exampleArgs),
			}
		}

		dueDate, found := extractDateParam(todo.DueDate, conversationHistory, now)
		if !found {
			return domain.AssistantMessage{
				Role:         domain.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_due_date", fmt.Sprintf("todo at index %d has invalid due_date.", i), exampleArgs),
			}
		}

		items = append(items, createItem{Title: title, DueDate: dueDate})
	}

	todos := make([]domain.Todo, 0, len(items))
	err = a.uow.Execute(ctx, func(uowCtx context.Context, uow domain.UnitOfWork) error {
		for i, item := range items {
			todo, createErr := a.creator.Create(uowCtx, uow, item.Title, item.DueDate)
			if createErr != nil {
				return fmt.Errorf("todo at index %d: %w", i, createErr)
			}
			todos = append(todos, todo)
		}
		return nil
	})
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("create_todos_error", err.Error(), exampleArgs),
		}
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      formatTodosRows(todos),
	}
}
