package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const actionApprovalEventsTopicID = "ActionApprovals"

// ActionApprovalDispatcher consumes approval decision messages and dispatches them
// into the in-memory approval dispatcher used by stream chat.
type ActionApprovalDispatcher struct {
	Logger              *log.Logger                        `resolve:""`
	Client              *pubsub.Client                     `resolve:""`
	Dispatcher          assistant.ActionApprovalDispatcher `resolve:""`
	SubscriptionID      string                             `config:"ACTION_APPROVAL_EVENTS_SUBSCRIPTION_ID"`
	ProjectID           string                             `config:"PUBSUB_PROJECT_ID"`
	ServerID            string
	workerExecutionChan chan struct{}
}

// Run starts the approval dispatcher worker.
func (w ActionApprovalDispatcher) Run(ctx context.Context) error {
	effectiveSubscriptionID := w.resolveSubscriptionID()
	if err := w.ensureSubscription(ctx, effectiveSubscriptionID); err != nil {
		return err
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := w.deleteSubscription(cleanupCtx, effectiveSubscriptionID); err != nil && w.Logger != nil {
			w.Logger.Printf(
				"ActionApprovalDispatcher: failed to delete subscription_id=%s: %v",
				effectiveSubscriptionID,
				err,
			)
		}
	}()

	w.Logger.Printf("ActionApprovalDispatcher: running (subscription_id=%s)...", effectiveSubscriptionID)

	subscriberErrCh := make(chan error, 1)

	go func() {
		err := w.Client.Subscriber(effectiveSubscriptionID).Receive(ctx, func(msgCtx context.Context, msg *pubsub.Message) {
			notifyProcessed := func() {
				if w.workerExecutionChan != nil {
					w.workerExecutionChan <- struct{}{}
				}
			}

			decision, err := decodeApprovalDecision(msg.Data)
			if err != nil {
				w.Logger.Printf("ActionApprovalDispatcher: invalid payload: %v", err)
				msg.Ack()
				notifyProcessed()
				return
			}

			dispatched := w.Dispatcher.Dispatch(msgCtx, decision)
			if !dispatched {
				w.Logger.Printf(
					"ActionApprovalDispatcher: no active waiter for conversation_id=%s turn_id=%s action_call_id=%s",
					decision.Key.ConversationID,
					decision.Key.TurnID,
					decision.Key.ActionCallID,
				)
			}
			msg.Ack()
			notifyProcessed()
		})
		if err != nil {
			subscriberErrCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		w.Logger.Println("ActionApprovalDispatcher: stopped")
		return nil
	case err := <-subscriberErrCh:
		return err
	}
}

// resolveSubscriptionID determines the effective subscription ID to use, applying server ID suffix if configured.
func (w ActionApprovalDispatcher) resolveSubscriptionID() string {
	base := strings.TrimSpace(w.SubscriptionID)
	if base == "" {
		return ""
	}

	serverID := strings.TrimSpace(w.ServerID)
	if serverID == "" {
		serverID = uuid.NewString()
	}
	serverID = sanitizeSubscriptionPart(serverID)
	if serverID == "" {
		return base
	}
	return base + "-" + serverID
}

// sanitizeSubscriptionPart cleans a string to be safely used as part of a Pub/Sub subscription ID,
// ensuring it meets character and length requirements.
func sanitizeSubscriptionPart(part string) string {
	trimmed := strings.TrimSpace(strings.ToLower(part))
	if trimmed == "" {
		return ""
	}

	var b strings.Builder
	prevDash := false
	for _, r := range trimmed {
		valid := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_'
		if valid {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}

	result := strings.Trim(b.String(), "-")
	const maxLen = 40
	if len(result) > maxLen {
		result = strings.Trim(result[:maxLen], "-")
	}
	return result
}

func (w ActionApprovalDispatcher) ensureSubscription(ctx context.Context, subscriptionID string) error {
	projectID := strings.TrimSpace(w.ProjectID)
	if projectID == "" {
		return errors.New("PUBSUB_PROJECT_ID is required")
	}
	if strings.TrimSpace(subscriptionID) == "" {
		return errors.New("ACTION_APPROVAL_EVENTS_SUBSCRIPTION_ID is required")
	}

	subscriptionPath := fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subscriptionID)
	_, err := w.Client.SubscriptionAdminClient.GetSubscription(
		ctx,
		&pubsubpb.GetSubscriptionRequest{Subscription: subscriptionPath},
	)
	if err == nil {
		return nil
	}

	if status.Code(err) != codes.NotFound {
		return err
	}

	topicPath := fmt.Sprintf("projects/%s/topics/%s", projectID, actionApprovalEventsTopicID)
	_, err = w.Client.SubscriptionAdminClient.CreateSubscription(
		ctx,
		&pubsubpb.Subscription{
			Name:  subscriptionPath,
			Topic: topicPath,
		},
	)
	if err != nil && status.Code(err) != codes.AlreadyExists {
		return err
	}

	return nil
}

func (w ActionApprovalDispatcher) deleteSubscription(ctx context.Context, subscriptionID string) error {
	projectID := strings.TrimSpace(w.ProjectID)
	if projectID == "" {
		return errors.New("PUBSUB_PROJECT_ID is required")
	}
	if strings.TrimSpace(subscriptionID) == "" {
		return errors.New("ACTION_APPROVAL_EVENTS_SUBSCRIPTION_ID is required")
	}

	subscriptionPath := fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subscriptionID)
	err := w.Client.SubscriptionAdminClient.DeleteSubscription(
		ctx,
		&pubsubpb.DeleteSubscriptionRequest{Subscription: subscriptionPath},
	)
	if err != nil && status.Code(err) != codes.NotFound {
		return err
	}

	return nil
}

// decodeApprovalDecision attempts to parse the incoming Pub/Sub message payload into an ActionApprovalDecision struct,
func decodeApprovalDecision(payload []byte) (assistant.ActionApprovalDecision, error) {
	var direct assistant.ActionApprovalDecision
	if err := json.Unmarshal(payload, &direct); err != nil {
		return assistant.ActionApprovalDecision{}, err
	}
	if err := direct.Validate(); err != nil {
		return assistant.ActionApprovalDecision{}, err
	}

	return direct, nil
}
