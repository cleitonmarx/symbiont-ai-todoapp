package usecases

import (
	"context"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
)

// ListChatMessages defines the interface for the ListChatMessages use case
type ListChatMessages interface {
	Query(ctx context.Context, conversationID uuid.UUID, page int, pageSize int) ([]domain.ChatMessage, bool, error)
}

// ListChatMessagesImpl is the implementation of the ListChatMessages use case
type ListChatMessagesImpl struct {
	ChatMessageRepo domain.ChatMessageRepository `resolve:""`
}

// NewListChatMessagesImpl creates a new instance of ListChatMessagesImpl
func NewListChatMessagesImpl(chatMessageRepo domain.ChatMessageRepository) ListChatMessagesImpl {
	return ListChatMessagesImpl{
		ChatMessageRepo: chatMessageRepo,
	}
}

// Query retrieves chat messages with pagination support
func (lcm ListChatMessagesImpl) Query(ctx context.Context, conversationID uuid.UUID, page int, pageSize int) ([]domain.ChatMessage, bool, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	messages, hasMore, err := lcm.ChatMessageRepo.ListChatMessages(spanCtx, conversationID, page, pageSize)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}

	return projectMessagesForUser(messages), hasMore, nil
}

// InitListChatMessages is the initializer for the ListChatMessages use case
type InitListChatMessages struct {
	Repo domain.ChatMessageRepository `resolve:""`
}

// Initialize registers the ListChatMessages use case in the dependency container
func (i InitListChatMessages) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListChatMessages](NewListChatMessagesImpl(i.Repo))
	return ctx, nil
}

// projectMessagesForUser applies projection rules to the list of messages for user-facing consumption.
func projectMessagesForUser(messages []domain.ChatMessage) []domain.ChatMessage {
	if len(messages) == 0 {
		return nil
	}

	messagesByTurn := make(map[uuid.UUID][]domain.ChatMessage)
	turnOrder := make([]uuid.UUID, 0, len(messages))
	for _, msg := range messages {
		if _, exists := messagesByTurn[msg.TurnID]; !exists {
			turnOrder = append(turnOrder, msg.TurnID)
		}
		messagesByTurn[msg.TurnID] = append(messagesByTurn[msg.TurnID], msg)
	}

	projected := make([]domain.ChatMessage, 0, len(messages))
	for _, turnID := range turnOrder {
		projected = append(projected, projectTurnMessages(messagesByTurn[turnID])...)
	}

	return projected
}

// projectTurnMessages applies projection rules to messages within a turn, including action call summarization.
func projectTurnMessages(messages []domain.ChatMessage) []domain.ChatMessage {
	if len(messages) == 0 {
		return nil
	}

	projected := make([]domain.ChatMessage, 0, 2)
	actionResultsByID := make(map[string]domain.ChatMessage)
	actionDetails := make([]domain.ChatMessageActionDetail, 0)
	var assistantMessage *domain.ChatMessage

	for _, msg := range messages {
		switch {
		case msg.ChatRole == domain.ChatRole_Tool && msg.ActionCallID != nil:
			actionResultsByID[*msg.ActionCallID] = msg
		case msg.ChatRole == domain.ChatRole_User && strings.TrimSpace(msg.Content) != "":
			projected = append(projected, msg)
		case msg.ChatRole == domain.ChatRole_Assistant && len(msg.ActionCalls) == 0:
			msgCopy := msg
			assistantMessage = &msgCopy
		}
	}

	for _, msg := range messages {
		if msg.ChatRole != domain.ChatRole_Assistant || len(msg.ActionCalls) == 0 {
			continue
		}
		for _, actionCall := range msg.ActionCalls {
			detail := domain.ChatMessageActionDetail{
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
func shouldReturnAssistantMessage(msg domain.ChatMessage) bool {
	return strings.TrimSpace(msg.Content) != "" ||
		msg.MessageState == domain.ChatMessageState_Failed ||
		len(msg.SelectedSkills) > 0 ||
		len(msg.ActionDetails) > 0
}
