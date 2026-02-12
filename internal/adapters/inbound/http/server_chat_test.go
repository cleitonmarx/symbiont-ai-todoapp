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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoAppServer_ClearChatMessages(t *testing.T) {
	tests := map[string]struct {
		setupUsecases  func(*usecases.MockDeleteConversation)
		expectedStatus int
		expectedError  *gen.ErrorResp
	}{
		"success": {
			setupUsecases: func(m *usecases.MockDeleteConversation) {
				m.EXPECT().
					Execute(mock.Anything).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		"use-case-error": {
			setupUsecases: func(m *usecases.MockDeleteConversation) {
				m.EXPECT().
					Execute(mock.Anything).
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
			mockDeleteConversation := usecases.NewMockDeleteConversation(t)
			tt.setupUsecases(mockDeleteConversation)

			server := &TodoAppServer{
				DeleteConversationUseCase: mockDeleteConversation,
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/chat/messages", nil)
			w := httptest.NewRecorder()

			server.ClearChatMessages(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != nil {
				var response gen.ErrorResp
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

	openAPIMessage := gen.ChatMessage{
		Id:        fixedID,
		Role:      gen.ChatMessageRole("user"),
		Content:   "Hello, how are you?",
		CreatedAt: fixedTime,
	}

	tests := map[string]struct {
		page           int
		pageSize       int
		setupUsecases  func(*usecases.MockListChatMessages)
		expectedStatus int
		expectedBody   *gen.ChatHistoryResp
		expectedError  *gen.ErrorResp
	}{
		"success-with-messages": {
			page:     1,
			pageSize: 10,
			setupUsecases: func(m *usecases.MockListChatMessages) {
				m.EXPECT().
					Query(mock.Anything, 1, 10).
					Return([]domain.ChatMessage{domainMessage}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ChatHistoryResp{
				ConversationId: domain.GlobalConversationID,
				Messages:       []gen.ChatMessage{openAPIMessage},
				Page:           1,
			},
		},
		"success-with-no-messages": {
			page:     1,
			pageSize: 10,
			setupUsecases: func(m *usecases.MockListChatMessages) {
				m.EXPECT().
					Query(mock.Anything, 1, 10).
					Return([]domain.ChatMessage{}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ChatHistoryResp{
				ConversationId: domain.GlobalConversationID,
				Messages:       []gen.ChatMessage{},
				Page:           1,
			},
		},
		"success-with-next-and-previous-page": {
			page:     2,
			pageSize: 10,
			setupUsecases: func(m *usecases.MockListChatMessages) {
				m.EXPECT().
					Query(mock.Anything, 2, 10).
					Return([]domain.ChatMessage{domainMessage}, true, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ChatHistoryResp{
				ConversationId: domain.GlobalConversationID,
				Messages:       []gen.ChatMessage{openAPIMessage},
				Page:           2,
				NextPage:       common.Ptr(3),
				PreviousPage:   common.Ptr(1),
			},
		},
		"use-case-error": {
			page:     1,
			pageSize: 10,
			setupUsecases: func(m *usecases.MockListChatMessages) {
				m.EXPECT().
					Query(mock.Anything, 1, 10).
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
			mockListChatMessages := usecases.NewMockListChatMessages(t)
			tt.setupUsecases(mockListChatMessages)

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

			gen.Handler(server).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response gen.ChatHistoryResp
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

			mockListChatMessages.AssertExpectations(t)
		})
	}
}

func TestTodoAppServer_StreamChat(t *testing.T) {
	tests := map[string]struct {
		requestBody    any
		setupUsecases  func(*usecases.MockStreamChat)
		expectedStatus int
		expectedEvents []string
		expectedError  *gen.ErrorResp
	}{
		"success": {
			requestBody: gen.StreamChatJSONRequestBody{Message: "Hello", Model: "qwen2.5:7B-Q4_0"},
			setupUsecases: func(m *usecases.MockStreamChat) {
				m.EXPECT().
					Execute(mock.Anything, "Hello", "qwen2.5:7B-Q4_0", mock.Anything).
					Run(func(ctx context.Context, msg string, model string, cb domain.LLMStreamEventCallback) {
						_ = cb(domain.LLMStreamEventType_Meta, map[string]string{"info": "test"})
						_ = cb(domain.LLMStreamEventType_Delta, map[string]string{"text": "Hi!"})
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedEvents: []string{"event: meta", "event: delta"},
		},
		"invalid-json": {
			requestBody:    []byte(`{invalid json}`),
			setupUsecases:  func(m *usecases.MockStreamChat) {},
			expectedStatus: http.StatusBadRequest,
			expectedError: &gen.ErrorResp{
				Error: gen.Error{
					Code:    gen.BADREQUEST,
					Message: "invalid request body",
				},
			},
		},
		"use-case-error": {
			requestBody: gen.StreamChatJSONRequestBody{Message: "fail", Model: "qwen2.5:7B-Q4_0"},
			setupUsecases: func(m *usecases.MockStreamChat) {
				m.EXPECT().
					Execute(mock.Anything, "fail", "qwen2.5:7B-Q4_0", mock.Anything).
					Return(errors.New("stream error"))
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
			mockStreamChat := usecases.NewMockStreamChat(t)
			if tt.setupUsecases != nil {
				tt.setupUsecases(mockStreamChat)
			}

			server := &TodoAppServer{
				StreamChatUseCase: mockStreamChat,
				Logger:            log.New(io.Discard, "", 0), // Prevents nil pointer panic
			}

			var req *http.Request
			switch v := tt.requestBody.(type) {
			case []byte:
				req = httptest.NewRequest(http.MethodPost, "/api/v1/chat/stream", bytes.NewReader(v))
			default:
				body, _ := json.Marshal(v)
				req = httptest.NewRequest(http.MethodPost, "/api/v1/chat/stream", bytes.NewReader(body))
			}
			req.Header.Set("Content-Type", "application/json")

			// For streaming, ResponseRecorder does not implement http.Flusher, so we use a custom ResponseWriter
			w := newMockFlusherRecorder()

			server.StreamChat(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedEvents != nil {
				body := w.Body.String()
				for _, event := range tt.expectedEvents {
					assert.Contains(t, body, event)
				}
			}

			if tt.expectedError != nil {
				var response gen.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedError, response)
			}

			mockStreamChat.AssertExpectations(t)
		})
	}
}

// mockFlusherRecorder is a ResponseRecorder that implements http.Flusher
type mockFlusherRecorder struct {
	*httptest.ResponseRecorder
}

func newMockFlusherRecorder() *mockFlusherRecorder {
	return &mockFlusherRecorder{httptest.NewRecorder()}
}

func (m *mockFlusherRecorder) Flush() {}

func TestTodoAppServer_ListAvailableModels(t *testing.T) {
	tests := map[string]struct {
		setupUsecase   func(*usecases.MockListAvailableLLMModels)
		expectedStatus int
		expectedBody   *gen.ModelListResp
		expectedError  *gen.ErrorResp
	}{
		"filters-only-chat-models": {
			setupUsecase: func(m *usecases.MockListAvailableLLMModels) {
				m.EXPECT().
					Query(mock.Anything).
					Return([]domain.LLMModelInfo{
						{Name: "gpt-4", Type: domain.LLMModelType_Chat},
						{Name: "text-embed", Type: domain.LLMModelType_Embedding},
						{Name: "gpt-3.5", Type: domain.LLMModelType_Chat},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ModelListResp{
				Models: []string{"gpt-4", "gpt-3.5"},
			},
		},
		"returns-error-on-usecase-failure": {
			setupUsecase: func(m *usecases.MockListAvailableLLMModels) {
				m.EXPECT().
					Query(mock.Anything).
					Return(nil, errors.New("database error"))
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
			mockListAvailable := usecases.NewMockListAvailableLLMModels(t)
			if tt.setupUsecase != nil {
				tt.setupUsecase(mockListAvailable)
			}

			api := TodoAppServer{
				ListAvailableLLMModels: mockListAvailable,
				Logger:                 log.New(io.Discard, "", 0),
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
			rr := httptest.NewRecorder()

			api.ListAvailableModels(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedBody != nil {
				var response gen.ModelListResp
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedBody.Models, response.Models)
			}

			if tt.expectedError != nil {
				var response gen.ErrorResp
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedError, response)
			}

			mockListAvailable.AssertExpectations(t)
		})
	}
}
