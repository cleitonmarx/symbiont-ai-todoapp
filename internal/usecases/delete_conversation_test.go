package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteConversationImpl_Execute(t *testing.T) {
	fixedConversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	tests := map[string]struct {
		setExpectations func(
			*domain.MockUnitOfWork,
			*domain.MockChatMessageRepository,
			*domain.MockConversationSummaryRepository,
			*domain.MockConversationRepository,
		)
		expectedErr error
	}{
		"success": {
			expectedErr: nil,
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				repo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				convRepo *domain.MockConversationRepository,
			) {
				convRepo.EXPECT().
					GetConversation(mock.Anything, fixedConversationID).
					Return(domain.Conversation{ID: fixedConversationID}, true, nil).
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

				uow.EXPECT().
					ChatMessage().
					Return(repo).
					Once()

				uow.EXPECT().
					ConversationSummary().
					Return(summaryRepo).
					Once()

				uow.EXPECT().
					Conversation().
					Return(convRepo).
					Twice()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
		},
		"repository-error": {
			expectedErr: errors.New("database error"),
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				repo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				convRepo *domain.MockConversationRepository,
			) {
				convRepo.EXPECT().
					GetConversation(mock.Anything, fixedConversationID).
					Return(domain.Conversation{ID: fixedConversationID}, true, nil).
					Once()
				repo.EXPECT().
					DeleteConversationMessages(mock.Anything, fixedConversationID).
					Return(errors.New("database error")).
					Once()

				uow.EXPECT().
					ChatMessage().
					Return(repo).
					Once()
				uow.EXPECT().
					Conversation().
					Return(convRepo).
					Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
		},
		"conversation-not-found": {
			expectedErr: domain.NewNotFoundErr("conversation with ID 00000000-0000-0000-0000-000000000001 not found"),
			setExpectations: func(
				uow *domain.MockUnitOfWork,
				repo *domain.MockChatMessageRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				convRepo *domain.MockConversationRepository,
			) {
				convRepo.EXPECT().
					GetConversation(mock.Anything, fixedConversationID).
					Return(domain.Conversation{}, false, nil).
					Once()
				uow.EXPECT().
					Conversation().
					Return(convRepo).
					Once()
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := domain.NewMockChatMessageRepository(t)
			summaryRepo := domain.NewMockConversationSummaryRepository(t)
			convRepo := domain.NewMockConversationRepository(t)
			uow := domain.NewMockUnitOfWork(t)
			tt.setExpectations(uow, repo, summaryRepo, convRepo)

			uc := NewDeleteConversationImpl(uow)
			err := uc.Execute(t.Context(), fixedConversationID)

			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestInitDeleteConversation_Initialize(t *testing.T) {
	idc := InitDeleteConversation{}

	_, err := idc.Initialize(context.Background())
	assert.NoError(t, err)

	uc, err := depend.Resolve[DeleteConversation]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)

}
