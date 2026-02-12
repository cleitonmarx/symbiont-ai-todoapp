package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteConversationImpl_Execute(t *testing.T) {
	tests := map[string]struct {
		setExpectations func(*domain.MockUnitOfWork, *domain.MockChatMessageRepository, *domain.MockConversationSummaryRepository)
		expectedErr     error
	}{
		"success": {
			expectedErr: nil,
			setExpectations: func(uow *domain.MockUnitOfWork, repo *domain.MockChatMessageRepository, summaryRepo *domain.MockConversationSummaryRepository) {
				repo.EXPECT().
					DeleteConversation(mock.Anything).
					Return(nil).
					Once()
				summaryRepo.EXPECT().
					DeleteConversationSummary(mock.Anything, domain.GlobalConversationID).
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
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(domain.UnitOfWork) error) error {
						return fn(uow)
					}).
					Once()
			},
		},
		"repository-error": {
			expectedErr: errors.New("database error"),
			setExpectations: func(uow *domain.MockUnitOfWork, repo *domain.MockChatMessageRepository, summaryRepo *domain.MockConversationSummaryRepository) {
				repo.EXPECT().
					DeleteConversation(mock.Anything).
					Return(errors.New("database error")).
					Once()

				uow.EXPECT().
					ChatMessage().
					Return(repo).
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
			uow := domain.NewMockUnitOfWork(t)
			tt.setExpectations(uow, repo, summaryRepo)

			uc := NewDeleteConversationImpl(uow)
			err := uc.Execute(t.Context())

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
