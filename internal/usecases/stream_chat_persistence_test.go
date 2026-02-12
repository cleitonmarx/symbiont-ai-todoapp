package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// persistCallExpectation describes one expected CreateChatMessages call.
type persistCallExpectation struct {
	Role             domain.ChatRole
	Content          string
	ID               *uuid.UUID
	MessageState     domain.ChatMessageState
	ErrorMessage     *string
	PromptTokens     *int
	CompletionTokens *int
	TotalTokens      *int
	ToolCallsLen     int
	HasToolCallID    bool
	CreateErr        error
}

// expectNowCalls enforces an exact number of current-time reads.
func expectNowCalls(timeProvider *domain.MockCurrentTimeProvider, fixedTime time.Time, times int) {
	timeProvider.EXPECT().
		Now().
		Return(fixedTime).
		Times(times)
}

// expectPersistSequence validates message persistence and matching outbox payloads in order.
func expectPersistSequence(
	t *testing.T,
	chatRepo *domain.MockChatMessageRepository,
	uow *domain.MockUnitOfWork,
	outbox *domain.MockOutboxRepository,
	fixedTime time.Time,
	expectations []persistCallExpectation,
) {
	t.Helper()

	successCount := 0
	for _, exp := range expectations {
		if exp.CreateErr == nil {
			successCount++
		}
	}

	uow.EXPECT().
		Execute(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
			return fn(uow)
		}).
		Times(len(expectations))

	uow.EXPECT().
		ChatMessage().
		Return(chatRepo).
		Times(len(expectations))

	uow.EXPECT().
		Outbox().
		Return(outbox).
		Times(successCount)

	var (
		createIdx         int
		successfulMessage []domain.ChatMessage
	)

	chatRepo.EXPECT().
		CreateChatMessages(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, msgs []domain.ChatMessage) error {
			assert.Len(t, msgs, 1)
			msg := msgs[0]
			exp := expectations[createIdx]
			createIdx++

			expectedState := exp.MessageState
			if expectedState == "" {
				expectedState = domain.ChatMessageState_Completed
			}

			assert.Equal(t, exp.Role, msg.ChatRole)
			assert.Equal(t, exp.Content, msg.Content)
			assert.Equal(t, expectedState, msg.MessageState)
			assert.Equal(t, exp.HasToolCallID, msg.ToolCallID != nil)
			assert.Len(t, msg.ToolCalls, exp.ToolCallsLen)
			assert.NotEqual(t, uuid.Nil, msg.TurnID)
			assert.Equal(t, int64(createIdx-1), msg.TurnSequence)
			assert.Equal(t, fixedTime, msg.CreatedAt)
			assert.Equal(t, fixedTime, msg.UpdatedAt)
			if exp.PromptTokens != nil {
				assert.Equal(t, *exp.PromptTokens, msg.PromptTokens)
			}
			if exp.CompletionTokens != nil {
				assert.Equal(t, *exp.CompletionTokens, msg.CompletionTokens)
			}
			if exp.TotalTokens != nil {
				assert.Equal(t, *exp.TotalTokens, msg.TotalTokens)
			}

			if exp.ID != nil {
				assert.Equal(t, *exp.ID, msg.ID)
			} else {
				assert.NotEqual(t, uuid.Nil, msg.ID)
			}

			if exp.ErrorMessage != nil {
				if assert.NotNil(t, msg.ErrorMessage) {
					assert.Equal(t, *exp.ErrorMessage, *msg.ErrorMessage)
				}
			} else {
				assert.Nil(t, msg.ErrorMessage)
			}

			if exp.CreateErr == nil {
				successfulMessage = append(successfulMessage, msg)
			}

			return exp.CreateErr
		}).
		Times(len(expectations))

	outboxCallIndex := 0
	outbox.EXPECT().
		CreateChatEvent(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, event domain.ChatMessageEvent) error {
			msg := successfulMessage[outboxCallIndex]
			outboxCallIndex++

			assert.Equal(t, domain.EventType_CHAT_MESSAGE_SENT, event.Type)
			assert.Equal(t, msg.ChatRole, event.ChatRole)
			assert.Equal(t, msg.ID, event.ChatMessageID)
			assert.Equal(t, msg.ConversationID, event.ConversationID)

			return nil
		}).
		Times(successCount)
}
