package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/google/uuid"
)

// requestActionApprovalIfRequired emits approval events and waits for a decision when the action requires approval.
func (sc StreamChatImpl) requestActionApprovalIfRequired(
	ctx context.Context,
	actionCall assistant.ActionCall,
	state *streamChatExecutionState,
	onEvent assistant.EventCallback,
) (assistant.ActionApprovalDecision, bool, error) {
	if sc.approvalDispatcher == nil {
		return assistant.ActionApprovalDecision{}, false, nil
	}

	definition, found := sc.actionRegistry.GetDefinition(actionCall.Name)
	if !found || !definition.RequiresApproval() {
		return assistant.ActionApprovalDecision{}, false, nil
	}

	approvalEvent := assistant.ActionApprovalRequired{
		ConversationID: state.conversation.ID,
		TurnID:         state.turnID,
		ActionCallID:   actionCall.ID,
		Name:           actionCall.Name,
		Input:          actionCall.Input,
		Title:          approvalTitle(definition),
		Description:    approvalDescription(definition),
		PreviewFields:  definition.Approval.PreviewFields,
		Timeout:        definition.Approval.Timeout,
	}
	if err := onEvent(ctx, assistant.EventType_ActionApprovalRequired, approvalEvent); err != nil {
		return assistant.ActionApprovalDecision{}, false, err
	}

	decision := sc.awaitActionApproval(
		ctx,
		state.conversation.ID,
		state.turnID,
		actionCall,
		definition.Approval.Timeout,
	)

	resolved := assistant.ActionApprovalResolved{
		ConversationID: state.conversation.ID,
		TurnID:         state.turnID,
		ActionCallID:   actionCall.ID,
		Name:           actionCall.Name,
		Status:         decision.Status,
		Reason:         decision.Reason,
	}
	if err := onEvent(ctx, assistant.EventType_ActionApprovalResolved, resolved); err != nil {
		return assistant.ActionApprovalDecision{}, false, err
	}

	return decision, decision.Status != assistant.ChatMessageApprovalStatus_Approved, nil
}

// awaitActionApproval waits for an approval decision for one action call and synthesizes fallback decisions on timeout or cancellation.
func (sc StreamChatImpl) awaitActionApproval(
	ctx context.Context,
	conversationID uuid.UUID,
	turnID uuid.UUID,
	actionCall assistant.ActionCall,
	timeout time.Duration,
) assistant.ActionApprovalDecision {
	key := assistant.ActionApprovalKey{
		ConversationID: conversationID,
		TurnID:         turnID,
		ActionCallID:   actionCall.ID,
	}

	now := sc.timeProvider.Now()
	if sc.approvalDispatcher == nil {
		reason := "approval dispatcher is not configured"
		return assistant.ActionApprovalDecision{
			Key:        key,
			ActionName: actionCall.Name,
			Status:     assistant.ChatMessageApprovalStatus_AutoRejected,
			Reason:     &reason,
			DecidedAt:  now,
		}
	}

	waitCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	decision, err := sc.approvalDispatcher.Wait(waitCtx, key)
	if err == nil {
		if decision.DecidedAt.IsZero() {
			decision.DecidedAt = sc.timeProvider.Now()
		}
		if strings.TrimSpace(decision.ActionName) == "" {
			decision.ActionName = actionCall.Name
		}
		return decision
	}

	status := assistant.ChatMessageApprovalStatus_AutoRejected
	reason := "approval wait canceled"
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		status = assistant.ChatMessageApprovalStatus_Expired
		reason = "approval request expired"
	case errors.Is(err, context.Canceled):
		status = assistant.ChatMessageApprovalStatus_AutoRejected
		reason = "approval request canceled"
	}

	return assistant.ActionApprovalDecision{
		Key:        key,
		ActionName: actionCall.Name,
		Status:     status,
		Reason:     &reason,
		DecidedAt:  sc.timeProvider.Now(),
	}
}

// approvalTitle returns the configured approval title or a generic fallback.
func approvalTitle(action assistant.ActionDefinition) string {
	if title := strings.TrimSpace(action.Approval.Title); title != "" {
		return title
	}
	return "Approval required"
}

// approvalDescription returns the configured approval description or a generic fallback.
func approvalDescription(action assistant.ActionDefinition) string {
	if description := strings.TrimSpace(action.Approval.Description); description != "" {
		return description
	}
	return fmt.Sprintf("Approve action '%s' execution.", action.Name)
}
