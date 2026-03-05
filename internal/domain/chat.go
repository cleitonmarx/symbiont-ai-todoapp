package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ChatRole represents the role of a chat message
type ChatRole string

const (
	ChatRole_User      ChatRole = "user"
	ChatRole_Assistant ChatRole = "assistant"
	ChatRole_System    ChatRole = "system"
	ChatRole_Developer ChatRole = "developer"
	ChatRole_Tool      ChatRole = "tool"
)

// ChatMessageState represents the persistence state of a chat message.
type ChatMessageState string

const (
	// ChatMessageState_Completed indicates the message was fully generated and persisted.
	ChatMessageState_Completed ChatMessageState = "COMPLETED"
	// ChatMessageState_Failed indicates message generation failed.
	ChatMessageState_Failed ChatMessageState = "FAILED"
)

// ChatMessageApprovalStatus represents the approval lifecycle status for a tool call message.
type ChatMessageApprovalStatus string

const (
	// ChatMessageApprovalStatus_Pending indicates an approval is still awaiting a decision.
	ChatMessageApprovalStatus_Pending ChatMessageApprovalStatus = "PENDING"
	// ChatMessageApprovalStatus_Approved indicates an approval was accepted.
	ChatMessageApprovalStatus_Approved ChatMessageApprovalStatus = "APPROVED"
	// ChatMessageApprovalStatus_Rejected indicates an approval was rejected by the user.
	ChatMessageApprovalStatus_Rejected ChatMessageApprovalStatus = "REJECTED"
	// ChatMessageApprovalStatus_AutoRejected indicates an approval was automatically rejected by the system.
	ChatMessageApprovalStatus_AutoRejected ChatMessageApprovalStatus = "AUTO_REJECTED"
	// ChatMessageApprovalStatus_Expired indicates an approval request timed out.
	ChatMessageApprovalStatus_Expired ChatMessageApprovalStatus = "EXPIRED"
)

// ChatMessage represents an AI chat message in a conversation
type ChatMessage struct {
	ID                     uuid.UUID
	ConversationID         uuid.UUID
	TurnID                 uuid.UUID
	TurnSequence           int64
	ChatRole               ChatRole
	Content                string
	ActionCallID           *string
	ActionCalls            []AssistantActionCall
	Model                  string
	MessageState           ChatMessageState
	ErrorMessage           *string
	ApprovalStatus         *ChatMessageApprovalStatus
	ApprovalDecisionReason *string
	ApprovalDecidedAt      *time.Time
	SelectedSkills         []AssistantSelectedSkill
	ActionExecuted         *bool
	ActionDetails          []ChatMessageActionDetail
	PromptTokens           int
	CompletionTokens       int
	TotalTokens            int
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// ChatMessageActionDetail summarizes one assistant action call for chat-history projections.
type ChatMessageActionDetail struct {
	ActionCallID           string
	Name                   string
	Input                  string
	Text                   string
	Output                 string
	MessageState           ChatMessageState
	ErrorMessage           *string
	ApprovalStatus         *ChatMessageApprovalStatus
	ApprovalDecisionReason *string
	ApprovalDecidedAt      *time.Time
	ActionExecuted         *bool
}

// IsActionCallSuccess returns true if the message represents a successful action call result.
func (m ChatMessage) IsActionCallSuccess() bool {
	return (m.ActionExecuted == nil || *m.ActionExecuted) &&
		m.ChatRole == ChatRole_Tool &&
		m.ActionCallID != nil &&
		m.MessageState == ChatMessageState_Completed
}

// IsApprovalPending returns true when the message is waiting for a human approval decision.
func (m ChatMessage) IsApprovalPending() bool {
	return m.ApprovalStatus != nil && *m.ApprovalStatus == ChatMessageApprovalStatus_Pending
}

// ListChatMessagesParams defines optional filters for listing chat messages.
type ListChatMessagesParams struct {
	AfterMessageID *uuid.UUID
}

// ListChatMessagesOption configures optional filters for listing chat messages.
type ListChatMessagesOption func(*ListChatMessagesParams)

// WithChatMessagesAfterMessageID filters the query to return messages after a checkpoint message ID.
func WithChatMessagesAfterMessageID(messageID uuid.UUID) ListChatMessagesOption {
	return func(options *ListChatMessagesParams) {
		options.AfterMessageID = &messageID
	}
}

// ChatMessageRepository defines the interface for chat message persistence
type ChatMessageRepository interface {
	// CreateChatMessages persists chat messages for a conversation
	CreateChatMessages(ctx context.Context, messages []ChatMessage) error

	// ListChatMessages retrieves paginated chat messages for a conversation, with optional filters.
	ListChatMessages(ctx context.Context, conversationID uuid.UUID, page int, pageSize int, options ...ListChatMessagesOption) ([]ChatMessage, bool, error)

	// DeleteConversationMessages removes all messages for a conversation.
	DeleteConversationMessages(ctx context.Context, conversationID uuid.UUID) error
}
