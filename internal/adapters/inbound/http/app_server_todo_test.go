package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/openapi"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases/mocks"
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
	restTodo = openapi.Todo{
		Id:        openapi_types.UUID(domainTodo.ID),
		Title:     domainTodo.Title,
		Status:    openapi.DONE,
		DueDate:   openapi_types.Date{Time: domainTodo.DueDate},
		CreatedAt: domainTodo.CreatedAt,
		UpdatedAt: domainTodo.UpdatedAt,
	}
)

func TestTodoAppServer_CreateTodo(t *testing.T) {
	tests := map[string]struct {
		requestBody    []byte
		setupMocks     func(*mocks.MockCreateTodo)
		expectedStatus int
		expectedBody   *openapi.Todo
		expectedError  *openapi.ErrorResp
	}{
		"success": {
			requestBody: serializeJSON(t, openapi.CreateTodoJSONRequestBody{
				Title:   "Buy groceries",
				DueDate: openapi_types.Date{Time: dueDate},
			}),
			setupMocks: func(m *mocks.MockCreateTodo) {
				m.EXPECT().
					Execute(mock.Anything, "Buy groceries", dueDate).Return(domainTodo, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   &restTodo,
		},
		"bad-request": {
			requestBody: serializeJSON(t, openapi.CreateTodoJSONRequestBody{
				DueDate: openapi_types.Date{Time: dueDate},
			}),
			setupMocks: func(m *mocks.MockCreateTodo) {
				m.EXPECT().
					Execute(mock.Anything, "", dueDate).
					Return(domain.Todo{}, domain.NewValidationErr("title is required"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.BADREQUEST,
					Message: "title is required",
				},
			},
		},
		"invalid-json-body": {
			requestBody:    []byte(`{"title": "Test todo", "due_date": "invalid-date"}`),
			setupMocks:     func(m *mocks.MockCreateTodo) {},
			expectedStatus: http.StatusBadRequest,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.BADREQUEST,
					Message: "invalid request body: parsing time \"invalid-date\" as \"2006-01-02\": cannot parse \"invalid-date\" as \"2006\"",
				},
			},
		},
		"internal-server-error": {
			requestBody: serializeJSON(t, openapi.CreateTodoJSONRequestBody{
				Title:   "Test todo",
				DueDate: openapi_types.Date{Time: time.Time{}},
			}),
			setupMocks: func(m *mocks.MockCreateTodo) {
				m.EXPECT().
					Execute(mock.Anything, "Test todo", time.Time{}).
					Return(domain.Todo{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.INTERNALERROR,
					Message: "internal server error",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockCreateTodo := mocks.NewMockCreateTodo(t)
			if tt.setupMocks != nil {
				tt.setupMocks(mockCreateTodo)
			}

			server := &TodoAppServer{
				CreateTodoUseCase: mockCreateTodo,
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			openapi.Handler(server).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response openapi.Todo
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody.Id, response.Id)
				assert.Equal(t, tt.expectedBody.Title, response.Title)
				assert.Equal(t, tt.expectedBody.Status, response.Status)
			}

			if tt.expectedError != nil {
				var response openapi.ErrorResp
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
		page           int
		pageSize       int
		todoStatus     *openapi.TodoStatus
		setupMocks     func(*mocks.MockListTodos)
		expectedStatus int
		expectedBody   *openapi.ListTodosResp
		expectedError  *openapi.ErrorResp
	}{
		"success-with-todos": {
			page:     1,
			pageSize: 1,
			setupMocks: func(m *mocks.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 1, mock.Anything).
					Return([]domain.Todo{domainTodo}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &openapi.ListTodosResp{
				Items: []openapi.Todo{toOpenAPITodo(domainTodo)},
				Page:  1,
			},
		},
		"success-with-no-todos": {
			page:     1,
			pageSize: 1,
			setupMocks: func(m *mocks.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 1, mock.Anything).
					Return([]domain.Todo{}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &openapi.ListTodosResp{
				Items: []openapi.Todo{},
				Page:  1,
			},
		},
		"success-with-next-and-previous-page": {
			page:     2,
			pageSize: 1,
			setupMocks: func(m *mocks.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 2, 1, mock.Anything).
					Return([]domain.Todo{domainTodo}, true, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &openapi.ListTodosResp{
				Items:        []openapi.Todo{toOpenAPITodo(domainTodo)},
				Page:         2,
				NextPage:     common.Ptr(3),
				PreviousPage: common.Ptr(1),
			},
		},
		"success-with-status-filter": {
			page:     1,
			pageSize: 10,
			todoStatus: func() *openapi.TodoStatus {
				s := openapi.DONE
				return &s
			}(),
			setupMocks: func(m *mocks.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 10, mock.Anything).
					Run(func(_ context.Context, _ int, _ int, opts ...domain.ListTodoOptions) {
						p := domain.ListTodosParams{}
						for _, opt := range opts {
							opt(&p)
						}
						assert.Equal(t, domain.TodoStatus_DONE, *p.Status)
					}).
					Return([]domain.Todo{domainTodo}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &openapi.ListTodosResp{
				Items: []openapi.Todo{restTodo},
				Page:  1,
			},
		},
		"use-case-error": {
			page:     1,
			pageSize: 10,
			setupMocks: func(m *mocks.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 10, mock.Anything).
					Return(nil, false, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.INTERNALERROR,
					Message: "internal server error",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockListTodos := mocks.NewMockListTodos(t)
			tt.setupMocks(mockListTodos)

			server := &TodoAppServer{
				ListTodosUseCase: mockListTodos,
			}

			u, err := url.Parse("http://localhost/api/v1/todos")
			assert.NoError(t, err)
			q := u.Query()
			q.Set("page", strconv.Itoa(tt.page))
			q.Set("pagesize", strconv.Itoa(tt.pageSize))
			if tt.todoStatus != nil {
				q.Set("status", string(*tt.todoStatus))
			}
			u.RawQuery = q.Encode()
			req := httptest.NewRequest(http.MethodGet, u.String(), nil)

			w := httptest.NewRecorder()

			openapi.Handler(server).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response openapi.ListTodosResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedBody, response)
			}

			if tt.expectedError != nil {
				var response openapi.ErrorResp
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
		setupMocks     func(*mocks.MockUpdateTodo)
		expectedStatus int
		expectedBody   *openapi.Todo
		expectedError  *openapi.ErrorResp
	}{
		"success": {
			todoID: domainTodo.ID.String(),
			requestBody: serializeJSON(t, openapi.UpdateTodoJSONRequestBody{
				Title:   common.Ptr("Buy groceries"),
				Status:  common.Ptr(openapi.DONE),
				DueDate: &openapi_types.Date{Time: dueDate},
			}),
			setupMocks: func(m *mocks.MockUpdateTodo) {
				m.EXPECT().
					Execute(mock.Anything, domainTodo.ID, common.Ptr("Buy groceries"), common.Ptr(domain.TodoStatus_DONE), &dueDate).
					Return(domainTodo, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   &restTodo,
		},
		"todo-not-found": {
			todoID: domainTodo.ID.String(),
			requestBody: serializeJSON(t, openapi.UpdateTodoJSONRequestBody{
				Status: common.Ptr(openapi.DONE),
			}),
			setupMocks: func(m *mocks.MockUpdateTodo) {
				m.EXPECT().
					Execute(mock.Anything, domainTodo.ID, (*string)(nil), common.Ptr(domain.TodoStatus_DONE), (*time.Time)(nil)).
					Return(domain.Todo{}, domain.NewNotFoundErr("todo not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.NOTFOUND,
					Message: "todo not found",
				},
			},
		},
		"invalid-status": {
			todoID:         domainTodo.ID.String(),
			requestBody:    []byte(`{"status": "INVALID_STATUS"}`),
			setupMocks:     func(m *mocks.MockUpdateTodo) {},
			expectedStatus: http.StatusBadRequest,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.BADREQUEST,
					Message: "invalid request body: unknown TodoStatus value: INVALID_STATUS",
				},
			},
		},
		"invalid-json-body": {
			todoID:         domainTodo.ID.String(),
			requestBody:    []byte(`{"title": "Test todo", "due_date": "invalid-date"}`),
			setupMocks:     func(m *mocks.MockUpdateTodo) {},
			expectedStatus: http.StatusBadRequest,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.BADREQUEST,
					Message: "invalid request body: error reading 'due_date': parsing time \"invalid-date\" as \"2006-01-02\": cannot parse \"invalid-date\" as \"2006\"",
				},
			},
		},
		"use-case-error": {
			todoID: domainTodo.ID.String(),
			requestBody: serializeJSON(t, openapi.UpdateTodoJSONRequestBody{
				Status: common.Ptr(openapi.DONE),
			}),
			setupMocks: func(m *mocks.MockUpdateTodo) {
				m.EXPECT().
					Execute(mock.Anything, domainTodo.ID, (*string)(nil), common.Ptr(domain.TodoStatus_DONE), (*time.Time)(nil)).
					Return(domain.Todo{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.INTERNALERROR,
					Message: "internal server error",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockUpdateTodo := mocks.NewMockUpdateTodo(t)
			tt.setupMocks(mockUpdateTodo)
			server := &TodoAppServer{
				UpdateTodoUseCase: mockUpdateTodo,
			}

			req := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/"+tt.todoID, bytes.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			openapi.Handler(server).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				var response openapi.Todo
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedBody, response)
			}
			if tt.expectedError != nil {
				var response openapi.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedError, response)
			}
		})
	}

}

func TestTodoAppServer_ClearChatMessages(t *testing.T) {
	tests := map[string]struct {
		setupMocks     func(*mocks.MockDeleteConversation)
		expectedStatus int
		expectedError  *openapi.ErrorResp
	}{
		"success": {
			setupMocks: func(m *mocks.MockDeleteConversation) {
				m.EXPECT().
					Execute(mock.Anything).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		"use-case-error": {
			setupMocks: func(m *mocks.MockDeleteConversation) {
				m.EXPECT().
					Execute(mock.Anything).
					Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.INTERNALERROR,
					Message: "internal server error",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockDeleteConversation := mocks.NewMockDeleteConversation(t)
			tt.setupMocks(mockDeleteConversation)

			server := &TodoAppServer{
				DeleteConversationUseCase: mockDeleteConversation,
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/chat/messages", nil)
			w := httptest.NewRecorder()

			server.ClearChatMessages(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != nil {
				var response openapi.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError.Error, response.Error)
			}

			mockDeleteConversation.AssertExpectations(t)
		})
	}
}

func TestTodoAppServer_ListChatMessages(t *testing.T) {
	fixedTime := time.Date(2026, 1, 22, 10, 30, 0, 0, time.UTC)
	fixedID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	domainMessage := domain.ChatMessage{
		ID:        fixedID,
		ChatRole:  "user",
		Content:   "Hello, how are you?",
		CreatedAt: fixedTime,
	}

	openAPIMessage := openapi.ChatMessage{
		Id:        fixedID,
		Role:      openapi.ChatMessageRole("user"),
		Content:   "Hello, how are you?",
		CreatedAt: fixedTime,
	}

	tests := map[string]struct {
		page           int
		pageSize       int
		setupMocks     func(*mocks.MockListChatMessages)
		expectedStatus int
		expectedBody   *openapi.ChatHistoryResp
		expectedError  *openapi.ErrorResp
	}{
		"success-with-messages": {
			page:     1,
			pageSize: 10,
			setupMocks: func(m *mocks.MockListChatMessages) {
				m.EXPECT().
					Query(mock.Anything, 1, 10).
					Return([]domain.ChatMessage{domainMessage}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &openapi.ChatHistoryResp{
				ConversationId: domain.GlobalConversationID,
				Messages:       []openapi.ChatMessage{openAPIMessage},
				Page:           1,
			},
		},
		"success-with-no-messages": {
			page:     1,
			pageSize: 10,
			setupMocks: func(m *mocks.MockListChatMessages) {
				m.EXPECT().
					Query(mock.Anything, 1, 10).
					Return([]domain.ChatMessage{}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &openapi.ChatHistoryResp{
				ConversationId: domain.GlobalConversationID,
				Messages:       []openapi.ChatMessage{},
				Page:           1,
			},
		},
		"success-with-next-and-previous-page": {
			page:     2,
			pageSize: 10,
			setupMocks: func(m *mocks.MockListChatMessages) {
				m.EXPECT().
					Query(mock.Anything, 2, 10).
					Return([]domain.ChatMessage{domainMessage}, true, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &openapi.ChatHistoryResp{
				ConversationId: domain.GlobalConversationID,
				Messages:       []openapi.ChatMessage{openAPIMessage},
				Page:           2,
				NextPage:       common.Ptr(3),
				PreviousPage:   common.Ptr(1),
			},
		},
		"use-case-error": {
			page:     1,
			pageSize: 10,
			setupMocks: func(m *mocks.MockListChatMessages) {
				m.EXPECT().
					Query(mock.Anything, 1, 10).
					Return(nil, false, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.INTERNALERROR,
					Message: "internal server error",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockListChatMessages := mocks.NewMockListChatMessages(t)
			tt.setupMocks(mockListChatMessages)

			server := &TodoAppServer{
				ListChatMessagesUseCase: mockListChatMessages,
			}

			u, err := url.Parse("http://localhost/api/v1/chat/messages")
			assert.NoError(t, err)
			q := u.Query()
			q.Set("page", strconv.Itoa(tt.page))
			q.Set("pagesize", strconv.Itoa(tt.pageSize))
			u.RawQuery = q.Encode()
			req := httptest.NewRequest(http.MethodGet, u.String(), nil)

			w := httptest.NewRecorder()

			openapi.Handler(server).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response openapi.ChatHistoryResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedBody, response)
			}

			if tt.expectedError != nil {
				var response openapi.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedError, response)
			}

			mockListChatMessages.AssertExpectations(t)
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
