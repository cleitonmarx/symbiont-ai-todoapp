package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

// renderActionResult converts a successful tool result into a deterministic
// assistant message when the registered action exposes a renderer.
func (sc StreamChatImpl) renderActionResult(
	actionCall domain.AssistantActionCall,
	actionMessage domain.AssistantMessage,
) (domain.AssistantMessage, bool) {
	renderer, found := sc.actionRegistry.GetRenderer(actionCall.Name)
	if !found || renderer == nil {
		return domain.AssistantMessage{}, false
	}

	return renderer.Render(actionCall, actionMessage)
}

// handleRenderedActionResult streams a deterministic assistant message and
// stores its content in the current turn state for final persistence.
func (sc StreamChatImpl) handleRenderedActionResult(
	ctx context.Context,
	rendered domain.AssistantMessage,
	state *streamChatExecutionState,
	onEvent domain.AssistantEventCallback,
) error {
	if rendered.Role != domain.ChatRole_Assistant || rendered.Content == "" {
		return nil
	}

	state.assistantMsgContent.WriteString(rendered.Content)
	return onEvent(ctx, domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{
		Text: rendered.Content,
	})
}
