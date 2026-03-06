package board

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGenerateBoardSummaryImpl_Execute(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	boardSummary := todo.BoardSummary{
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Content: todo.BoardSummaryContent{
			Counts: todo.StatusCounts{
				Open: 2,
				Done: 1,
			},
			NextUp: []todo.NextUpItem{
				{
					Title:  "Open task 1",
					Reason: "Due in 5 days",
				},
			},
			Overdue:      []string{"Todo task 2"},
			NearDeadline: []string{"Open task 3"},
			Summary:      "You have 2 open todos, 1 overdue todo, and 1 completed todo.",
		},
		Model:         "mistral",
		GeneratedAt:   fixedTime,
		SourceVersion: 1,
	}

	calculated := boardSummary.Content
	calculated.Summary = ""

	tests := map[string]struct {
		setExpectations func(
			*todo.MockBoardSummaryRepository,
			*core.MockCurrentTimeProvider,
			*assistant.MockAssistant,
		)
		expectedErr error
	}{
		"success": {
			setExpectations: func(
				sr *todo.MockBoardSummaryRepository,
				tp *core.MockCurrentTimeProvider,
				assist *assistant.MockAssistant,
			) {

				tp.EXPECT().Now().Return(fixedTime)

				sr.EXPECT().CalculateSummaryContent(mock.Anything).
					Return(
						calculated,
						nil,
					)

				sr.EXPECT().GetLatestSummary(mock.Anything).
					Return(todo.BoardSummary{}, false, nil)

				assist.EXPECT().RunTurnSync(
					mock.Anything,
					mock.MatchedBy(func(req assistant.TurnRequest) bool {
						return req.Model == "mistral" &&
							len(req.Messages) == 2 &&
							req.Messages[0].Role == "system" &&
							req.Messages[1].Role == "user" &&
							strings.Contains(req.Messages[0].Content, "You are a helpful assistant that summarizes todo progress") &&
							strings.Contains(req.Messages[1].Content, "Open: 2\n  Done: 1")
					}),
				).Return(assistant.TurnResponse{Content: "You have 2 open todos, 1 overdue todo, and 1 completed todo."}, nil)

				sr.EXPECT().StoreSummary(
					mock.Anything,
					boardSummary,
				).Return(nil)
			},
			expectedErr: nil,
		},
		"llm-client-error": {
			setExpectations: func(
				sr *todo.MockBoardSummaryRepository,
				tp *core.MockCurrentTimeProvider,
				assist *assistant.MockAssistant,
			) {
				tp.EXPECT().Now().Return(fixedTime)

				sr.EXPECT().CalculateSummaryContent(mock.Anything).
					Return(
						calculated,
						nil,
					)

				sr.EXPECT().GetLatestSummary(mock.Anything).
					Return(todo.BoardSummary{}, false, nil)

				assist.EXPECT().RunTurnSync(
					mock.Anything,
					mock.Anything,
				).Return(assistant.TurnResponse{}, assert.AnError)
			},
			expectedErr: assert.AnError,
		},
		"store-summary-error": {
			setExpectations: func(
				sr *todo.MockBoardSummaryRepository,
				tp *core.MockCurrentTimeProvider,
				assist *assistant.MockAssistant,
			) {
				tp.EXPECT().Now().Return(fixedTime)

				sr.EXPECT().CalculateSummaryContent(mock.Anything).
					Return(
						calculated,
						nil,
					)

				sr.EXPECT().GetLatestSummary(mock.Anything).
					Return(todo.BoardSummary{}, false, nil)

				assist.EXPECT().RunTurnSync(
					mock.Anything,
					mock.Anything,
				).Return(assistant.TurnResponse{Content: "You have 2 open todos, 1 overdue todo, and 1 completed todo."}, nil)

				sr.EXPECT().StoreSummary(
					mock.Anything,
					boardSummary,
				).Return(assert.AnError)
			},
			expectedErr: assert.AnError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sr := todo.NewMockBoardSummaryRepository(t)
			tp := core.NewMockCurrentTimeProvider(t)
			assist := assistant.NewMockAssistant(t)

			if tt.setExpectations != nil {
				tt.setExpectations(sr, tp, assist)
			}

			gbs := NewGenerateBoardSummaryImpl(sr, tp, assist, "mistral", nil)

			err := gbs.Execute(context.Background())
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
