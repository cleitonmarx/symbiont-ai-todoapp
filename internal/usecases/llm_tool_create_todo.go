package usecases

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

// TodoCreatorTool is an LLM tool for creating todos.
type TodoCreatorTool struct {
	uow          domain.UnitOfWork
	creator      TodoCreator
	timeProvider domain.CurrentTimeProvider
}

// NewTodoCreatorTool creates a new instance of TodoCreatorTool.
func NewTodoCreatorTool(uow domain.UnitOfWork, creator TodoCreator, timeProvider domain.CurrentTimeProvider) TodoCreatorTool {
	return TodoCreatorTool{
		uow:          uow,
		creator:      creator,
		timeProvider: timeProvider,
	}
}

// StatusMessage returns a status message about the tool execution.
func (t TodoCreatorTool) StatusMessage() string {
	return "📝 Creating your todo..."
}

// Tool returns the LLMTool definition for the TodoCreatorTool.
func (tct TodoCreatorTool) Definition() domain.LLMToolDefinition {
	return domain.LLMToolDefinition{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "create_todo",
			Description: "Create exactly one todo. Required keys: title (string) and due_date (YYYY-MM-DD). No extra keys. For batch creation requests, call this tool once per task until all tasks are saved. Valid: {\"title\":\"Pay rent\",\"due_date\":\"2026-04-30\"}. Invalid: {\"title\":\"Pay rent\",\"due\":\"tomorrow\",\"priority\":\"high\"}.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
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
		},
	}
}

// Call executes the TodoCreatorTool with the provided function call.
func (tct TodoCreatorTool) Call(ctx context.Context, call domain.LLMStreamEventToolCall, conversationHistory []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		Title   string `json:"title"`
		DueDate string `json:"due_date"`
	}{}

	exampleArgs := `{"title":"Pay rent","due_date":"2026-04-30"}`

	err := unmarshalToolArguments(call.Arguments, &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	now := tct.timeProvider.Now()
	dueDate, found := extractDateParam(params.DueDate, conversationHistory, now)
	if !found {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_due_date","details":"Due date cannot be empty. ISO 8601 string is required.", "example":%s}`, exampleArgs),
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
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"create_todo_error","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	return domain.LLMChatMessage{
		Role:       domain.ChatRole_Tool,
		ToolCallID: &call.ID,
		Content:    "Your todo was created successfully! Created todo: " + todo.ToLLMInput(),
	}
}
