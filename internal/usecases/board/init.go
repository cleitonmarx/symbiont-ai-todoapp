package board

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitGenerateBoardSummary initializes the GenerateBoardSummary use case.
type InitGenerateBoardSummary struct {
	SummaryRepo  todo.BoardSummaryRepository `resolve:""`
	TimeProvider core.CurrentTimeProvider    `resolve:""`
	Assistant    assistant.Assistant         `resolve:""`
	Model        string                      `config:"LLM_SUMMARY_MODEL"`
}

// Initialize registers the GenerateBoardSummary use case in the dependency container.
func (igbs InitGenerateBoardSummary) Initialize(ctx context.Context) (context.Context, error) {
	queue, _ := depend.Resolve[CompletedBoardSummaryChannel]()
	depend.Register[GenerateBoardSummary](NewGenerateBoardSummaryImpl(
		igbs.SummaryRepo, igbs.TimeProvider, igbs.Assistant, igbs.Model, queue,
	))
	return ctx, nil
}

// InitGetBoardSummary initializes the GetBoardSummary use case.
type InitGetBoardSummary struct {
	SummaryRepo todo.BoardSummaryRepository `resolve:""`
}

// Initialize registers the GetBoardSummary use case in the dependency container.
func (igbs InitGetBoardSummary) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[GetBoardSummary](NewGetBoardSummaryImpl(igbs.SummaryRepo))
	return ctx, nil
}
