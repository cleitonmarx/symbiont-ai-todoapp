package usecases

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	domain_mocks "github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGenerateBoardSummaryImpl_Execute(t *testing.T) {
	fixedUUID := func() uuid.UUID {
		return uuid.MustParse("223e4567-e89b-12d3-a456-426614174000")
	}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	todos := []domain.Todo{
		{
			ID:        fixedUUID(),
			Title:     "Open task 1",
			Status:    domain.TodoStatus_OPEN,
			DueDate:   fixedTime.AddDate(0, 0, 5),
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		},
		{
			ID:        uuid.MustParse("323e4567-e89b-12d3-a456-426614174000"),
			Title:     "Done task 1",
			Status:    domain.TodoStatus_DONE,
			DueDate:   fixedTime.AddDate(0, 0, -1),
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		},
	}

	boardSummary := domain.BoardSummary{
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Content: domain.BoardSummaryContent{
			Counts: domain.TodoStatusCounts{
				Open: 1,
				Done: 1,
			},
			NextUp: []domain.NextUpTodoItem{
				{
					Title:  "Open task 1",
					Reason: "Due in 5 days",
				},
			},
			Overdue:      []string{},
			NearDeadline: []string{"Done task 1"},
			Summary:      "You have 1 open todo and 1 completed todo.",
		},
		Model:         "mistral",
		GeneratedAt:   fixedTime,
		SourceVersion: 1,
	}

	tests := map[string]struct {
		setExpectations func(
			*domain_mocks.MockBoardSummaryRepository,
			*domain_mocks.MockTodoRepository,
			*domain_mocks.MockCurrentTimeProvider,
			*domain_mocks.MockLLMClient,
		)
		expectedErr error
	}{
		"success": {
			setExpectations: func(
				sr *domain_mocks.MockBoardSummaryRepository,
				td *domain_mocks.MockTodoRepository,
				tp *domain_mocks.MockCurrentTimeProvider,
				c *domain_mocks.MockLLMClient,
			) {
				td.EXPECT().ListTodos(
					mock.Anything,
					1,
					1000,
				).Return(todos, false, nil)

				tp.EXPECT().Now().Return(fixedTime)

				c.EXPECT().Chat(
					mock.Anything,
					mock.MatchedBy(func(req domain.LLMChatRequest) bool {
						return req.Model == "mistral" &&
							len(req.Messages) == 2 &&
							req.Messages[0].Role == "system" &&
							req.Messages[1].Role == "user" &&
							strings.Contains(req.Messages[0].Content, "You are a JSON-only processor") &&
							strings.Contains(req.Messages[1].Content, "Open task 1")
					}),
				).Return(`{
					"counts": { "OPEN": 1, "DONE": 1 },
					"next_up": [ { "title": "Open task 1", "reason": "Due in 5 days" } ],
					"overdue": [],
					"near_deadline": [ "Done task 1" ],
					"summary": "You have 1 open todo and 1 completed todo."
				}`, nil)

				sr.EXPECT().StoreSummary(
					mock.Anything,
					boardSummary,
				).Return(nil)
			},
			expectedErr: nil,
		},
		"list-todos-error": {
			setExpectations: func(
				sr *domain_mocks.MockBoardSummaryRepository,
				td *domain_mocks.MockTodoRepository,
				tp *domain_mocks.MockCurrentTimeProvider,
				c *domain_mocks.MockLLMClient,
			) {
				td.EXPECT().ListTodos(
					mock.Anything,
					1,
					1000,
				).Return(nil, false, assert.AnError)
			},
			expectedErr: assert.AnError,
		},
		"llm-client-error": {
			setExpectations: func(
				sr *domain_mocks.MockBoardSummaryRepository,
				td *domain_mocks.MockTodoRepository,
				tp *domain_mocks.MockCurrentTimeProvider,
				c *domain_mocks.MockLLMClient,
			) {
				td.EXPECT().ListTodos(
					mock.Anything,
					1,
					1000,
				).Return(todos, false, nil)

				tp.EXPECT().Now().Return(fixedTime)

				c.EXPECT().Chat(
					mock.Anything,
					mock.Anything,
				).Return("", assert.AnError)
			},
			expectedErr: assert.AnError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sr := domain_mocks.NewMockBoardSummaryRepository(t)
			td := domain_mocks.NewMockTodoRepository(t)
			tp := domain_mocks.NewMockCurrentTimeProvider(t)
			c := domain_mocks.NewMockLLMClient(t)

			if tt.setExpectations != nil {
				tt.setExpectations(sr, td, tp, c)
			}

			gbs := NewGenerateBoardSummaryImpl(sr, td, tp, c, "mistral", nil)

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
