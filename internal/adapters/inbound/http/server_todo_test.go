package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	dueDate    = time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC)
	domainTodo = domain.Todo{
		ID:        uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Title:     "Buy groceries",
		Status:    domain.TodoStatus_DONE,
		DueDate:   dueDate,
		CreatedAt: time.Date(2026, 1, 22, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 22, 10, 30, 0, 0, time.UTC),
	}
	restTodo = gen.Todo{
		Id:        openapi_types.UUID(domainTodo.ID),
		Title:     domainTodo.Title,
		Status:    gen.DONE,
		DueDate:   openapi_types.Date{Time: domainTodo.DueDate},
		CreatedAt: domainTodo.CreatedAt,
		UpdatedAt: domainTodo.UpdatedAt,
	}
)

func TestTodoAppServer_CreateTodo(t *testing.T) {
	tests := map[string]struct {
		requestBody    []byte
		setupUsecases  func(*usecases.MockCreateTodo)
		expectedStatus int
		expectedBody   *gen.Todo
		expectedError  *gen.ErrorResp
	}{
		"success": {
			requestBody: serializeJSON(t, gen.CreateTodoJSONRequestBody{
				Title:   "Buy groceries",
				DueDate: openapi_types.Date{Time: dueDate},
			}),
			setupUsecases: func(m *usecases.MockCreateTodo) {
				m.EXPECT().
					Execute(mock.Anything, "Buy groceries", dueDate).Return(domainTodo, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   &restTodo,
		},
		"bad-request": {
			requestBody: serializeJSON(t, gen.CreateTodoJSONRequestBody{
				DueDate: openapi_types.Date{Time: dueDate},
			}),
			setupUsecases: func(m *usecases.MockCreateTodo) {
				m.EXPECT().
					Execute(mock.Anything, "", dueDate).
					Return(domain.Todo{}, domain.NewValidationErr("title is required"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError: &gen.ErrorResp{
				Error: gen.Error{
					Code:    gen.BADREQUEST,
					Message: "title is required",
				},
			},
		},
		"invalid-json-body": {
			requestBody:    []byte(`{"title": "Test todo", "due_date": "invalid-date"}`),
			setupUsecases:  func(m *usecases.MockCreateTodo) {},
			expectedStatus: http.StatusBadRequest,
			expectedError: &gen.ErrorResp{
				Error: gen.Error{
					Code:    gen.BADREQUEST,
					Message: "invalid request body: parsing time \"invalid-date\" as \"2006-01-02\": cannot parse \"invalid-date\" as \"2006\"",
				},
			},
		},
		"internal-server-error": {
			requestBody: serializeJSON(t, gen.CreateTodoJSONRequestBody{
				Title:   "Test todo",
				DueDate: openapi_types.Date{Time: time.Time{}},
			}),
			setupUsecases: func(m *usecases.MockCreateTodo) {
				m.EXPECT().
					Execute(mock.Anything, "Test todo", time.Time{}).
					Return(domain.Todo{}, errors.New("database error"))
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
			mockCreateTodo := usecases.NewMockCreateTodo(t)
			if tt.setupUsecases != nil {
				tt.setupUsecases(mockCreateTodo)
			}

			server := &TodoAppServer{
				CreateTodoUseCase: mockCreateTodo,
				Logger:            log.New(io.Discard, "", 0),
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			gen.Handler(server).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response gen.Todo
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody.Id, response.Id)
				assert.Equal(t, tt.expectedBody.Title, response.Title)
				assert.Equal(t, tt.expectedBody.Status, response.Status)
			}

			if tt.expectedError != nil {
				var response gen.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError.Error, response.Error)
			}

			mockCreateTodo.AssertExpectations(t)
		})
	}
}

func TestTodoAppServer_ListTodos(t *testing.T) {
	tests := map[string]struct {
		page            int
		pageSize        int
		todoStatus      *gen.TodoStatus
		query           *string
		dateRange       *gen.DateRange
		sortBy          *string
		setExpectations func(*usecases.MockListTodos)
		expectedStatus  int
		expectedBody    *gen.ListTodosResp
		expectedError   *gen.ErrorResp
	}{
		"success-with-todos": {
			page:     1,
			pageSize: 1,
			setExpectations: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 1, mock.Anything).
					Return([]domain.Todo{domainTodo}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ListTodosResp{
				Items: []gen.Todo{toTodo(domainTodo)},
				Page:  1,
			},
		},
		"success-with-no-todos": {
			page:     1,
			pageSize: 1,
			setExpectations: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 1, mock.Anything).
					Return([]domain.Todo{}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ListTodosResp{
				Items: []gen.Todo{},
				Page:  1,
			},
		},
		"success-with-next-and-previous-page": {
			page:     2,
			pageSize: 1,
			setExpectations: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 2, 1, mock.Anything).
					Return([]domain.Todo{domainTodo}, true, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ListTodosResp{
				Items:        []gen.Todo{toTodo(domainTodo)},
				Page:         2,
				NextPage:     common.Ptr(3),
				PreviousPage: common.Ptr(1),
			},
		},
		"success-with-status-filter": {
			page:     1,
			pageSize: 10,
			todoStatus: func() *gen.TodoStatus {
				s := gen.DONE
				return &s
			}(),
			setExpectations: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 10, mock.Anything).
					Run(func(_ context.Context, _ int, _ int, opts ...usecases.ListTodoOptions) {
						p := usecases.ListTodoParams{}
						for _, opt := range opts {
							opt(&p)
						}
						assert.Equal(t, domain.TodoStatus_DONE, *p.Status)
					}).
					Return([]domain.Todo{domainTodo}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ListTodosResp{
				Items: []gen.Todo{restTodo},
				Page:  1,
			},
		},
		"success-with-query": {
			page:     1,
			pageSize: 10,
			query:    common.Ptr("groceries"),
			setExpectations: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 10, mock.Anything).
					Run(func(_ context.Context, _ int, _ int, opts ...usecases.ListTodoOptions) {
						p := usecases.ListTodoParams{}
						for _, opt := range opts {
							opt(&p)
						}
						assert.Equal(t, "groceries", *p.Query)
					}).
					Return([]domain.Todo{domainTodo}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ListTodosResp{
				Items: []gen.Todo{restTodo},
				Page:  1,
			},
		},
		"sucess-with-date-range": {
			page:     1,
			pageSize: 10,
			dateRange: &gen.DateRange{
				DueAfter:  &openapi_types.Date{Time: time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)},
				DueBefore: &openapi_types.Date{Time: time.Date(2026, 1, 30, 0, 0, 0, 0, time.UTC)},
			},
			setExpectations: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 10, mock.Anything).
					Run(func(_ context.Context, _ int, _ int, opts ...usecases.ListTodoOptions) {
						p := usecases.ListTodoParams{}
						for _, opt := range opts {
							opt(&p)
						}
						assert.Equal(t, time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC), *p.DueAfter)
						assert.Equal(t, time.Date(2026, 1, 30, 0, 0, 0, 0, time.UTC), *p.DueBefore)
					}).
					Return([]domain.Todo{domainTodo}, false, nil)
			},
			expectedStatus: http.StatusOK,
		},
		"sucess-with-sort-by": {
			page:     1,
			pageSize: 10,
			sortBy:   common.Ptr("dueDateDesc"),
			setExpectations: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 10, mock.Anything).
					Run(func(_ context.Context, _ int, _ int, opts ...usecases.ListTodoOptions) {
						p := usecases.ListTodoParams{}
						for _, opt := range opts {
							opt(&p)
						}
						assert.Equal(t, "dueDateDesc", *p.SortBy)
					}).
					Return([]domain.Todo{domainTodo}, false, nil)
			},
			expectedStatus: http.StatusOK,
		},
		"use-case-error": {
			page:     1,
			pageSize: 10,
			setExpectations: func(m *usecases.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 10, mock.Anything).
					Return(nil, false, errors.New("database error"))
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
			mockListTodos := usecases.NewMockListTodos(t)
			tt.setExpectations(mockListTodos)

			server := &TodoAppServer{
				ListTodosUseCase: mockListTodos,
				Logger:           log.New(io.Discard, "", 0),
			}

			u, err := url.Parse("http://localhost/api/v1/todos")
			assert.NoError(t, err)
			q := u.Query()
			q.Set("page", strconv.Itoa(tt.page))
			q.Set("pageSize", strconv.Itoa(tt.pageSize))
			if tt.todoStatus != nil {
				q.Set("status", string(*tt.todoStatus))
			}
			if tt.query != nil {
				q.Set("query", *tt.query)
			}
			if tt.dateRange != nil {
				q.Set("dateRange[dueAfter]", tt.dateRange.DueAfter.String())
				q.Set("dateRange[dueBefore]", tt.dateRange.DueBefore.String())
			}
			if tt.sortBy != nil {
				q.Set("sort", *tt.sortBy)
			}
			u.RawQuery = q.Encode()
			req := httptest.NewRequest(http.MethodGet, u.String(), nil)

			w := httptest.NewRecorder()

			gen.Handler(server).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response gen.ListTodosResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedBody, response)
			}

			if tt.expectedError != nil {
				var response gen.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedError, response)
			}

			mockListTodos.AssertExpectations(t)
		})
	}
}

func TestTodoAppServer_UpdateTodo(t *testing.T) {
	tests := map[string]struct {
		todoID         string
		requestBody    []byte
		setupUsecases  func(*usecases.MockUpdateTodo)
		expectedStatus int
		expectedBody   *gen.Todo
		expectedError  *gen.ErrorResp
	}{
		"success": {
			todoID: domainTodo.ID.String(),
			requestBody: serializeJSON(t, gen.UpdateTodoJSONRequestBody{
				Title:   common.Ptr("Buy groceries"),
				Status:  common.Ptr(gen.DONE),
				DueDate: &openapi_types.Date{Time: dueDate},
			}),
			setupUsecases: func(m *usecases.MockUpdateTodo) {
				m.EXPECT().
					Execute(mock.Anything, domainTodo.ID, common.Ptr("Buy groceries"), common.Ptr(domain.TodoStatus_DONE), &dueDate).
					Return(domainTodo, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   &restTodo,
		},
		"todo-not-found": {
			todoID: domainTodo.ID.String(),
			requestBody: serializeJSON(t, gen.UpdateTodoJSONRequestBody{
				Status: common.Ptr(gen.DONE),
			}),
			setupUsecases: func(m *usecases.MockUpdateTodo) {
				m.EXPECT().
					Execute(mock.Anything, domainTodo.ID, (*string)(nil), common.Ptr(domain.TodoStatus_DONE), (*time.Time)(nil)).
					Return(domain.Todo{}, domain.NewNotFoundErr("todo not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError: &gen.ErrorResp{
				Error: gen.Error{
					Code:    gen.NOTFOUND,
					Message: "todo not found",
				},
			},
		},
		"invalid-status": {
			todoID:         domainTodo.ID.String(),
			requestBody:    []byte(`{"status": "INVALID_STATUS"}`),
			setupUsecases:  func(m *usecases.MockUpdateTodo) {},
			expectedStatus: http.StatusBadRequest,
			expectedError: &gen.ErrorResp{
				Error: gen.Error{
					Code:    gen.BADREQUEST,
					Message: "invalid request body: unknown TodoStatus value: INVALID_STATUS",
				},
			},
		},
		"invalid-json-body": {
			todoID:         domainTodo.ID.String(),
			requestBody:    []byte(`{"title": "Test todo", "due_date": "invalid-date"}`),
			setupUsecases:  func(m *usecases.MockUpdateTodo) {},
			expectedStatus: http.StatusBadRequest,
			expectedError: &gen.ErrorResp{
				Error: gen.Error{
					Code:    gen.BADREQUEST,
					Message: "invalid request body: error reading 'due_date': parsing time \"invalid-date\" as \"2006-01-02\": cannot parse \"invalid-date\" as \"2006\"",
				},
			},
		},
		"use-case-error": {
			todoID: domainTodo.ID.String(),
			requestBody: serializeJSON(t, gen.UpdateTodoJSONRequestBody{
				Status: common.Ptr(gen.DONE),
			}),
			setupUsecases: func(m *usecases.MockUpdateTodo) {
				m.EXPECT().
					Execute(mock.Anything, domainTodo.ID, (*string)(nil), common.Ptr(domain.TodoStatus_DONE), (*time.Time)(nil)).
					Return(domain.Todo{}, errors.New("database error"))
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
			mockUpdateTodo := usecases.NewMockUpdateTodo(t)
			tt.setupUsecases(mockUpdateTodo)
			server := &TodoAppServer{
				UpdateTodoUseCase: mockUpdateTodo,
				Logger:            log.New(io.Discard, "", 0),
			}

			req := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/"+tt.todoID, bytes.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			gen.Handler(server).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				var response gen.Todo
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedBody, response)
			}
			if tt.expectedError != nil {
				var response gen.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedError, response)
			}
		})
	}

}

func TestTodoAppServer_DeleteTodo(t *testing.T) {
	tests := map[string]struct {
		todoID         string
		setupMocks     func(*usecases.MockDeleteTodo)
		expectedStatus int
		expectedError  *gen.ErrorResp
	}{
		"success": {
			todoID: domainTodo.ID.String(),
			setupMocks: func(m *usecases.MockDeleteTodo) {
				m.EXPECT().
					Execute(mock.Anything, openapi_types.UUID(domainTodo.ID)).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		"todo-not-found": {
			todoID: domainTodo.ID.String(),
			setupMocks: func(m *usecases.MockDeleteTodo) {
				m.EXPECT().
					Execute(mock.Anything, openapi_types.UUID(domainTodo.ID)).
					Return(domain.NewNotFoundErr("todo not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError: &gen.ErrorResp{
				Error: gen.Error{
					Code:    gen.NOTFOUND,
					Message: "todo not found",
				},
			},
		},
		"use-case-error": {
			todoID: domainTodo.ID.String(),
			setupMocks: func(m *usecases.MockDeleteTodo) {
				m.EXPECT().
					Execute(mock.Anything, openapi_types.UUID(domainTodo.ID)).
					Return(errors.New("database error"))
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
			mockDeleteTodo := usecases.NewMockDeleteTodo(t)
			tt.setupMocks(mockDeleteTodo)
			server := &TodoAppServer{
				DeleteTodoUseCase: mockDeleteTodo,
				Logger:            log.New(io.Discard, "", 0),
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/"+tt.todoID, nil)
			w := httptest.NewRecorder()

			gen.Handler(server).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != nil {
				var response gen.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedError, response)
			}
		})
	}
}

// serializeJSON is a helper function to marshal a value to JSON for test requests.
func serializeJSON(t *testing.T, v any) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return data
}
