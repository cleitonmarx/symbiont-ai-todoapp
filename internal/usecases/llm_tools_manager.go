package usecases

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LLMToolManager manages a collection of LLM tools.
type LLMToolManager struct {
	tools map[string]domain.LLMTool
}

// NewLLMToolManager creates a new LLMToolManager with the provided tools.
func NewLLMToolManager(tools ...domain.LLMTool) LLMToolManager {
	toolMap := make(map[string]domain.LLMTool)
	for _, tool := range tools {
		toolMap[tool.Definition().Function.Name] = tool
	}
	return LLMToolManager{
		tools: toolMap,
	}
}

// StatusMessage returns a status message about the tool execution.
func (m LLMToolManager) StatusMessage(functionName string) string {
	if tool, ok := m.tools[functionName]; ok {
		if msg := tool.StatusMessage(); msg != "" {
			return msg
		}
	}
	return "⏳ Processing request..."
}

// List returns all registered LLM tools.
func (ctr LLMToolManager) List() []domain.LLMToolDefinition {
	toolList := make([]domain.LLMToolDefinition, 0, len(ctr.tools))
	for _, tool := range ctr.tools {
		toolList = append(toolList, tool.Definition())
	}
	return toolList
}

// Call invokes the appropriate tool based on the function call.
func (ctr LLMToolManager) Call(ctx context.Context, call domain.LLMStreamEventToolCall, conversationHistory []domain.LLMChatMessage) domain.LLMChatMessage {
	spanCtx, span := telemetry.Start(ctx,
		trace.WithAttributes(
			attribute.String("tool.function", call.Function),
		),
	)
	defer span.End()
	tool, exists := ctr.tools[call.Function]
	if !exists {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"unknown_tool","details":"Tool '%s' is not registered."}`, call.Function),
		}
	}
	return tool.Call(spanCtx, call, conversationHistory)
}

type InitLLMToolRegistry struct {
	Uow            domain.UnitOfWork          `resolve:""`
	TodoCreator    TodoCreator                `resolve:""`
	TodoUpdater    TodoUpdater                `resolve:""`
	TodoDeleter    TodoDeleter                `resolve:""`
	TodoRepo       domain.TodoRepository      `resolve:""`
	LLMClient      domain.LLMClient           `resolve:""`
	TimeProvider   domain.CurrentTimeProvider `resolve:""`
	EmbeddingModel string                     `config:"LLM_EMBEDDING_MODEL"`
}

func (i InitLLMToolRegistry) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.LLMToolRegistry](NewLLMToolManager(
		NewUIFiltersSetterTool(),
		NewTodoFetcherTool(
			i.TodoRepo,
			i.LLMClient,
			i.TimeProvider,
			i.EmbeddingModel,
		),
		NewTodoCreatorTool(
			i.Uow,
			i.TodoCreator,
			i.TimeProvider,
		),
		NewTodoUpdaterTool(
			i.Uow,
			i.TodoUpdater,
		),
		NewTodoDueDateUpdaterTool(
			i.Uow,
			i.TodoUpdater,
			i.TimeProvider,
		),
		NewTodoDeleterTool(
			i.Uow,
			i.TodoDeleter,
		),
	))
	return ctx, nil
}
