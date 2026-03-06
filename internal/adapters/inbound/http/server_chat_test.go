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

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/chat"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoAppServer_ListChatMessages(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	fixedTime := time.Date(2026, 1, 22, 10, 30, 0, 0, time.UTC)
	fixedID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	turnID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	actionExecuted := true

	domainMessage := assistant.ChatMessage{
		ID:             fixedID,
		TurnID:         turnID,
		ChatRole:       "user",
		Content:        "Hello, how are you?",
		CreatedAt:      fixedTime,
		ActionExecuted: &actionExecuted,
		SelectedSkills: []assistant.SelectedSkill{
			{
				Name:   "update_todos",
				Source: "skills/update_todos.md",
				Tools:  []string{"fetch_todos", "update_todos"},
			},
		},
		ActionDetails: []assistant.ChatMessageActionDetail{
			{
				ActionCallID:   "call-1",
				Name:           "update_todos",
				Input:          `{"todos":[{"id":"1"}]}`,
				Text:           "Updating todos...",
				Output:         "todo updated",
				MessageState:   assistant.ChatMessageState_Completed,
				ActionExecuted: &actionExecuted,
			},
		},
	}

	openAPIMessage := gen.ChatMessage{
		Id:             fixedID,
		TurnId:         common.Ptr(openapi_types.UUID(turnID)),
		Role:           gen.ChatMessageRole("user"),
		Content:        "Hello, how are you?",
		CreatedAt:      fixedTime,
		ActionExecuted: &actionExecuted,
		SelectedSkills: &[]gen.SelectedSkill{
			{
				Name:   "update_todos",
				Source: "skills/update_todos.md",
				Tools:  []string{"fetch_todos", "update_todos"},
			},
		},
		ActionDetails: &[]gen.ChatMessageActionDetail{
			{
				ActionCallId:   "call-1",
				Name:           "update_todos",
				Input:          `{"todos":[{"id":"1"}]}`,
				Text:           "Updating todos...",
				Output:         "todo updated",
				MessageState:   gen.ChatMessageActionDetailMessageState(assistant.ChatMessageState_Completed),
				ActionExecuted: &actionExecuted,
			},
		},
	}

	tests := map[string]struct {
		page           int
		pageSize       int
		setupUsecases  func(*chat.MockListChatMessages)
		expectedStatus int
		expectedBody   *gen.ChatHistoryResp
		expectedError  *gen.ErrorResp
	}{
		"success-with-messages": {
			page:     1,
			pageSize: 10,
			setupUsecases: func(m *chat.MockListChatMessages) {
				m.EXPECT().
					Query(mock.Anything, conversationID, 1, 10).
					Return([]assistant.ChatMessage{domainMessage}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ChatHistoryResp{
				ConversationId: conversationID,
				Messages:       []gen.ChatMessage{openAPIMessage},
				Page:           1,
			},
		},
		"success-with-no-messages": {
			page:     1,
			pageSize: 10,
			setupUsecases: func(m *chat.MockListChatMessages) {
				m.EXPECT().
					Query(mock.Anything, conversationID, 1, 10).
					Return([]assistant.ChatMessage{}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ChatHistoryResp{
				ConversationId: conversationID,
				Messages:       []gen.ChatMessage{},
				Page:           1,
			},
		},
		"success-with-next-and-previous-page": {
			page:     2,
			pageSize: 10,
			setupUsecases: func(m *chat.MockListChatMessages) {
				m.EXPECT().
					Query(mock.Anything, conversationID, 2, 10).
					Return([]assistant.ChatMessage{domainMessage}, true, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ChatHistoryResp{
				ConversationId: conversationID,
				Messages:       []gen.ChatMessage{openAPIMessage},
				Page:           2,
				NextPage:       common.Ptr(3),
				PreviousPage:   common.Ptr(1),
			},
		},
		"use-case-error": {
			page:     1,
			pageSize: 10,
			setupUsecases: func(m *chat.MockListChatMessages) {
				m.EXPECT().
					Query(mock.Anything, conversationID, 1, 10).
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
			mockListChatMessages := chat.NewMockListChatMessages(t)
			tt.setupUsecases(mockListChatMessages)

			server := &TodoAppServer{
				ListChatMessagesUseCase: mockListChatMessages,
			}

			u, err := url.Parse("http://localhost/api/v1/chat/messages")
			assert.NoError(t, err)
			q := u.Query()
			q.Set("conversation_id", conversationID.String())
			q.Set("page", strconv.Itoa(tt.page))
			q.Set("pageSize", strconv.Itoa(tt.pageSize))
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
	t.Parallel()

	tests := map[string]struct {
		requestBody    any
		setupUsecases  func(*chat.MockStreamChat)
		options        []chat.StreamChatOption
		expectedStatus int
		expectedEvents []string
		expectedError  *gen.ErrorResp
	}{
		"success": {
			requestBody: gen.StreamChatJSONRequestBody{Message: "Hello", Model: "qwen2.5:7B-Q4_0"},
			setupUsecases: func(m *chat.MockStreamChat) {
				m.EXPECT().
					Execute(mock.Anything, "Hello", "qwen2.5:7B-Q4_0", mock.Anything, mock.Anything).
					Run(func(ctx context.Context, userMessage string, model string, cb assistant.EventCallback, opts ...chat.StreamChatOption) {
						_ = cb(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{})
						_ = cb(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "Hi!"})
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedEvents: []string{"event: turn_started", "event: message_delta"},
		},
		"success-with-conversation-id": {
			requestBody: gen.StreamChatJSONRequestBody{
				Message:        "Hello",
				Model:          "qwen2.5:7B-Q4_0",
				ConversationId: common.Ptr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},
			setupUsecases: func(m *chat.MockStreamChat) {
				m.EXPECT().
					Execute(mock.Anything, "Hello", "qwen2.5:7B-Q4_0", mock.Anything, mock.Anything).
					Run(func(ctx context.Context, userMessage string, model string, cb assistant.EventCallback, opts ...chat.StreamChatOption) {
						// Verify that the conversation ID option is passed correctly
						params := &chat.StreamChatParams{}
						for _, opt := range opts {
							opt(params)
						}
						assert.NotNil(t, params.ConversationID)
						assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000001"), *params.ConversationID)

						_ = cb(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{})
						_ = cb(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "Hi!"})
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedEvents: []string{"event: turn_started", "event: message_delta"},
		},
		"invalid-json": {
			requestBody:    []byte(`{invalid json}`),
			setupUsecases:  func(m *chat.MockStreamChat) {},
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
			setupUsecases: func(m *chat.MockStreamChat) {
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
			mockStreamChat := chat.NewMockStreamChat(t)
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
	t.Parallel()

	tests := map[string]struct {
		setupUsecase   func(*chat.MockListAvailableModels)
		expectedStatus int
		expectedBody   *gen.ModelListResp
		expectedError  *gen.ErrorResp
	}{
		"filters-only-chat-models": {
			setupUsecase: func(m *chat.MockListAvailableModels) {
				m.EXPECT().
					Query(mock.Anything).
					Return([]assistant.ModelInfo{
						{ID: "gpt-4", Name: "gpt-4", Kind: assistant.ModelKindAssistant},
						{ID: "text-embed", Name: "text-embed", Kind: assistant.ModelKindEmbedding},
						{ID: "gpt-3.5", Name: "gpt-3.5", Kind: assistant.ModelKindAssistant},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.ModelListResp{
				Models: []gen.ModelInfo{
					{Id: "gpt-4", Name: "gpt-4"},
					{Id: "gpt-3.5", Name: "gpt-3.5"},
				},
			},
		},
		"returns-error-on-usecase-failure": {
			setupUsecase: func(m *chat.MockListAvailableModels) {
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
			mockListAvailable := chat.NewMockListAvailableModels(t)
			if tt.setupUsecase != nil {
				tt.setupUsecase(mockListAvailable)
			}

			api := TodoAppServer{
				ListAvailableModelsUseCase: mockListAvailable,
				Logger:                     log.New(io.Discard, "", 0),
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

func TestTodoAppServer_ListAvailableSkills(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupUsecase   func(*chat.MockListAvailableSkills)
		expectedStatus int
		expectedBody   *gen.SkillListResp
		expectedError  *gen.ErrorResp
	}{
		"success": {
			setupUsecase: func(m *chat.MockListAvailableSkills) {
				m.EXPECT().
					Query(mock.Anything).
					Return([]assistant.SkillDefinition{
						{
							Name:        "web_research",
							DisplayName: "Web Research",
							Aliases:     []string{"research", "web"},
							Description: "Research online sources",
							Tools:       []string{"search_web"},
						},
						{
							Name:        "update_todos",
							DisplayName: "Update Todos",
							Aliases:     []string{"update", "edit"},
							Description: "Update existing todos",
							Tools:       []string{"fetch_todos", "update_todos"},
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.SkillListResp{
				Skills: []gen.AvailableSkill{
					{
						Name:        "web_research",
						DisplayName: "Web Research",
						Aliases:     []string{"research", "web"},
						Description: "Research online sources",
						Tools:       []string{"search_web"},
					},
					{
						Name:        "update_todos",
						DisplayName: "Update Todos",
						Aliases:     []string{"update", "edit"},
						Description: "Update existing todos",
						Tools:       []string{"fetch_todos", "update_todos"},
					},
				},
			},
		},
		"returns-error-on-usecase-failure": {
			setupUsecase: func(m *chat.MockListAvailableSkills) {
				m.EXPECT().
					Query(mock.Anything).
					Return(nil, errors.New("catalog unavailable"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError: &gen.ErrorResp{
				Error: gen.Error{
					Code:    gen.INTERNALERROR,
					Message: "internal server error",
				},
			},
		},
		"falls-back-to-use-when-when-description-missing": {
			setupUsecase: func(m *chat.MockListAvailableSkills) {
				m.EXPECT().
					Query(mock.Anything).
					Return([]assistant.SkillDefinition{
						{Name: "todo_read_view", UseWhen: "List and filter existing todos", Tools: []string{"fetch_todos"}},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &gen.SkillListResp{
				Skills: []gen.AvailableSkill{
					{
						Name:        "todo_read_view",
						DisplayName: "todo_read_view",
						Aliases:     []string{},
						Description: "List and filter existing todos",
						Tools:       []string{"fetch_todos"},
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockListAvailable := chat.NewMockListAvailableSkills(t)
			if tt.setupUsecase != nil {
				tt.setupUsecase(mockListAvailable)
			}

			api := TodoAppServer{
				ListAvailableSkillsUseCase: mockListAvailable,
				Logger:                     log.New(io.Discard, "", 0),
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/skills", nil)
			rr := httptest.NewRecorder()

			api.ListAvailableSkills(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedBody != nil {
				var response gen.SkillListResp
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedBody, response)
			}

			if tt.expectedError != nil {
				var response gen.ErrorResp
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedError, response)
			}
		})
	}
}
