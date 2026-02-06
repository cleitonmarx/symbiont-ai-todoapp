package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
)

type GetBoardSummary interface {
	Query(ctx context.Context) (domain.BoardSummary, error)
}

type GetBoardSummaryImpl struct {
	summaryRepo domain.BoardSummaryRepository
}

func NewGetBoardSummaryImpl(r domain.BoardSummaryRepository) GetBoardSummaryImpl {
	return GetBoardSummaryImpl{
		summaryRepo: r,
	}
}

func (gbs GetBoardSummaryImpl) Query(ctx context.Context) (domain.BoardSummary, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	summary, found, err := gbs.summaryRepo.GetLatestSummary(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return domain.BoardSummary{}, err
	}
	if !found {
		err := domain.NewNotFoundErr("board summary not found")
		return domain.BoardSummary{}, err
	}

	return summary, nil
}

type InitGetBoardSummary struct {
	SummaryRepo domain.BoardSummaryRepository `resolve:""`
}

func (igbs InitGetBoardSummary) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[GetBoardSummary](NewGetBoardSummaryImpl(igbs.SummaryRepo))

	return ctx, nil
}
