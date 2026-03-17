package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/google/uuid"
	"github.com/toon-format/toon-go"
)

// ActionPipeline handles assistant-requested actions within an in-flight streamed turn.
type ActionPipeline interface {
	// Handle processes one requested action and reports whether the turn should continue.
	Handle(
		ctx context.Context,
		actionCall assistant.ActionCall,
		state TurnState,
		onEvent assistant.EventCallback,
	) (bool, error)
}

// ActionPipelineImpl implements ActionPipeline.
type ActionPipelineImpl struct {
	actionRegistry     assistant.ActionRegistry
	approvalDispatcher assistant.ActionApprovalDispatcher
	transcriptWriter   ConversationTranscriptWriter
	timeProvider       core.CurrentTimeProvider
}

// NewActionPipelineImpl creates an ActionPipelineImpl.
func NewActionPipelineImpl(
	actionRegistry assistant.ActionRegistry,
	approvalDispatcher assistant.ActionApprovalDispatcher,
	transcriptWriter ConversationTranscriptWriter,
	timeProvider core.CurrentTimeProvider,
) ActionPipelineImpl {
	return ActionPipelineImpl{
		actionRegistry:     actionRegistry,
		approvalDispatcher: approvalDispatcher,
		transcriptWriter:   transcriptWriter,
		timeProvider:       timeProvider,
	}
}

// Handle implements ActionPipeline.
func (p ActionPipelineImpl) Handle(
	ctx context.Context,
	actionCall assistant.ActionCall,
	state TurnState,
	onEvent assistant.EventCallback,
) (bool, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	if state.HasExceededMaxActionCycles() || state.HasExceededRepeatedActionCalls(actionCall.Name, actionCall.Input) {
		return false, nil
	}
	actionCall.Text = p.actionRegistry.StatusMessage(actionCall.Name)

	conversation := state.Conversation()
	assistantActionCallMsg := assistant.ChatMessage{
		ID:             uuid.New(),
		ConversationID: conversation.ID,
		TurnID:         state.TurnID(),
		TurnSequence:   state.NextTurnSequence(),
		ChatRole:       assistant.ChatRole_Assistant,
		ActionCalls:    []assistant.ActionCall{actionCall},
		Model:          state.Model(),
		MessageState:   assistant.ChatMessageState_Completed,
		CreatedAt:      p.timeProvider.Now(),
	}
	assistantActionCallMsg.UpdatedAt = assistantActionCallMsg.CreatedAt
	if err := p.transcriptWriter.WriteMessage(spanCtx, conversation, assistantActionCallMsg); err != nil {
		return false, err
	}

	approvalDecision, blockedByApproval, approvalErr := p.requestApprovalIfRequired(
		spanCtx,
		actionCall,
		state,
		onEvent,
	)
	if approvalErr != nil {
		return false, approvalErr
	}

	if blockedByApproval {
		return p.handleBlockedAction(spanCtx, actionCall, state, onEvent, approvalDecision)
	}

	if err := onEvent(spanCtx, assistant.EventType_ActionStarted, actionCall); err != nil {
		return false, err
	}

	request := state.Request()
	actionMessage := p.actionRegistry.Execute(spanCtx, actionCall, request.Messages)
	actionSucceeded := actionMessage.IsActionCallSuccess()
	now := p.timeProvider.Now()
	actionChatMsg := assistant.ChatMessage{
		ID:             uuid.New(),
		ConversationID: conversation.ID,
		TurnID:         state.TurnID(),
		TurnSequence:   state.NextTurnSequence(),
		ChatRole:       assistant.ChatRole_Tool,
		ActionCallID:   &actionCall.ID,
		Content:        actionMessage.Content,
		Model:          state.Model(),
		MessageState:   assistant.ChatMessageState_Completed,
		ActionExecuted: common.Ptr(true),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if !actionSucceeded {
		actionChatMsg.MessageState = assistant.ChatMessageState_Failed
		actionChatMsg.ErrorMessage = resolveActionErrorMessage(actionMessage)
	}
	if approvalDecision.Status != "" {
		actionChatMsg.ApprovalStatus = &approvalDecision.Status
		actionChatMsg.ApprovalDecisionReason = approvalDecision.Reason
		actionChatMsg.ApprovalDecidedAt = common.Ptr(approvalDecision.DecidedAt)
	}

	if err := p.transcriptWriter.WriteMessage(spanCtx, conversation, actionChatMsg); err != nil {
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
		actionCompleted.Error = resolveActionErrorMessage(actionMessage)
	}
	if err := onEvent(spanCtx, assistant.EventType_ActionCompleted, actionCompleted); err != nil {
		return false, err
	}

	if actionSucceeded {
		if renderedMessage, ok := p.renderActionResult(actionCall, actionMessage); ok {
			state.AppendRequestMessages(
				assistant.Message{
					Role:        assistant.ChatRole_Assistant,
					ActionCalls: []assistant.ActionCall{actionCall},
				},
				actionMessage,
				renderedMessage,
			)
			if err := p.streamRenderedMessage(spanCtx, renderedMessage, state, onEvent); err != nil {
				return false, err
			}
			return true, nil
		}
	}

	messages := []assistant.Message{
		{
			Role:        assistant.ChatRole_Assistant,
			ActionCalls: []assistant.ActionCall{actionCall},
		},
		{
			Role:         actionMessage.Role,
			Content:      actionMessage.Content,
			ActionCallID: actionMessage.ActionCallID,
			ActionCalls:  actionMessage.ActionCalls,
			ActionError:  actionMessage.ActionError,
		},
	}
	if !actionSucceeded {
		messages = append(messages, assistant.Message{
			Role: assistant.ChatRole_System,
			Content: "Tool call failed. Read the tool error details/example, then retry with corrected arguments or another tool. " +
				"If updating/deleting todos failed due to missing or unmatched IDs, fetch todos first to resolve UUIDs, then retry.",
		})
	}
	state.AppendRequestMessages(messages...)

	return true, nil
}

// handleBlockedAction persists and emits the synthetic tool result produced when approval blocks execution.
func (p ActionPipelineImpl) handleBlockedAction(
	ctx context.Context,
	actionCall assistant.ActionCall,
	state TurnState,
	onEvent assistant.EventCallback,
	approvalDecision assistant.ActionApprovalDecision,
) (bool, error) {
	reason := approvalDecisionReason(approvalDecision)
	actionContent := approvalBlockedActionContent(actionCall, approvalDecision.Status, reason)

	actionMessage := assistant.Message{
		Role:         assistant.ChatRole_Tool,
		ActionCallID: common.Ptr(actionCall.ID),
		Content:      actionContent,
		ActionError:  common.Ptr(reason),
	}
	now := p.timeProvider.Now()
	conversation := state.Conversation()
	actionChatMsg := assistant.ChatMessage{
		ID:                     uuid.New(),
		ConversationID:         conversation.ID,
		TurnID:                 state.TurnID(),
		TurnSequence:           state.NextTurnSequence(),
		ChatRole:               assistant.ChatRole_Tool,
		ActionCallID:           &actionCall.ID,
		Content:                actionContent,
		Model:                  state.Model(),
		MessageState:           assistant.ChatMessageState_Failed,
		ErrorMessage:           &reason,
		ApprovalStatus:         &approvalDecision.Status,
		ApprovalDecisionReason: approvalDecision.Reason,
		ApprovalDecidedAt:      common.Ptr(approvalDecision.DecidedAt),
		ActionExecuted:         common.Ptr(false),
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if err := p.transcriptWriter.WriteMessage(ctx, conversation, actionChatMsg); err != nil {
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

	state.AppendRequestMessages(
		assistant.Message{
			Role:        assistant.ChatRole_Assistant,
			ActionCalls: []assistant.ActionCall{actionCall},
		},
		actionMessage,
	)

	return true, nil
}

// resolveActionErrorMessage extracts the best available machine-readable tool error.
func resolveActionErrorMessage(message assistant.Message) *string {
	if message.ActionError != nil {
		return message.ActionError
	}
	if strings.TrimSpace(message.Content) == "" {
		return nil
	}
	return &message.Content
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

// actionOutputPreviewMaxChars bounds the UI preview stored for action output.
const actionOutputPreviewMaxChars = 4000

// buildOutputPreview returns a truncated UI preview for action output.
func buildOutputPreview(content string) *string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	preview := truncateToFirstChars(trimmed, actionOutputPreviewMaxChars)
	return common.Ptr(preview)
}

// isOutputPreviewTruncated reports whether the preview omits trailing action output.
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

// requestApprovalIfRequired emits approval events and waits for a decision when the action requires approval.
func (p ActionPipelineImpl) requestApprovalIfRequired(
	ctx context.Context,
	actionCall assistant.ActionCall,
	state TurnState,
	onEvent assistant.EventCallback,
) (assistant.ActionApprovalDecision, bool, error) {
	if p.approvalDispatcher == nil {
		return assistant.ActionApprovalDecision{}, false, nil
	}

	definition, found := p.actionRegistry.GetDefinition(actionCall.Name)
	if !found || !definition.RequiresApproval() {
		return assistant.ActionApprovalDecision{}, false, nil
	}

	conversation := state.Conversation()
	approvalEvent := assistant.ActionApprovalRequired{
		ConversationID: conversation.ID,
		TurnID:         state.TurnID(),
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

	decision := p.awaitActionApproval(
		ctx,
		conversation.ID,
		state.TurnID(),
		actionCall,
		definition.Approval.Timeout,
	)

	resolved := assistant.ActionApprovalResolved{
		ConversationID: conversation.ID,
		TurnID:         state.TurnID(),
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

// awaitActionApproval waits for one approval decision and synthesizes timeout or cancellation fallbacks.
func (p ActionPipelineImpl) awaitActionApproval(
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

	waitCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	decision, err := p.approvalDispatcher.Wait(waitCtx, key)
	if err == nil {
		if decision.DecidedAt.IsZero() {
			decision.DecidedAt = p.timeProvider.Now()
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
		DecidedAt:  p.timeProvider.Now(),
	}
}

// renderActionResult converts a successful tool result into a deterministic assistant message when available.
func (p ActionPipelineImpl) renderActionResult(
	actionCall assistant.ActionCall,
	actionMessage assistant.Message,
) (assistant.Message, bool) {
	renderer, found := p.actionRegistry.GetRenderer(actionCall.Name)
	if !found || renderer == nil {
		return assistant.Message{}, false
	}

	return renderer.Render(actionCall, actionMessage)
}

// streamRenderedMessage emits a deterministic assistant message and stores its content for final persistence.
func (p ActionPipelineImpl) streamRenderedMessage(
	ctx context.Context,
	rendered assistant.Message,
	state TurnState,
	onEvent assistant.EventCallback,
) error {
	if rendered.Role != assistant.ChatRole_Assistant || rendered.Content == "" {
		return nil
	}

	state.AppendAssistantContent(rendered.Content)
	return onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{
		Text: rendered.Content,
	})
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
