package chat

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteConversationImpl_Execute(t *testing.T) {
	t.Parallel()

	fixedConversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	tests := map[string]struct {
		setExpectations func(
			*transaction.MockUnitOfWork,
			*assistant.MockChatMessageRepository,
			*assistant.MockConversationSummaryRepository,
			*assistant.MockConversationRepository,
		)
		expectedErr error
	}{
		"success": {
			expectedErr: nil,
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				repo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				convRepo *assistant.MockConversationRepository,
			) {
				convRepo.EXPECT().
					GetConversation(mock.Anything, fixedConversationID).
					Return(assistant.Conversation{ID: fixedConversationID}, true, nil).
					Once()
				repo.EXPECT().
					DeleteConversationMessages(mock.Anything, fixedConversationID).
					Return(nil).
					Once()
				summaryRepo.EXPECT().
					DeleteConversationSummary(mock.Anything, fixedConversationID).
					Return(nil).
					Once()
				convRepo.EXPECT().
					DeleteConversation(mock.Anything, fixedConversationID).
					Return(nil).
					Once()

				scope := transaction.NewMockScope(t)
				scope.EXPECT().ChatMessage().Return(repo).Once()
				scope.EXPECT().ConversationSummary().Return(summaryRepo).Once()
				scope.EXPECT().Conversation().Return(convRepo).Twice()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					}).
					Once()
			},
		},
		"repository-error": {
			expectedErr: errors.New("database error"),
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				repo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				convRepo *assistant.MockConversationRepository,
			) {
				convRepo.EXPECT().
					GetConversation(mock.Anything, fixedConversationID).
					Return(assistant.Conversation{ID: fixedConversationID}, true, nil).
					Once()
				repo.EXPECT().
					DeleteConversationMessages(mock.Anything, fixedConversationID).
					Return(errors.New("database error")).
					Once()

				scope := transaction.NewMockScope(t)
				scope.EXPECT().ChatMessage().Return(repo).Once()
				scope.EXPECT().Conversation().Return(convRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(uowCtx context.Context, scope transaction.Scope) error) error {
						return fn(ctx, scope)
					}).
					Once()
			},
		},
		"conversation-not-found": {
			expectedErr: core.NewNotFoundErr("conversation with ID 00000000-0000-0000-0000-000000000001 not found"),
			setExpectations: func(
				uow *transaction.MockUnitOfWork,
				repo *assistant.MockChatMessageRepository,
				summaryRepo *assistant.MockConversationSummaryRepository,
				convRepo *assistant.MockConversationRepository,
			) {
				convRepo.EXPECT().
					GetConversation(mock.Anything, fixedConversationID).
					Return(assistant.Conversation{}, false, nil).
					Once()
				scope := transaction.NewMockScope(t)
				scope.EXPECT().Conversation().Return(convRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					}).
					Once()
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := assistant.NewMockChatMessageRepository(t)
			summaryRepo := assistant.NewMockConversationSummaryRepository(t)
			convRepo := assistant.NewMockConversationRepository(t)
			uow := transaction.NewMockUnitOfWork(t)
			tt.setExpectations(uow, repo, summaryRepo, convRepo)

			uc := NewDeleteConversationImpl(uow)
			err := uc.Execute(t.Context(), fixedConversationID)

			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
