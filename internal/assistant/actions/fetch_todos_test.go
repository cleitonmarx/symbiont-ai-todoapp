package actions

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTodoFetcherAction(t *testing.T) {
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)
	testTodo := domain.Todo{
		ID:      uuid.New(),
		Title:   "Test Todo",
		DueDate: fixedTime,
		Status:  domain.TodoStatus_OPEN,
	}

	tests := map[string]struct {
		setupMocks func(
			*domain.MockTodoRepository,
			*domain.MockSemanticEncoder,
			*domain.MockCurrentTimeProvider,
		)
		functionCall domain.AssistantActionCall
		validateResp func(t *testing.T, resp domain.AssistantMessage)
	}{
		"fetch-todos-success": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Return([]domain.Todo{testTodo}, false, nil).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				assert.NotNil(t, output["todos"])
			},
		},
		"fetch-todos-with-status-and-similarity": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
				semanticEncoder.EXPECT().
					VectorizeQuery(mock.Anything, "embedding-model", "urgent").
					Return(domain.EmbeddingVector{Vector: []float64{0.3, 0.4}}, nil).
					Once()

				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Run(func(ctx context.Context, page, pageSize int, opts ...domain.ListTodoOption) {
						param := domain.ListTodosParams{}
						for _, opt := range opts {
							opt(&param)
						}
						assert.Equal(t, domain.TodoStatus_OPEN, *param.Status)
						assert.Equal(t, []float64{0.3, 0.4}, param.Embedding)
					}).
					Return([]domain.Todo{testTodo}, false, nil).
					Once()

			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "status": "OPEN", "search_by_similarity": "urgent"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				assert.NotNil(t, output["todos"])
			},
		},
		"fetch-todos-by-title": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Run(func(ctx context.Context, page, pageSize int, opts ...domain.ListTodoOption) {
						param := domain.ListTodosParams{}
						for _, opt := range opts {
							opt(&param)
						}
						assert.NotNil(t, param.TitleContains)
						assert.Equal(t, "report", *param.TitleContains)
					}).
					Return([]domain.Todo{testTodo}, false, nil).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "search_by_title": "report"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				assert.NotNil(t, output["todos"])
			},
		},
		"fetch-todos-with-sortby": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {

				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Run(func(ctx context.Context, page, pageSize int, opts ...domain.ListTodoOption) {
						param := domain.ListTodosParams{}
						for _, opt := range opts {
							opt(&param)
						}
						assert.Equal(t, &domain.TodoSortBy{Field: "duedate", Direction: "ASC"}, param.SortBy)
					}).
					Return([]domain.Todo{}, false, nil)
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "sort_by": "duedateAsc"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				todos, ok := output["todos"].([]any)
				require.True(t, ok)
				assert.Len(t, todos, 0)
			},
		},
		"fetch-todos-with-due-date-filters": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()

				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Run(func(ctx context.Context, page, pageSize int, opts ...domain.ListTodoOption) {
						param := domain.ListTodosParams{}
						for _, opt := range opts {
							opt(&param)
						}
						expectedDueAfter, _ := time.Parse("2006-01-02", "2026-01-20")
						expectedDueBefore, _ := time.Parse("2006-01-02", "2026-01-30")
						assert.Equal(t, expectedDueAfter, *param.DueAfter)
						assert.Equal(t, expectedDueBefore, *param.DueBefore)
					}).
					Return([]domain.Todo{testTodo}, false, nil).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "due_after": "2026-01-20", "due_before": "2026-01-30"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				assert.NotNil(t, output["todos"])
			},
		},
		"fetch-todos-invalid-due-after": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {

				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "due_after": "invalid-date", "due_before": "2026-01-30"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_after")
			},
		},
		"fetch-todos-invalid-due-before": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {

				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "due_after": "2026-01-20", "due_before": "invalid-date"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_before")
			},
		},

		"fetch-todos-invalid-arguments": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `invalid json`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"fetch-todos-invalid-status-with-concatenated-field": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page":1,"page_size":10,"search_by_similarity":"abul dinner","sort_by":"similarityAsc","status":"OPEN','sort_by':'dueDateAsc"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_filters")
				assert.Contains(t, resp.Content, "status must be either OPEN or DONE")
			},
		},
		"fetch-todos-invalid-partial-due-range": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "due_after": "2026-01-20"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_range")
			},
		},
		"fetch-todos-invalid-due-range-order": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
				timeProvider.EXPECT().
					Now().
					Return(fixedTime).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "due_after": "2026-01-30", "due_before": "2026-01-20"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_range")
				assert.Contains(t, resp.Content, "due_after must be less than or equal to due_before")
			},
		},
		"fetch-todos-similarity-sort-without-search-term": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "sort_by": "similarityAsc"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "missing_search_by_similarity_for_similarity_sort")
			},
		},
		"fetch-todos-embedding-error": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
				semanticEncoder.EXPECT().
					VectorizeQuery(mock.Anything, "embedding-model", "search").
					Return(domain.EmbeddingVector{}, errors.New("embedding failed")).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "search_by_similarity": "search"}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "embedding_error")
			},
		},
		"fetch-todos-list-error": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
				todoRepo.EXPECT().
					ListTodos(mock.Anything, 1, 10, mock.Anything).
					Return(nil, false, errors.New("db error")).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "list_todos_error")
			},
		},
		"fetch-todos-has-more": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
				todoRepo.EXPECT().
					ListTodos(mock.Anything, 1, 10, mock.Anything).
					Return([]domain.Todo{testTodo}, true, nil).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				assert.Equal(t, float64(2), output["next_page"])
			},
		},
		"fetch-todos-no-results": {
			setupMocks: func(todoRepo *domain.MockTodoRepository, semanticEncoder *domain.MockSemanticEncoder, timeProvider *domain.MockCurrentTimeProvider) {
				todoRepo.EXPECT().
					ListTodos(mock.Anything, 1, 10, mock.Anything).
					Return([]domain.Todo{}, false, nil).
					Once()
			},
			functionCall: domain.AssistantActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10}`,
			},
			validateResp: func(t *testing.T, resp domain.AssistantMessage) {
				assert.Equal(t, domain.ChatRole_Tool, resp.Role)
				var output map[string]any
				err := json.Unmarshal([]byte(resp.Content), &output)
				require.NoError(t, err)
				todos, ok := output["todos"].([]any)
				require.True(t, ok)
				assert.Len(t, todos, 0)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			todoRepo := domain.NewMockTodoRepository(t)
			semanticEncoder := domain.NewMockSemanticEncoder(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			tt.setupMocks(todoRepo, semanticEncoder, timeProvider)

			tool := NewTodoFetcherAction(todoRepo, semanticEncoder, timeProvider, "embedding-model")

			resp := tool.Execute(context.Background(), tt.functionCall, []domain.AssistantMessage{})
			tt.validateResp(t, resp)
		})
	}
}
