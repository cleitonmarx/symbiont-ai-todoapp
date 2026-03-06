package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/google/uuid"
	"github.com/toon-format/toon-go"
)

// handleStreamEvent dispatches one assistant streaming event to the matching handler.
func (sc StreamChatImpl) handleStreamEvent(
	ctx context.Context,
	eventType assistant.EventType,
	data any,
	model string,
	req *assistant.TurnRequest,
	state *streamChatExecutionState,
	onEvent assistant.EventCallback,
) (bool, error) {
	switch eventType {
	case assistant.EventType_TurnStarted:
		return false, sc.handleMetaEvent(ctx, data, state, onEvent)
	case assistant.EventType_ActionRequested:
		return sc.handleActionCallEvent(ctx, data, model, req, state, onEvent)
	case assistant.EventType_MessageDelta:
		return false, sc.handleDeltaEvent(ctx, data, state, onEvent)
	case assistant.EventType_TurnCompleted:
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
	onEvent assistant.EventCallback,
) error {
	if state.assistantMsgID != uuid.Nil {
		return nil
	}

	meta := data.(assistant.TurnStarted)
	meta.ConversationID = state.conversation.ID
	meta.ConversationCreated = state.conversationCreated
	meta.TurnID = state.turnID
	meta.SelectedSkills = state.selectedSkills
	state.assistantMsgID = meta.AssistantMessageID
	state.userMsg.ID = meta.UserMessageID
	state.userMsg.CreatedAt = sc.timeProvider.Now()
	state.userMsg.UpdatedAt = state.userMsg.CreatedAt
	state.userMsgPersistTried = true
	if err := sc.persistChatMessage(ctx, state.userMsg, state.conversation); err != nil {
		return err
	}
	state.userMsgPersisted = true
	return onEvent(ctx, assistant.EventType_TurnStarted, meta)
}

// handleActionCallEvent persists action call/result messages and injects the outcome back into the turn request.
func (sc StreamChatImpl) handleActionCallEvent(
	ctx context.Context,
	data any,
	model string,
	req *assistant.TurnRequest,
	state *streamChatExecutionState,
	onEvent assistant.EventCallback,
) (bool, error) {
	actionCall := data.(assistant.ActionCall)
	if state.tracker.hasExceededMaxCycles() || state.tracker.hasExceededMaxActionCalls(actionCall.Name, actionCall.Input) {
		return false, nil
	}
	actionCall.Text = sc.actionRegistry.StatusMessage(actionCall.Name)

	assistantActionCallMsg := assistant.ChatMessage{
		ID:             uuid.New(),
		ConversationID: state.conversation.ID,
		TurnID:         state.turnID,
		TurnSequence:   state.nextTurnSequence(),
		ChatRole:       assistant.ChatRole_Assistant,
		ActionCalls:    []assistant.ActionCall{actionCall},
		Model:          model,
		MessageState:   assistant.ChatMessageState_Completed,
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

		actionMessage := assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: common.Ptr(actionCall.ID),
			Content:      actionContent,
		}
		now := sc.timeProvider.Now()
		actionChatMsg := assistant.ChatMessage{
			ID:                     uuid.New(),
			ConversationID:         state.conversation.ID,
			TurnID:                 state.turnID,
			TurnSequence:           state.nextTurnSequence(),
			ChatRole:               assistant.ChatRole_Tool,
			ActionCallID:           &actionCall.ID,
			Content:                actionContent,
			Model:                  model,
			MessageState:           assistant.ChatMessageState_Failed,
			ErrorMessage:           &reason,
			ApprovalStatus:         &approvalDecision.Status,
			ApprovalDecisionReason: approvalDecision.Reason,
			ApprovalDecidedAt:      common.Ptr(approvalDecision.DecidedAt),
			ActionExecuted:         common.Ptr(false),
			CreatedAt:              now,
			UpdatedAt:              now,
		}

		if err := sc.persistChatMessage(ctx, actionChatMsg, state.conversation); err != nil {
			return false, err
		}

		actionCompleted := assistant.ActionCompleted{
			ID:              actionCall.ID,
			Name:            actionCall.Name,
			Success:         false,
			ShouldRefetch:   false,
			Error:           &reason,
			ApprovalStatus:  &approvalDecision.Status,
			ActionExecuted:  common.Ptr(false),
			OutputPreview:   buildOutputPreview(actionContent),
			OutputTruncated: isOutputPreviewTruncated(actionContent),
		}
		if err := onEvent(ctx, assistant.EventType_ActionCompleted, actionCompleted); err != nil {
			return false, err
		}

		req.Messages = append(req.Messages,
			assistant.Message{
				Role:        assistant.ChatRole_Assistant,
				ActionCalls: []assistant.ActionCall{actionCall},
			},
			actionMessage,
		)

		return true, nil
	}

	if err := onEvent(ctx, assistant.EventType_ActionStarted, actionCall); err != nil {
		return false, err
	}

	actionMessage := sc.actionRegistry.Execute(ctx, actionCall, req.Messages)
	actionSucceeded := actionMessage.IsActionCallSuccess()
	now := sc.timeProvider.Now()
	actionChatMsg := assistant.ChatMessage{
		ID:             uuid.New(),
		ConversationID: state.conversation.ID,
		TurnID:         state.turnID,
		TurnSequence:   state.nextTurnSequence(),
		ChatRole:       assistant.ChatRole_Tool,
		ActionCallID:   &actionCall.ID,
		Content:        actionMessage.Content,
		Model:          model,
		MessageState:   assistant.ChatMessageState_Completed,
		ActionExecuted: common.Ptr(true),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if !actionSucceeded {
		actionChatMsg.MessageState = assistant.ChatMessageState_Failed
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

	actionCompleted := assistant.ActionCompleted{
		ID:              actionCall.ID,
		Name:            actionCall.Name,
		Success:         actionSucceeded,
		ShouldRefetch:   actionSucceeded,
		ApprovalStatus:  &approvalDecision.Status,
		ActionExecuted:  common.Ptr(true),
		OutputPreview:   buildOutputPreview(actionMessage.Content),
		OutputTruncated: isOutputPreviewTruncated(actionMessage.Content),
	}
	if !actionSucceeded {
		actionCompleted.Error = &actionMessage.Content
	}
	if err := onEvent(ctx, assistant.EventType_ActionCompleted, actionCompleted); err != nil {
		return false, err
	}

	if actionSucceeded {
		if renderedMessage, ok := sc.renderActionResult(actionCall, actionMessage); ok {
			req.Messages = append(req.Messages,
				assistant.Message{
					Role:        assistant.ChatRole_Assistant,
					ActionCalls: []assistant.ActionCall{actionCall},
				},
				assistant.Message{
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
		assistant.Message{
			Role:        assistant.ChatRole_Assistant,
			ActionCalls: []assistant.ActionCall{actionCall},
		},
		assistant.Message{
			Role:         actionMessage.Role,
			Content:      actionMessage.Content,
			ActionCallID: actionMessage.ActionCallID,
			ActionCalls:  actionMessage.ActionCalls,
		},
	)
	if !actionSucceeded {
		req.Messages = append(req.Messages, assistant.Message{
			Role: assistant.ChatRole_System,
			Content: "Tool call failed. Read the tool error details/example, then retry with corrected arguments or another tool. " +
				"If updating/deleting todos failed due to missing or unmatched IDs, fetch todos first to resolve UUIDs, then retry.",
		})
	}

	return true, nil
}

// approvalDecisionReason derives a human-readable explanation for an approval decision.
func approvalDecisionReason(decision assistant.ActionApprovalDecision) string {
	if decision.Reason != nil {
		if reason := strings.TrimSpace(*decision.Reason); reason != "" {
			return reason
		}
	}

	switch decision.Status {
	case assistant.ChatMessageApprovalStatus_Expired:
		return "approval request expired"
	case assistant.ChatMessageApprovalStatus_AutoRejected:
		return "approval request canceled"
	case assistant.ChatMessageApprovalStatus_Rejected:
		return "action execution rejected by user"
	default:
		return "action execution was not approved"
	}
}

// actionOutputPreviewMaxChars defines the maximum number of characters for the action output preview.
const actionOutputPreviewMaxChars = 4000

// buildOutputPreview generates a truncated preview of the action output for display
// in the UI and to assist with token limits.
func buildOutputPreview(content string) *string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	preview := truncateToFirstChars(trimmed, actionOutputPreviewMaxChars)
	return common.Ptr(preview)
}

// isOutputPreviewTruncated checks if the action output exceeds the maximum character limit for the preview.
func isOutputPreviewTruncated(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	return len([]rune(trimmed)) > actionOutputPreviewMaxChars
}

// approvalBlockedActionContent builds the synthetic tool payload used when approval blocks execution.
func approvalBlockedActionContent(
	actionCall assistant.ActionCall,
	status assistant.ChatMessageApprovalStatus,
	reason string,
) string {
	type blockedPayload struct {
		ApprovalStatus assistant.ChatMessageApprovalStatus `json:"approval_status"`
		ActionName     string                              `json:"action_name"`
		ActionCallID   string                              `json:"action_call_id"`
		Executed       bool                                `json:"executed"`
		Reason         string                              `json:"reason"`
		Message        string                              `json:"message"`
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
	onEvent assistant.EventCallback,
) error {
	delta := data.(assistant.MessageDelta)
	state.assistantMsgContent.WriteString(delta.Text)
	return onEvent(ctx, assistant.EventType_MessageDelta, data)
}

// handleDoneEvent accumulates token usage reported by one completed assistant turn.
func (sc StreamChatImpl) handleDoneEvent(data any, state *streamChatExecutionState) {
	done := data.(assistant.TurnCompleted)
	state.tokenUsage.CompletionTokens += done.Usage.CompletionTokens
	state.tokenUsage.PromptTokens += done.Usage.PromptTokens
	state.tokenUsage.TotalTokens += done.Usage.TotalTokens
}
