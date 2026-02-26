package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type submitActionApprovalStub struct {
	execute func(ctx context.Context, input usecases.SubmitActionApprovalInput) error
}

func (s submitActionApprovalStub) Execute(ctx context.Context, input usecases.SubmitActionApprovalInput) error {
	return s.execute(ctx, input)
}

func TestTodoAppServer_SubmitActionApproval(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	turnID := uuid.MustParse("10000000-0000-0000-0000-000000000001")

	tests := map[string]struct {
		body           []byte
		setupUsecase   func(t *testing.T) usecases.SubmitActionApproval
		expectedStatus int
		expectedError  *gen.ErrorResp
	}{
		"success": {
			body: serializeJSON(t, gen.SubmitActionApprovalJSONRequestBody{
				ConversationId: conversationID,
				TurnId:         turnID,
				ActionCallId:   "call-1",
				ActionName:     common.Ptr("delete_todo"),
				Status:         gen.APPROVED,
				Reason:         common.Ptr("approved"),
			}),
			setupUsecase: func(t *testing.T) usecases.SubmitActionApproval {
				t.Helper()
				return submitActionApprovalStub{
					execute: func(ctx context.Context, input usecases.SubmitActionApprovalInput) error {
						assert.Equal(t, conversationID, input.ConversationID)
						assert.Equal(t, turnID, input.TurnID)
						assert.Equal(t, "call-1", input.ActionCallID)
						assert.Equal(t, "delete_todo", input.ActionName)
						assert.Equal(t, domain.ChatMessageApprovalStatus_Approved, input.Status)
						assert.Equal(t, common.Ptr("approved"), input.Reason)
						return nil
					},
				}
			},
			expectedStatus: http.StatusAccepted,
		},
		"invalid-json": {
			body: []byte(`{"invalid"`),
			setupUsecase: func(t *testing.T) usecases.SubmitActionApproval {
				t.Helper()
				return submitActionApprovalStub{
					execute: func(ctx context.Context, input usecases.SubmitActionApprovalInput) error {
						t.Fatal("usecase should not be called")
						return nil
					},
				}
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
				Status:         gen.REJECTED,
			}),
			setupUsecase: func(t *testing.T) usecases.SubmitActionApproval {
				t.Helper()
				return submitActionApprovalStub{
					execute: func(ctx context.Context, input usecases.SubmitActionApprovalInput) error {
						return domain.NewValidationErr("status must be APPROVED or REJECTED")
					},
				}
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
				Status:         gen.REJECTED,
			}),
			setupUsecase: func(t *testing.T) usecases.SubmitActionApproval {
				t.Helper()
				return submitActionApprovalStub{
					execute: func(ctx context.Context, input usecases.SubmitActionApprovalInput) error {
						return errors.New("pubsub down")
					},
				}
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
			server := &TodoAppServer{
				SubmitActionApprovalUseCase: tt.setupUsecase(t),
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
