package chat

import (
	"context"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/google/uuid"
)

// ListChatMessages defines the interface for the ListChatMessages use case
type ListChatMessages interface {
	Query(ctx context.Context, conversationID uuid.UUID, page int, pageSize int) ([]assistant.ChatMessage, bool, error)
}

// ListChatMessagesImpl is the implementation of the ListChatMessages use case
type ListChatMessagesImpl struct {
	ChatMessageRepo assistant.ChatMessageRepository `resolve:""`
}

// NewListChatMessagesImpl creates a new instance of ListChatMessagesImpl
func NewListChatMessagesImpl(chatMessageRepo assistant.ChatMessageRepository) ListChatMessagesImpl {
	return ListChatMessagesImpl{
		ChatMessageRepo: chatMessageRepo,
	}
}

// Query retrieves chat messages with pagination support
func (lcm ListChatMessagesImpl) Query(ctx context.Context, conversationID uuid.UUID, page int, pageSize int) ([]assistant.ChatMessage, bool, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	messages, hasMore, err := lcm.ChatMessageRepo.ListChatMessages(spanCtx, conversationID, page, pageSize)
	if telemetry.IsErrorRecorded(span, err) {
		return nil, false, err
	}

	return projectMessagesForUser(messages), hasMore, nil
}

// projectMessagesForUser applies projection rules to the list of messages for user-facing consumption.
func projectMessagesForUser(messages []assistant.ChatMessage) []assistant.ChatMessage {
	if len(messages) == 0 {
		return nil
	}

	messagesByTurn := make(map[uuid.UUID][]assistant.ChatMessage)
	turnOrder := make([]uuid.UUID, 0, len(messages))
	for _, msg := range messages {
		if _, exists := messagesByTurn[msg.TurnID]; !exists {
			turnOrder = append(turnOrder, msg.TurnID)
		}
		messagesByTurn[msg.TurnID] = append(messagesByTurn[msg.TurnID], msg)
	}

	projected := make([]assistant.ChatMessage, 0, len(messages))
	for _, turnID := range turnOrder {
		projected = append(projected, projectTurnMessages(messagesByTurn[turnID])...)
	}

	return projected
}

// projectTurnMessages applies projection rules to messages within a turn, including action call summarization.
func projectTurnMessages(messages []assistant.ChatMessage) []assistant.ChatMessage {
	if len(messages) == 0 {
		return nil
	}

	projected := make([]assistant.ChatMessage, 0, 2)
	actionResultsByID := make(map[string]assistant.ChatMessage)
	actionDetails := make([]assistant.ChatMessageActionDetail, 0)
	var assistantMessage *assistant.ChatMessage

	for _, msg := range messages {
		switch {
		case msg.ChatRole == assistant.ChatRole_Tool && msg.ActionCallID != nil:
			actionResultsByID[*msg.ActionCallID] = msg
		case msg.ChatRole == assistant.ChatRole_User && strings.TrimSpace(msg.Content) != "":
			projected = append(projected, msg)
		case msg.ChatRole == assistant.ChatRole_Assistant && len(msg.ActionCalls) == 0:
			msgCopy := msg
			assistantMessage = &msgCopy
		}
	}

	for _, msg := range messages {
		if msg.ChatRole != assistant.ChatRole_Assistant || len(msg.ActionCalls) == 0 {
			continue
		}
		for _, actionCall := range msg.ActionCalls {
			detail := assistant.ChatMessageActionDetail{
				ActionCallID: actionCall.ID,
				Name:         actionCall.Name,
				Input:        actionCall.Input,
				Text:         actionCall.Text,
			}
			if result, found := actionResultsByID[actionCall.ID]; found {
				detail.Output = result.Content
				detail.MessageState = result.MessageState
				detail.ErrorMessage = result.ErrorMessage
				detail.ApprovalStatus = result.ApprovalStatus
				detail.ApprovalDecisionReason = result.ApprovalDecisionReason
				detail.ApprovalDecidedAt = result.ApprovalDecidedAt
				detail.ActionExecuted = result.ActionExecuted
			}
			actionDetails = append(actionDetails, detail)
		}
	}

	if assistantMessage == nil {
		return projected
	}
	if len(actionDetails) > 0 {
		assistantMessage.ActionDetails = actionDetails
	}
	if shouldReturnAssistantMessage(*assistantMessage) {
		projected = append(projected, *assistantMessage)
	}

	return projected
}

// shouldReturnAssistantMessage determines if an assistant message should be
// included in the projected results based on content and action details.
func shouldReturnAssistantMessage(msg assistant.ChatMessage) bool {
	return strings.TrimSpace(msg.Content) != "" ||
		msg.MessageState == assistant.ChatMessageState_Failed ||
		len(msg.SelectedSkills) > 0 ||
		len(msg.ActionDetails) > 0
}
