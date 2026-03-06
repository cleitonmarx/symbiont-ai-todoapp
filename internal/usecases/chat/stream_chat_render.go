package chat

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
)

// renderActionResult converts a successful tool result into a deterministic
// assistant message when the registered action exposes a renderer.
func (sc StreamChatImpl) renderActionResult(
	actionCall assistant.ActionCall,
	actionMessage assistant.Message,
) (assistant.Message, bool) {
	renderer, found := sc.actionRegistry.GetRenderer(actionCall.Name)
	if !found || renderer == nil {
		return assistant.Message{}, false
	}

	return renderer.Render(actionCall, actionMessage)
}

// handleRenderedActionResult streams a deterministic assistant message and
// stores its content in the current turn state for final persistence.
func (sc StreamChatImpl) handleRenderedActionResult(
	ctx context.Context,
	rendered assistant.Message,
	state *streamChatExecutionState,
	onEvent assistant.EventCallback,
) error {
	if rendered.Role != assistant.ChatRole_Assistant || rendered.Content == "" {
		return nil
	}

	state.assistantMsgContent.WriteString(rendered.Content)
	return onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{
		Text: rendered.Content,
	})
}
