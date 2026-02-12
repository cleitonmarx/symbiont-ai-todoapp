package usecases

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGenerateBoardSummaryImpl_Execute(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	boardSummary := domain.BoardSummary{
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Content: domain.BoardSummaryContent{
			Counts: domain.TodoStatusCounts{
				Open: 2,
				Done: 1,
			},
			NextUp: []domain.NextUpTodoItem{
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
			*domain.MockBoardSummaryRepository,
			*domain.MockCurrentTimeProvider,
			*domain.MockLLMClient,
		)
		expectedErr error
	}{
		"success": {
			setExpectations: func(
				sr *domain.MockBoardSummaryRepository,
				tp *domain.MockCurrentTimeProvider,
				c *domain.MockLLMClient,
			) {

				tp.EXPECT().Now().Return(fixedTime)

				sr.EXPECT().CalculateSummaryContent(mock.Anything).
					Return(
						calculated,
						nil,
					)

				sr.EXPECT().GetLatestSummary(mock.Anything).
					Return(domain.BoardSummary{}, false, nil)

				c.EXPECT().Chat(
					mock.Anything,
					mock.MatchedBy(func(req domain.LLMChatRequest) bool {
						return req.Model == "mistral" &&
							len(req.Messages) == 2 &&
							req.Messages[0].Role == "developer" &&
							req.Messages[1].Role == "user" &&
							strings.Contains(req.Messages[0].Content, "You are a helpful assistant that summarizes todo progress") &&
							strings.Contains(req.Messages[1].Content, "Open: 2\n  Done: 1")
					}),
				).Return(domain.LLMChatResponse{Content: "You have 2 open todos, 1 overdue todo, and 1 completed todo."}, nil)

				sr.EXPECT().StoreSummary(
					mock.Anything,
					boardSummary,
				).Return(nil)
			},
			expectedErr: nil,
		},
		"llm-client-error": {
			setExpectations: func(
				sr *domain.MockBoardSummaryRepository,
				tp *domain.MockCurrentTimeProvider,
				c *domain.MockLLMClient,
			) {
				tp.EXPECT().Now().Return(fixedTime)

				sr.EXPECT().CalculateSummaryContent(mock.Anything).
					Return(
						calculated,
						nil,
					)

				sr.EXPECT().GetLatestSummary(mock.Anything).
					Return(domain.BoardSummary{}, false, nil)

				c.EXPECT().Chat(
					mock.Anything,
					mock.Anything,
				).Return(domain.LLMChatResponse{}, assert.AnError)
			},
			expectedErr: assert.AnError,
		},
		"store-summary-error": {
			setExpectations: func(
				sr *domain.MockBoardSummaryRepository,
				tp *domain.MockCurrentTimeProvider,
				c *domain.MockLLMClient,
			) {
				tp.EXPECT().Now().Return(fixedTime)

				sr.EXPECT().CalculateSummaryContent(mock.Anything).
					Return(
						calculated,
						nil,
					)

				sr.EXPECT().GetLatestSummary(mock.Anything).
					Return(domain.BoardSummary{}, false, nil)

				c.EXPECT().Chat(
					mock.Anything,
					mock.Anything,
				).Return(domain.LLMChatResponse{Content: "You have 2 open todos, 1 overdue todo, and 1 completed todo."}, nil)

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
			sr := domain.NewMockBoardSummaryRepository(t)
			tp := domain.NewMockCurrentTimeProvider(t)
			c := domain.NewMockLLMClient(t)

			if tt.setExpectations != nil {
				tt.setExpectations(sr, tp, c)
			}

			gbs := NewGenerateBoardSummaryImpl(sr, tp, c, "mistral", nil)

			err := gbs.Execute(context.Background())
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestInitGenerateBoardSummary_Initialize(t *testing.T) {
	igbs := InitGenerateBoardSummary{}

	ctx, err := igbs.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredGbs, err := depend.Resolve[GenerateBoardSummary]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredGbs)
}

func TestApplySummarySafetyGuards(t *testing.T) {
	tests := map[string]struct {
		content domain.BoardSummaryContent
		summary string
		want    string
	}{
		"strips-overdue-qualifiers": {
			content: domain.BoardSummaryContent{
				Overdue: []string{},
			},
			summary: "Great progress! Focus on the overdue chimney cleaning and late digital backups.",
			want:    "Great progress! Focus on the chimney cleaning and digital backups.",
		},
		"keeps-overdue-wording": {
			content: domain.BoardSummaryContent{
				Overdue: []string{"Schedule chimney cleaning"},
			},
			summary: "Focus on the overdue chimney cleaning.",
			want:    "Focus on the overdue chimney cleaning.",
		},
		"preserves-no-overdue-statements": {
			content: domain.BoardSummaryContent{
				Overdue: []string{},
			},
			summary: "Nice momentum, no overdue tasks right now.",
			want:    "Nice momentum, no overdue tasks right now.",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := applySummarySafetyGuards(tt.summary, tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}
