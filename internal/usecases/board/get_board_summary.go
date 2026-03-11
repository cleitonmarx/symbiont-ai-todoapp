package board

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// GetBoardSummary is a use case interface for retrieving a summary of the board.
type GetBoardSummary interface {
	Query(ctx context.Context) (todo.BoardSummary, error)
}

// GetBoardSummaryImpl is the implementation of the GetBoardSummary use case.
type GetBoardSummaryImpl struct {
	summaryRepo todo.BoardSummaryRepository
}

// NewGetBoardSummaryImpl creates a new instance of GetBoardSummaryImpl.
func NewGetBoardSummaryImpl(r todo.BoardSummaryRepository) GetBoardSummaryImpl {
	return GetBoardSummaryImpl{
		summaryRepo: r,
	}
}

// Query retrieves the latest board summary from the repository.
//
//	It returns an error if the summary is not found or if there is an issue with the repository.
func (gbs GetBoardSummaryImpl) Query(ctx context.Context) (todo.BoardSummary, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	summary, found, err := gbs.summaryRepo.GetLatestSummary(spanCtx)
	if telemetry.IsErrorRecorded(span, err) {
		return todo.BoardSummary{}, err
	}
	if !found {
		err := core.NewNotFoundErr("board summary not found")
		return todo.BoardSummary{}, err
	}

	return summary, nil
}
