package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFetchTodosAction(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)
	testTodo := todo.Todo{
		ID:      uuid.New(),
		Title:   "Test Todo",
		DueDate: fixedTime,
		Status:  todo.Status_OPEN,
	}

	tests := map[string]struct {
		setupMocks func(
			*todo.MockRepository,
			*semantic.MockEncoder,
		)
		functionCall assistant.ActionCall
		validateResp func(t *testing.T, resp assistant.Message)
	}{
		"fetch-todos-success": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Return([]todo.Todo{testTodo}, false, nil).
					Once()
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "todos[1]{id,title,due_date,status}:")
			},
		},
		"fetch-todos-with-status-and-similarity": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
				semanticEncoder.EXPECT().
					VectorizeQuery(mock.Anything, "embedding-model", "urgent").
					Return(semantic.EmbeddingVector{Vector: []float64{0.3, 0.4}}, nil).
					Once()

				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Run(func(ctx context.Context, page, pageSize int, opts ...todo.ListOption) {
						param := todo.ListParams{}
						for _, opt := range opts {
							opt(&param)
						}
						assert.Equal(t, todo.Status_OPEN, *param.Status)
						assert.Equal(t, []float64{0.3, 0.4}, param.Embedding)
					}).
					Return([]todo.Todo{testTodo}, false, nil).
					Once()

			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "status": "OPEN", "search_by_similarity": "urgent"}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "todos[1]{id,title,due_date,status}:")
			},
		},
		"fetch-todos-by-title": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Run(func(ctx context.Context, page, pageSize int, opts ...todo.ListOption) {
						param := todo.ListParams{}
						for _, opt := range opts {
							opt(&param)
						}
						assert.NotNil(t, param.TitleContains)
						assert.Equal(t, "report", *param.TitleContains)
					}).
					Return([]todo.Todo{testTodo}, false, nil).
					Once()
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "search_by_title": "report"}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "todos[1]{id,title,due_date,status}:")
			},
		},
		"fetch-todos-with-sortby": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Run(func(ctx context.Context, page, pageSize int, opts ...todo.ListOption) {
						param := todo.ListParams{}
						for _, opt := range opts {
							opt(&param)
						}
						assert.Equal(t, &todo.SortBy{Field: "dueDate", Direction: "ASC"}, param.SortBy)
					}).
					Return([]todo.Todo{}, false, nil)
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "sort_by": "dueDateAsc"}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "todos[0]:")
			},
		},
		"fetch-todos-with-due-date-filters": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
				todoRepo.EXPECT().
					ListTodos(
						mock.Anything,
						1,
						10,
						mock.Anything,
					).
					Run(func(ctx context.Context, page, pageSize int, opts ...todo.ListOption) {
						param := todo.ListParams{}
						for _, opt := range opts {
							opt(&param)
						}
						expectedDueAfter, _ := time.Parse("2006-01-02", "2026-01-20")
						expectedDueBefore, _ := time.Parse("2006-01-02", "2026-01-30")
						assert.Equal(t, expectedDueAfter, *param.DueAfter)
						assert.Equal(t, expectedDueBefore, *param.DueBefore)
					}).
					Return([]todo.Todo{testTodo}, false, nil).
					Once()
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "due_after": "2026-01-20", "due_before": "2026-01-30"}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "todos[1]{id,title,due_date,status}:")
			},
		},
		"fetch-todos-invalid-due-after": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "due_after": "invalid-date", "due_before": "2026-01-30"}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_after")
			},
		},
		"fetch-todos-invalid-due-before": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "due_after": "2026-01-20", "due_before": "invalid-date"}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_before")
			},
		},

		"fetch-todos-invalid-arguments": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `invalid json`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_arguments")
			},
		},
		"fetch-todos-invalid-status-with-concatenated-field": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page":1,"page_size":10,"search_by_similarity":"abul dinner","sort_by":"similarityAsc","status":"OPEN','sort_by':'dueDateAsc"}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_status")
				assert.Contains(t, resp.Content, "status must be either OPEN or DONE")
			},
		},
		"fetch-todos-invalid-partial-due-range": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "due_after": "2026-01-20"}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_range")
			},
		},
		"fetch-todos-invalid-due-range-order": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "due_after": "2026-01-30", "due_before": "2026-01-20"}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "invalid_due_range")
				assert.Contains(t, resp.Content, "due_after must be less than or equal to due_before")
			},
		},
		"fetch-todos-similarity-sort-without-search-term": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "sort_by": "similarityAsc"}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "missing_search_by_similarity_for_similarity_sort")
			},
		},
		"fetch-todos-embedding-error": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
				semanticEncoder.EXPECT().
					VectorizeQuery(mock.Anything, "embedding-model", "search").
					Return(semantic.EmbeddingVector{}, errors.New("embedding failed")).
					Once()
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10, "search_by_similarity": "search"}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "embedding_error")
			},
		},
		"fetch-todos-list-error": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
				todoRepo.EXPECT().
					ListTodos(mock.Anything, 1, 10, mock.Anything).
					Return(nil, false, errors.New("db error")).
					Once()
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "list_todos_error")
			},
		},
		"fetch-todos-has-more": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
				todoRepo.EXPECT().
					ListTodos(mock.Anything, 1, 10, mock.Anything).
					Return([]todo.Todo{testTodo}, true, nil).
					Once()
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "next_page: 2\ntodos[1]")
			},
		},
		"fetch-todos-no-results": {
			setupMocks: func(todoRepo *todo.MockRepository, semanticEncoder *semantic.MockEncoder) {
				todoRepo.EXPECT().
					ListTodos(mock.Anything, 1, 10, mock.Anything).
					Return([]todo.Todo{}, false, nil).
					Once()
			},
			functionCall: assistant.ActionCall{
				Name:  "fetch_todos",
				Input: `{"page": 1, "page_size": 10}`,
			},
			validateResp: func(t *testing.T, resp assistant.Message) {
				assert.Equal(t, assistant.ChatRole_Tool, resp.Role)
				assert.Contains(t, resp.Content, "todos[0]")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			todoRepo := todo.NewMockRepository(t)
			semanticEncoder := semantic.NewMockEncoder(t)
			tt.setupMocks(todoRepo, semanticEncoder)

			action := NewFetchTodosAction(todoRepo, semanticEncoder, "embedding-model")
			assert.NotEmpty(t, action.StatusMessage())

			definition := action.Definition()
			assert.Equal(t, "fetch_todos", definition.Name)
			assert.NotEmpty(t, definition.Description)
			assert.NotEmpty(t, definition.Input)

			resp := action.Execute(t.Context(), tt.functionCall, []assistant.Message{})
			tt.validateResp(t, resp)
		})
	}
}
