package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoAppServer_DeleteConversation(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tests := map[string]struct {
		setupUsecases  func(*usecases.MockDeleteConversation)
		expectedStatus int
		expectedError  *gen.ErrorResp
	}{
		"success": {
			setupUsecases: func(m *usecases.MockDeleteConversation) {
				m.EXPECT().
					Execute(mock.Anything, conversationID).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		"use-case-error": {
			setupUsecases: func(m *usecases.MockDeleteConversation) {
				m.EXPECT().
					Execute(mock.Anything, conversationID).
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

			server.DeleteConversation(w, req, conversationID)

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

func TestTodoAppServer_UpdateConversation(t *testing.T) {
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	newTitle := "Updated Conversation Title"

	tests := map[string]struct {
		conversationID       openapi_types.UUID
		requestBody          []byte
		setExpectations      func(uc *usecases.MockUpdateConversation)
		expectedStatusCode   int
		expectedResponseBody interface{}
		expectedErr          bool
	}{
		"success-update-title": {
			conversationID: openapi_types.UUID(fixedUUID),
			requestBody:    serializeJSON(t, gen.UpdateConversationRequest{Title: newTitle}),
			setExpectations: func(uc *usecases.MockUpdateConversation) {
				uc.EXPECT().Execute(mock.Anything, fixedUUID, newTitle).Return(
					domain.Conversation{
						ID:          fixedUUID,
						Title:       newTitle,
						TitleSource: domain.ConversationTitleSource_User,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedErr:        false,
		},
		"malformed-json": {
			conversationID:     openapi_types.UUID(fixedUUID),
			requestBody:        []byte(`{invalid json}`),
			setExpectations:    func(uc *usecases.MockUpdateConversation) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErr:        false,
		},
		"empty-request-body": {
			conversationID:     openapi_types.UUID(fixedUUID),
			requestBody:        []byte(``),
			setExpectations:    func(uc *usecases.MockUpdateConversation) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErr:        false,
		},
		"conversation-not-found": {
			conversationID: openapi_types.UUID(fixedUUID),
			requestBody:    serializeJSON(t, gen.UpdateConversationRequest{Title: newTitle}),
			setExpectations: func(uc *usecases.MockUpdateConversation) {
				uc.EXPECT().Execute(mock.Anything, fixedUUID, newTitle).Return(
					domain.Conversation{},
					domain.NewNotFoundErr("conversation not found"))
			},
			expectedStatusCode: http.StatusNotFound,
			expectedErr:        false,
		},
		"validation-error": {
			conversationID: openapi_types.UUID(fixedUUID),
			requestBody:    serializeJSON(t, gen.UpdateConversationRequest{Title: ""}),
			setExpectations: func(uc *usecases.MockUpdateConversation) {
				uc.EXPECT().Execute(mock.Anything, fixedUUID, "").Return(
					domain.Conversation{},
					domain.NewValidationErr("conversation title cannot be empty"))
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedErr:        false,
		},
		"use-case-error": {
			conversationID: openapi_types.UUID(fixedUUID),
			requestBody:    serializeJSON(t, gen.UpdateConversationRequest{Title: newTitle}),
			setExpectations: func(uc *usecases.MockUpdateConversation) {
				uc.EXPECT().Execute(mock.Anything, fixedUUID, newTitle).Return(
					domain.Conversation{},
					errors.New("internal server error"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedErr:        false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockUC := usecases.NewMockUpdateConversation(t)
			if tt.setExpectations != nil {
				tt.setExpectations(mockUC)
			}

			server := TodoAppServer{
				UpdateConversationUseCase: mockUC,
			}

			req := httptest.NewRequest(http.MethodPatch, "/api/conversations/"+tt.conversationID.String(), bytes.NewBuffer(tt.requestBody))
			w := httptest.NewRecorder()

			server.UpdateConversation(w, req, tt.conversationID)

			assert.Equal(t, tt.expectedStatusCode, w.Code)
			mockUC.AssertExpectations(t)
		})
	}
}

func TestTodoAppServer_ListConversations(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		page                int
		pageSize            int
		setExpectations     func(uc *usecases.MockListConversations)
		expectedStatusCode  int
		expectedHasNextPage bool
		expectedHasPrevPage bool
		expectedErr         bool
	}{
		"success-first-page": {
			page:     1,
			pageSize: 10,
			setExpectations: func(uc *usecases.MockListConversations) {
				uc.EXPECT().Query(mock.Anything, 1, 10).Return([]domain.Conversation{
					{
						ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						Title:       "Conversation 1",
						TitleSource: domain.ConversationTitleSource_User,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
				}, true, nil)
			},
			expectedStatusCode:  http.StatusOK,
			expectedHasNextPage: true,
			expectedHasPrevPage: false,
			expectedErr:         false,
		},
		"success-middle-page": {
			page:     2,
			pageSize: 10,
			setExpectations: func(uc *usecases.MockListConversations) {
				uc.EXPECT().Query(mock.Anything, 2, 10).Return([]domain.Conversation{
					{
						ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						Title:       "Conversation 2",
						TitleSource: domain.ConversationTitleSource_LLM,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
				}, true, nil)
			},
			expectedStatusCode:  http.StatusOK,
			expectedHasNextPage: true,
			expectedHasPrevPage: true,
			expectedErr:         false,
		},
		"success-last-page": {
			page:     3,
			pageSize: 10,
			setExpectations: func(uc *usecases.MockListConversations) {
				uc.EXPECT().Query(mock.Anything, 3, 10).Return([]domain.Conversation{
					{
						ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						Title:       "Conversation 3",
						TitleSource: domain.ConversationTitleSource_Auto,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
				}, false, nil)
			},
			expectedStatusCode:  http.StatusOK,
			expectedHasNextPage: false,
			expectedHasPrevPage: true,
			expectedErr:         false,
		},
		"success-empty-list": {
			page:     1,
			pageSize: 10,
			setExpectations: func(uc *usecases.MockListConversations) {
				uc.EXPECT().Query(mock.Anything, 1, 10).Return([]domain.Conversation{}, false, nil)
			},
			expectedStatusCode:  http.StatusOK,
			expectedHasNextPage: false,
			expectedHasPrevPage: false,
			expectedErr:         false,
		},
		"use-case-error": {
			page:     1,
			pageSize: 10,
			setExpectations: func(uc *usecases.MockListConversations) {
				uc.EXPECT().Query(mock.Anything, 1, 10).Return(nil, false, errors.New("database error"))
			},
			expectedStatusCode:  http.StatusInternalServerError,
			expectedHasNextPage: false,
			expectedHasPrevPage: false,
			expectedErr:         false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockUC := usecases.NewMockListConversations(t)
			if tt.setExpectations != nil {
				tt.setExpectations(mockUC)
			}

			server := TodoAppServer{
				ListConversationsUseCase: mockUC,
			}

			req := httptest.NewRequest(http.MethodGet, "/api/conversations", nil)
			req = req.WithContext(context.Background())

			params := gen.ListConversationsParams{
				Page:     tt.page,
				PageSize: tt.pageSize,
			}

			w := httptest.NewRecorder()

			server.ListConversations(w, req, params)

			assert.Equal(t, tt.expectedStatusCode, w.Code)

			if w.Code == http.StatusOK {
				var resp gen.ConversationListResp
				json.NewDecoder(w.Body).Decode(&resp)

				if tt.expectedHasNextPage {
					assert.NotNil(t, resp.NextPage)
				} else {
					assert.Nil(t, resp.NextPage)
				}

				if tt.expectedHasPrevPage {
					assert.NotNil(t, resp.PreviousPage)
				} else {
					assert.Nil(t, resp.PreviousPage)
				}
			}
		})
	}
}
