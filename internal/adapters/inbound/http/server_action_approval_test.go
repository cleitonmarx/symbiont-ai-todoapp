package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/chat"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoAppServer_SubmitActionApproval(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	turnID := uuid.MustParse("10000000-0000-0000-0000-000000000001")

	tests := map[string]struct {
		body           []byte
		setupUsecase   func(*chat.MockSubmitActionApproval)
		expectedStatus int
		expectedError  *gen.ErrorResp
	}{
		"success": {
			body: serializeJSON(t, gen.SubmitActionApprovalJSONRequestBody{
				ConversationId: conversationID,
				TurnId:         turnID,
				ActionCallId:   "call-1",
				ActionName:     common.Ptr("delete_todo"),
				Status:         gen.ActionApprovalStatusAPPROVED,
				Reason:         common.Ptr("approved"),
			}),
			setupUsecase: func(m *chat.MockSubmitActionApproval) {
				m.EXPECT().
					Execute(mock.Anything, chat.SubmitActionApprovalInput{
						ConversationID: conversationID,
						TurnID:         turnID,
						ActionCallID:   "call-1",
						ActionName:     "delete_todo",
						Status:         assistant.ChatMessageApprovalStatus_Approved,
						Reason:         common.Ptr("approved"),
					}).
					Return(nil)
			},
			expectedStatus: http.StatusAccepted,
		},
		"invalid-json": {
			body: []byte(`{"invalid"`),
			setupUsecase: func(m *chat.MockSubmitActionApproval) {
			},
			expectedStatus: http.StatusBadRequest,
			expectedError: &gen.ErrorResp{
				Error: gen.Error{
					Code:    gen.BADREQUEST,
					Message: "invalid request body",
				},
			},
		},
		"usecase-validation-error": {
			body: serializeJSON(t, gen.SubmitActionApprovalJSONRequestBody{
				ConversationId: conversationID,
				TurnId:         turnID,
				ActionCallId:   "call-3",
				Status:         gen.ActionApprovalStatusREJECTED,
			}),
			setupUsecase: func(m *chat.MockSubmitActionApproval) {
				m.EXPECT().
					Execute(mock.Anything, chat.SubmitActionApprovalInput{
						ConversationID: conversationID,
						TurnID:         turnID,
						ActionCallID:   "call-3",
						Status:         assistant.ChatMessageApprovalStatus_Rejected,
					}).
					Return(core.NewValidationErr("status must be APPROVED or REJECTED"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError: &gen.ErrorResp{
				Error: gen.Error{
					Code:    gen.BADREQUEST,
					Message: "status must be APPROVED or REJECTED",
				},
			},
		},
		"usecase-internal-error": {
			body: serializeJSON(t, gen.SubmitActionApprovalJSONRequestBody{
				ConversationId: conversationID,
				TurnId:         turnID,
				ActionCallId:   "call-4",
				Status:         gen.ActionApprovalStatusREJECTED,
			}),
			setupUsecase: func(m *chat.MockSubmitActionApproval) {
				m.EXPECT().
					Execute(mock.Anything, chat.SubmitActionApprovalInput{
						ConversationID: conversationID,
						TurnID:         turnID,
						ActionCallID:   "call-4",
						Status:         assistant.ChatMessageApprovalStatus_Rejected,
					}).
					Return(errors.New("pubsub down"))
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
			mockUC := chat.NewMockSubmitActionApproval(t)
			tt.setupUsecase(mockUC)
			server := &TodoAppServer{
				SubmitActionApprovalUseCase: mockUC,
				Logger:                      log.New(io.Discard, "", 0),
			}

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/chat/approvals",
				bytes.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")
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
