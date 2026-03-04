package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
)

// requestActionApprovalIfRequired emits approval events and waits for a decision when the action requires approval.
func (sc StreamChatImpl) requestActionApprovalIfRequired(
	ctx context.Context,
	actionCall domain.AssistantActionCall,
	state *streamChatExecutionState,
	onEvent domain.AssistantEventCallback,
) (domain.AssistantActionApprovalDecision, bool, error) {
	if sc.approvalDispatcher == nil {
		return domain.AssistantActionApprovalDecision{}, false, nil
	}

	definition, found := sc.actionRegistry.GetDefinition(actionCall.Name)
	if !found || !definition.RequiresApproval() {
		return domain.AssistantActionApprovalDecision{}, false, nil
	}

	approvalEvent := domain.AssistantActionApprovalRequired{
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
	if err := onEvent(ctx, domain.AssistantEventType_ActionApprovalRequired, approvalEvent); err != nil {
		return domain.AssistantActionApprovalDecision{}, false, err
	}

	decision := sc.awaitActionApproval(
		ctx,
		state.conversation.ID,
		state.turnID,
		actionCall,
		definition.Approval.Timeout,
	)

	resolved := domain.AssistantActionApprovalResolved{
		ConversationID: state.conversation.ID,
		TurnID:         state.turnID,
		ActionCallID:   actionCall.ID,
		Name:           actionCall.Name,
		Status:         decision.Status,
		Reason:         decision.Reason,
	}
	if err := onEvent(ctx, domain.AssistantEventType_ActionApprovalResolved, resolved); err != nil {
		return domain.AssistantActionApprovalDecision{}, false, err
	}

	return decision, decision.Status != domain.ChatMessageApprovalStatus_Approved, nil
}

// awaitActionApproval waits for an approval decision for one action call and synthesizes fallback decisions on timeout or cancellation.
func (sc StreamChatImpl) awaitActionApproval(
	ctx context.Context,
	conversationID uuid.UUID,
	turnID uuid.UUID,
	actionCall domain.AssistantActionCall,
	timeout time.Duration,
) domain.AssistantActionApprovalDecision {
	key := domain.AssistantActionApprovalKey{
		ConversationID: conversationID,
		TurnID:         turnID,
		ActionCallID:   actionCall.ID,
	}

	now := sc.timeProvider.Now()
	if sc.approvalDispatcher == nil {
		reason := "approval dispatcher is not configured"
		return domain.AssistantActionApprovalDecision{
			Key:        key,
			ActionName: actionCall.Name,
			Status:     domain.ChatMessageApprovalStatus_AutoRejected,
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

	status := domain.ChatMessageApprovalStatus_AutoRejected
	reason := "approval wait canceled"
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		status = domain.ChatMessageApprovalStatus_Expired
		reason = "approval request expired"
	case errors.Is(err, context.Canceled):
		status = domain.ChatMessageApprovalStatus_AutoRejected
		reason = "approval request canceled"
	}

	return domain.AssistantActionApprovalDecision{
		Key:        key,
		ActionName: actionCall.Name,
		Status:     status,
		Reason:     &reason,
		DecidedAt:  sc.timeProvider.Now(),
	}
}

// approvalTitle returns the configured approval title or a generic fallback.
func approvalTitle(action domain.AssistantActionDefinition) string {
	if title := strings.TrimSpace(action.Approval.Title); title != "" {
		return title
	}
	return "Approval required"
}

// approvalDescription returns the configured approval description or a generic fallback.
func approvalDescription(action domain.AssistantActionDefinition) string {
	if description := strings.TrimSpace(action.Approval.Description); description != "" {
		return description
	}
	return fmt.Sprintf("Approve action '%s' execution.", action.Name)
}
