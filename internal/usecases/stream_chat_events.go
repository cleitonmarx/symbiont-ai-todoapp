package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/toon-format/toon-go"
)

// handleStreamEvent dispatches one assistant streaming event to the matching handler.
func (sc StreamChatImpl) handleStreamEvent(
	ctx context.Context,
	eventType domain.AssistantEventType,
	data any,
	model string,
	req *domain.AssistantTurnRequest,
	state *streamChatExecutionState,
	onEvent domain.AssistantEventCallback,
) (bool, error) {
	switch eventType {
	case domain.AssistantEventType_TurnStarted:
		return false, sc.handleMetaEvent(ctx, data, state, onEvent)
	case domain.AssistantEventType_ActionRequested:
		return sc.handleActionCallEvent(ctx, data, model, req, state, onEvent)
	case domain.AssistantEventType_MessageDelta:
		return false, sc.handleDeltaEvent(ctx, data, state, onEvent)
	case domain.AssistantEventType_TurnCompleted:
		sc.handleDoneEvent(data, state)
		return false, nil
	default:
		return false, nil
	}
}

// handleMetaEvent captures server-assigned message IDs and persists the user message once.
func (sc StreamChatImpl) handleMetaEvent(
	ctx context.Context,
	data any,
	state *streamChatExecutionState,
	onEvent domain.AssistantEventCallback,
) error {
	if state.assistantMsgID != uuid.Nil {
		return nil
	}

	meta := data.(domain.AssistantTurnStarted)
	meta.ConversationID = state.conversation.ID
	meta.ConversationCreated = state.conversationCreated
	state.assistantMsgID = meta.AssistantMessageID
	state.userMsg.ID = meta.UserMessageID
	state.userMsg.CreatedAt = sc.timeProvider.Now()
	state.userMsg.UpdatedAt = state.userMsg.CreatedAt
	state.userMsgPersistTried = true
	if err := sc.persistChatMessage(ctx, state.userMsg, state.conversation); err != nil {
		return err
	}
	state.userMsgPersisted = true
	return onEvent(ctx, domain.AssistantEventType_TurnStarted, meta)
}

// handleActionCallEvent persists action call/result messages and injects the outcome back into the turn request.
func (sc StreamChatImpl) handleActionCallEvent(
	ctx context.Context,
	data any,
	model string,
	req *domain.AssistantTurnRequest,
	state *streamChatExecutionState,
	onEvent domain.AssistantEventCallback,
) (bool, error) {
	actionCall := data.(domain.AssistantActionCall)
	if state.tracker.hasExceededMaxCycles() || state.tracker.hasExceededMaxActionCalls(actionCall.Name, actionCall.Input) {
		return false, nil
	}

	assistantActionCallMsg := domain.ChatMessage{
		ID:             uuid.New(),
		ConversationID: state.conversation.ID,
		TurnID:         state.turnID,
		TurnSequence:   state.nextTurnSequence(),
		ChatRole:       domain.ChatRole_Assistant,
		ActionCalls:    []domain.AssistantActionCall{actionCall},
		Model:          model,
		MessageState:   domain.ChatMessageState_Completed,
		CreatedAt:      sc.timeProvider.Now(),
	}
	assistantActionCallMsg.UpdatedAt = assistantActionCallMsg.CreatedAt
	if err := sc.persistChatMessage(ctx, assistantActionCallMsg, state.conversation); err != nil {
		return false, err
	}

	approvalDecision, blockedByApproval, approvalErr := sc.requestActionApprovalIfRequired(
		ctx,
		actionCall,
		state,
		onEvent,
	)
	if approvalErr != nil {
		return false, approvalErr
	}

	if blockedByApproval {
		reason := approvalDecisionReason(approvalDecision)
		actionContent := approvalBlockedActionContent(actionCall, approvalDecision.Status, reason)

		actionMessage := domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: common.Ptr(actionCall.ID),
			Content:      actionContent,
		}
		now := sc.timeProvider.Now()
		actionChatMsg := domain.ChatMessage{
			ID:                     uuid.New(),
			ConversationID:         state.conversation.ID,
			TurnID:                 state.turnID,
			TurnSequence:           state.nextTurnSequence(),
			ChatRole:               domain.ChatRole_Tool,
			ActionCallID:           &actionCall.ID,
			Content:                actionContent,
			Model:                  model,
			MessageState:           domain.ChatMessageState_Failed,
			ErrorMessage:           &reason,
			ApprovalStatus:         &approvalDecision.Status,
			ApprovalDecisionReason: approvalDecision.Reason,
			ApprovalDecidedAt:      common.Ptr(approvalDecision.DecidedAt),
			CreatedAt:              now,
			UpdatedAt:              now,
		}

		if err := sc.persistChatMessage(ctx, actionChatMsg, state.conversation); err != nil {
			return false, err
		}

		actionCompleted := domain.AssistantActionCompleted{
			ID:            actionCall.ID,
			Name:          actionCall.Name,
			Success:       false,
			ShouldRefetch: false,
			Error:         &reason,
		}
		if err := onEvent(ctx, domain.AssistantEventType_ActionCompleted, actionCompleted); err != nil {
			return false, err
		}

		req.Messages = append(req.Messages,
			domain.AssistantMessage{
				Role:        domain.ChatRole_Assistant,
				ActionCalls: []domain.AssistantActionCall{actionCall},
			},
			actionMessage,
		)

		return true, nil
	}

	actionCall.Text = sc.actionRegistry.StatusMessage(actionCall.Name)
	if err := onEvent(ctx, domain.AssistantEventType_ActionStarted, actionCall); err != nil {
		return false, err
	}

	actionMessage := sc.actionRegistry.Execute(ctx, actionCall, req.Messages)
	actionSucceeded := actionMessage.IsActionCallSuccess()
	now := sc.timeProvider.Now()
	actionChatMsg := domain.ChatMessage{
		ID:             uuid.New(),
		ConversationID: state.conversation.ID,
		TurnID:         state.turnID,
		TurnSequence:   state.nextTurnSequence(),
		ChatRole:       domain.ChatRole_Tool,
		ActionCallID:   &actionCall.ID,
		Content:        actionMessage.Content,
		Model:          model,
		MessageState:   domain.ChatMessageState_Completed,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if !actionSucceeded {
		actionChatMsg.MessageState = domain.ChatMessageState_Failed
		actionChatMsg.ErrorMessage = &actionMessage.Content
	}
	if approvalDecision.Status != "" {
		actionChatMsg.ApprovalStatus = &approvalDecision.Status
		actionChatMsg.ApprovalDecisionReason = approvalDecision.Reason
		actionChatMsg.ApprovalDecidedAt = common.Ptr(approvalDecision.DecidedAt)
	}

	if err := sc.persistChatMessage(ctx, actionChatMsg, state.conversation); err != nil {
		return false, err
	}

	actionCompleted := domain.AssistantActionCompleted{
		ID:            actionCall.ID,
		Name:          actionCall.Name,
		Success:       actionSucceeded,
		ShouldRefetch: actionSucceeded,
	}
	if !actionSucceeded {
		actionCompleted.Error = &actionMessage.Content
	}
	if err := onEvent(ctx, domain.AssistantEventType_ActionCompleted, actionCompleted); err != nil {
		return false, err
	}

	if actionSucceeded {
		if renderedMessage, ok := sc.renderActionResult(actionCall, actionMessage); ok {
			req.Messages = append(req.Messages,
				domain.AssistantMessage{
					Role:        domain.ChatRole_Assistant,
					ActionCalls: []domain.AssistantActionCall{actionCall},
				},
				domain.AssistantMessage{
					Role:         actionMessage.Role,
					Content:      actionMessage.Content,
					ActionCallID: actionMessage.ActionCallID,
					ActionCalls:  actionMessage.ActionCalls,
				},
				renderedMessage,
			)
			if err := sc.handleRenderedActionResult(ctx, renderedMessage, state, onEvent); err != nil {
				return false, err
			}
			return false, nil
		}
	}

	req.Messages = append(req.Messages,
		domain.AssistantMessage{
			Role:        domain.ChatRole_Assistant,
			ActionCalls: []domain.AssistantActionCall{actionCall},
		},
		domain.AssistantMessage{
			Role:         actionMessage.Role,
			Content:      actionMessage.Content,
			ActionCallID: actionMessage.ActionCallID,
			ActionCalls:  actionMessage.ActionCalls,
		},
	)
	if !actionSucceeded {
		req.Messages = append(req.Messages, domain.AssistantMessage{
			Role: domain.ChatRole_System,
			Content: "Tool call failed. Read the tool error details/example, then retry with corrected arguments or another tool. " +
				"If updating/deleting todos failed due to missing or unmatched IDs, fetch todos first to resolve UUIDs, then retry.",
		})
	}

	return true, nil
}

// approvalDecisionReason derives a human-readable explanation for an approval decision.
func approvalDecisionReason(decision domain.AssistantActionApprovalDecision) string {
	if decision.Reason != nil {
		if reason := strings.TrimSpace(*decision.Reason); reason != "" {
			return reason
		}
	}

	switch decision.Status {
	case domain.ChatMessageApprovalStatus_Expired:
		return "approval request expired"
	case domain.ChatMessageApprovalStatus_AutoRejected:
		return "approval request canceled"
	case domain.ChatMessageApprovalStatus_Rejected:
		return "action execution rejected by user"
	default:
		return "action execution was not approved"
	}
}

// approvalBlockedActionContent builds the synthetic tool payload used when approval blocks execution.
func approvalBlockedActionContent(
	actionCall domain.AssistantActionCall,
	status domain.ChatMessageApprovalStatus,
	reason string,
) string {
	type blockedPayload struct {
		ApprovalStatus domain.ChatMessageApprovalStatus `json:"approval_status"`
		ActionName     string                           `json:"action_name"`
		ActionCallID   string                           `json:"action_call_id"`
		Executed       bool                             `json:"executed"`
		Reason         string                           `json:"reason"`
		Message        string                           `json:"message"`
	}

	payload := blockedPayload{
		ApprovalStatus: status,
		ActionName:     actionCall.Name,
		ActionCallID:   actionCall.ID,
		Executed:       false,
		Reason:         reason,
		Message:        "Action execution blocked by approval policy. Do not assume this action was executed.",
	}

	data, err := toon.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(
			"Action execution blocked by approval policy. action=%s action_call_id=%s approval_status=%s reason=%s",
			actionCall.Name,
			actionCall.ID,
			status,
			reason,
		)
	}
	return string(data)
}

// handleDeltaEvent appends streamed assistant text and forwards the delta event to the caller.
func (sc StreamChatImpl) handleDeltaEvent(
	ctx context.Context,
	data any,
	state *streamChatExecutionState,
	onEvent domain.AssistantEventCallback,
) error {
	delta := data.(domain.AssistantMessageDelta)
	state.assistantMsgContent.WriteString(delta.Text)
	return onEvent(ctx, domain.AssistantEventType_MessageDelta, data)
}

// handleDoneEvent accumulates token usage reported by one completed assistant turn.
func (sc StreamChatImpl) handleDoneEvent(data any, state *streamChatExecutionState) {
	done := data.(domain.AssistantTurnCompleted)
	state.tokenUsage.CompletionTokens += done.Usage.CompletionTokens
	state.tokenUsage.PromptTokens += done.Usage.PromptTokens
	state.tokenUsage.TotalTokens += done.Usage.TotalTokens
}
