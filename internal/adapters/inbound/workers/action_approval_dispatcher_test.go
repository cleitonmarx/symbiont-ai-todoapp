package workers

import (
	"encoding/json"
	"log"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestActionApprovalDispatcher_Run(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	turnID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	actionCallID := "func-1"

	tests := map[string]struct {
		payload                    []byte
		expectDispatch             bool
		expectedDispatchStatus     assistant.ChatMessageApprovalStatus
		expectedDispatchActionName string
		expectedDispatchReason     *string
		dispatchReturn             bool
	}{
		"accepts-domain-decision-payload": {
			payload: approvalDecisionJSON(t, assistant.ActionApprovalDecision{
				Key: assistant.ActionApprovalKey{
					ConversationID: conversationID,
					TurnID:         turnID,
					ActionCallID:   actionCallID,
				},
				ActionName: "delete_todo",
				Status:     assistant.ChatMessageApprovalStatus_Approved,
				Reason:     common.Ptr("approved"),
			}),
			expectDispatch:             true,
			expectedDispatchStatus:     assistant.ChatMessageApprovalStatus_Approved,
			expectedDispatchActionName: "delete_todo",
			expectedDispatchReason:     common.Ptr("approved"),
			dispatchReturn:             true,
		},
		"accepts-snake-case-payload-and-normalizes-status": {
			payload: approvalDecisionSnakeJSON(t, map[string]any{
				"conversation_id": conversationID.String(),
				"turn_id":         turnID.String(),
				"action_call_id":  actionCallID,
				"action_name":     "delete_todo",
				"status":          "approved",
				"reason":          "approved from endpoint",
			}),
			expectDispatch:             true,
			expectedDispatchStatus:     assistant.ChatMessageApprovalStatus_Approved,
			expectedDispatchActionName: "delete_todo",
			expectedDispatchReason:     common.Ptr("approved from endpoint"),
			dispatchReturn:             true,
		},
		"invalid-payload": {
			payload:        []byte(`{"invalid"`),
			expectDispatch: false,
		},
		"invalid-status": {
			payload: approvalDecisionSnakeJSON(t, map[string]any{
				"conversation_id": conversationID.String(),
				"turn_id":         turnID.String(),
				"action_call_id":  actionCallID,
				"action_name":     "delete_todo",
				"status":          "PENDING",
			}),
			expectDispatch: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			topicID := actionApprovalEventsTopicID
			subscriptionID := "approval-sub-" + name
			client, topicName := setupPubSubServer(t, ctx, topicID, subscriptionID)
			dispatcher := assistant.NewMockActionApprovalDispatcher(t)

			if tc.expectDispatch {
				dispatcher.EXPECT().
					Dispatch(mock.Anything, mock.MatchedBy(func(decision assistant.ActionApprovalDecision) bool {
						if decision.Key.ConversationID != conversationID {
							return false
						}
						if decision.Key.TurnID != turnID {
							return false
						}
						if decision.Key.ActionCallID != actionCallID {
							return false
						}
						if decision.ActionName != tc.expectedDispatchActionName {
							return false
						}
						if decision.Status != tc.expectedDispatchStatus {
							return false
						}
						if tc.expectedDispatchReason == nil {
							return decision.Reason == nil
						}
						return decision.Reason != nil && *decision.Reason == *tc.expectedDispatchReason
					})).
					Return(tc.dispatchReturn).
					Once()
			}

			signalChan := make(chan struct{}, 10)
			worker := ActionApprovalDispatcher{
				Logger:              log.Default(),
				Client:              client,
				Dispatcher:          dispatcher,
				SubscriptionID:      subscriptionID,
				ProjectID:           "test-project",
				ServerID:            "server_" + name,
				workerExecutionChan: signalChan,
			}
			effectiveSubscriptionID := worker.resolveSubscriptionID()

			cancel, doneChan := run(t, ctx, worker)

			err := publishMessages(ctx, client, topicName, [][]byte{tc.payload})
			assert.NoError(t, err)

			waitForBatchSignals(t, signalChan, 1, 1*time.Second)

			cancel()
			waitRunnableStop(t, doneChan)

			_, err = client.SubscriptionAdminClient.GetSubscription(
				ctx,
				&pubsubpb.GetSubscriptionRequest{
					Subscription: "projects/test-project/subscriptions/" + effectiveSubscriptionID,
				},
			)
			assert.Error(t, err)
			assert.Equal(t, codes.NotFound, status.Code(err))
		})
	}
}

func TestActionApprovalDispatcher_resolveSubscriptionID(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		worker               ActionApprovalDispatcher
		expectedSubscription string
		expectedPattern      string
	}{
		"custom-server-id": {
			worker: ActionApprovalDispatcher{
				SubscriptionID: "action_approval_dispatcher",
				ServerID:       "api@node#1",
			},
			expectedSubscription: "action_approval_dispatcher-api-node-1",
		},
		"generated-server-id": {
			worker: ActionApprovalDispatcher{
				SubscriptionID: "action_approval_dispatcher",
			},
			expectedPattern: `^action_approval_dispatcher-[a-z0-9_-]+$`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.worker.resolveSubscriptionID()
			if tt.expectedSubscription != "" {
				assert.Equal(t, tt.expectedSubscription, got)
				return
			}

			assert.Regexp(t, tt.expectedPattern, got)
			assert.NotEqual(t, tt.worker.SubscriptionID, got)
		})
	}
}

func approvalDecisionJSON(t *testing.T, decision assistant.ActionApprovalDecision) []byte {
	t.Helper()

	data, err := json.Marshal(decision)
	assert.NoError(t, err)
	return data
}

func approvalDecisionSnakeJSON(t *testing.T, payload map[string]any) []byte {
	t.Helper()

	data, err := json.Marshal(payload)
	assert.NoError(t, err)
	return data
}
