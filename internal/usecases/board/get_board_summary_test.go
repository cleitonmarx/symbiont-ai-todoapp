package board

import (
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetBoardSummaryImpl_Query(t *testing.T) {
	t.Parallel()

	fixedUUID := func() uuid.UUID {
		return uuid.MustParse("223e4567-e89b-12d3-a456-426614174000")
	}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	boardSummary := todo.BoardSummary{
		ID: fixedUUID(),
		Content: todo.BoardSummaryContent{
			Counts: todo.StatusCounts{
				Open: 3,
				Done: 5,
			},
			NextUp: []todo.NextUpItem{
				{
					Title:  "Review project proposal",
					Reason: "Due tomorrow",
				},
				{
					Title:  "Submit report",
					Reason: "Overdue by 2 days",
				},
			},
			Overdue: []string{
				"Submit report",
				"Update documentation",
			},
			NearDeadline: []string{
				"Review project proposal",
			},
			Summary: "You have 2 overdue tasks and 1 task due tomorrow.",
		},
		Model:         "mistral",
		GeneratedAt:   fixedTime,
		SourceVersion: 1,
	}

	tests := map[string]struct {
		setExpectations func(summaryRepo *todo.MockBoardSummaryRepository)
		expectedSummary todo.BoardSummary
		expectedErr     error
	}{
		"success": {
			setExpectations: func(summaryRepo *todo.MockBoardSummaryRepository) {
				summaryRepo.EXPECT().GetLatestSummary(
					mock.Anything,
				).Return(boardSummary, true, nil)
			},
			expectedSummary: boardSummary,
			expectedErr:     nil,
		},
		"repository-error": {
			setExpectations: func(summaryRepo *todo.MockBoardSummaryRepository) {
				summaryRepo.EXPECT().GetLatestSummary(
					mock.Anything,
				).Return(todo.BoardSummary{}, false, errors.New("database error"))
			},
			expectedSummary: todo.BoardSummary{},
			expectedErr:     errors.New("database error"),
		},
		"no-summary-found": {
			setExpectations: func(summaryRepo *todo.MockBoardSummaryRepository) {
				summaryRepo.EXPECT().GetLatestSummary(
					mock.Anything,
				).Return(todo.BoardSummary{}, false, nil)
			},
			expectedSummary: todo.BoardSummary{},
			expectedErr:     core.NewNotFoundErr("board summary not found"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			summaryRepo := todo.NewMockBoardSummaryRepository(t)

			if tt.setExpectations != nil {
				tt.setExpectations(summaryRepo)
			}

			gbs := NewGetBoardSummaryImpl(summaryRepo)

			got, gotErr := gbs.Query(t.Context())
			assert.Equal(t, tt.expectedErr, gotErr)
			assert.Equal(t, tt.expectedSummary.ID, got.ID)
			assert.Equal(t, tt.expectedSummary.Content.Counts.Open, got.Content.Counts.Open)
			assert.Equal(t, tt.expectedSummary.Content.Counts.Done, got.Content.Counts.Done)
			assert.Equal(t, tt.expectedSummary.Content.Summary, got.Content.Summary)

		})
	}
}
