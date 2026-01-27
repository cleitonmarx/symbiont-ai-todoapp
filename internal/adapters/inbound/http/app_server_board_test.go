package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoAppServer_GetBoardSummary(t *testing.T) {
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	generatedAt := time.Date(2026, 1, 22, 10, 30, 0, 0, time.UTC)

	tests := map[string]struct {
		setupMocks     func(*mocks.MockGetBoardSummary)
		expectedStatus int
		expectedBody   *gen.BoardSummary
		expectedError  *gen.ErrorResp
	}{
		"success": {
			setupMocks: func(m *mocks.MockGetBoardSummary) {
				m.EXPECT().Query(mock.Anything).Return(domain.BoardSummary{
					ID:            fixedUUID,
					Model:         "ai/gpt-oss:latest",
					GeneratedAt:   generatedAt,
					SourceVersion: 1,
					Content: domain.BoardSummaryContent{
						Counts: domain.TodoStatusCounts{
							Open: 5,
							Done: 3,
						},
						NextUp: []domain.NextUpTodoItem{
							{
								Title:  "Buy groceries",
								Reason: "Due tomorrow",
							},
						},
						Overdue:      []string{"Pay electricity bill"},
						NearDeadline: []string{"Renew car insurance"},
						Summary:      "You have 1 overdue task and 2 tasks due this week.",
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.BoardSummary{
				Counts: gen.TodoStatusCounts{
					OPEN: 5,
					DONE: 3,
				},
				NextUp: []gen.NextUpTodoItem{
					{
						Title:  "Buy groceries",
						Reason: "Due tomorrow",
					},
				},
				Overdue:      []string{"Pay electricity bill"},
				NearDeadline: []string{"Renew car insurance"},
				Summary:      "You have 1 overdue task and 2 tasks due this week.",
			},
		},
		"summary-not-found": {
			setupMocks: func(m *mocks.MockGetBoardSummary) {
				m.EXPECT().
					Query(mock.Anything).
					Return(domain.BoardSummary{}, domain.NewNotFoundErr("board summary not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError: &gen.ErrorResp{
				Error: gen.Error{
					Code:    gen.NOTFOUND,
					Message: "board summary not found",
				},
			},
		},
		"use-case-error": {
			setupMocks: func(m *mocks.MockGetBoardSummary) {
				m.EXPECT().
					Query(mock.Anything).
					Return(domain.BoardSummary{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError: &gen.ErrorResp{
				Error: gen.Error{
					Code:    gen.INTERNALERROR,
					Message: "internal server error",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockGetBoardSummary := mocks.NewMockGetBoardSummary(t)
			tt.setupMocks(mockGetBoardSummary)

			server := &TodoAppServer{
				GetBoardSummaryUseCase: mockGetBoardSummary,
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/board/summary", nil)
			w := httptest.NewRecorder()

			server.GetBoardSummary(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response gen.BoardSummary
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedBody, response)
			}

			if tt.expectedError != nil {
				var response gen.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError.Error, response.Error)
			}

			mockGetBoardSummary.AssertExpectations(t)
		})
	}
}
