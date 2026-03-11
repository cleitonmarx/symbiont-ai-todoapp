package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpdateConversationImpl_Execute(t *testing.T) {
	t.Parallel()

	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	newTitle := "Updated Conversation Title"

	expectedConversation := assistant.Conversation{
		ID:          fixedUUID,
		Title:       newTitle,
		TitleSource: assistant.ConversationTitleSource_User,
		CreatedAt:   fixedTime,
		UpdatedAt:   fixedTime,
	}

	tests := map[string]struct {
		conversationID  uuid.UUID
		title           string
		setExpectations func(uow *transaction.MockUnitOfWork, timeProvider *core.MockCurrentTimeProvider)
		expectedConv    assistant.Conversation
		expectedErr     error
	}{
		"success-update-title": {
			conversationID: fixedUUID,
			title:          newTitle,
			setExpectations: func(uow *transaction.MockUnitOfWork, timeProvider *core.MockCurrentTimeProvider) {
				mockConvRepo := assistant.NewMockConversationRepository(t)
				mockConvRepo.EXPECT().
					GetConversation(mock.Anything, fixedUUID).
					Return(assistant.Conversation{
						ID:          fixedUUID,
						Title:       "Old Title",
						TitleSource: assistant.ConversationTitleSource_Auto,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					}, true, nil)

				mockConvRepo.EXPECT().
					UpdateConversation(mock.Anything, mock.MatchedBy(func(c assistant.Conversation) bool {
						return c.Title == newTitle &&
							c.TitleSource == assistant.ConversationTitleSource_User &&
							c.UpdatedAt.Equal(fixedTime)
					})).
					Return(nil)
				timeProvider.EXPECT().Now().Return(fixedTime).Once()

				scope := transaction.NewMockScope(t)
				scope.EXPECT().Conversation().Return(mockConvRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					})
			},
			expectedConv: expectedConversation,
			expectedErr:  nil,
		},
		"error-conversation-not-found": {
			conversationID: fixedUUID,
			title:          newTitle,
			setExpectations: func(uow *transaction.MockUnitOfWork, timeProvider *core.MockCurrentTimeProvider) {
				mockConvRepo := assistant.NewMockConversationRepository(t)
				mockConvRepo.EXPECT().
					GetConversation(mock.Anything, fixedUUID).
					Return(assistant.Conversation{}, false, nil)

				scope := transaction.NewMockScope(t)
				scope.EXPECT().Conversation().Return(mockConvRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					})
			},
			expectedConv: assistant.Conversation{},
			expectedErr:  core.NewNotFoundErr("conversation with ID 123e4567-e89b-12d3-a456-426614174000 not found"),
		},
		"error-get-conversation-failure": {
			conversationID: fixedUUID,
			title:          newTitle,
			setExpectations: func(uow *transaction.MockUnitOfWork, timeProvider *core.MockCurrentTimeProvider) {
				mockConvRepo := assistant.NewMockConversationRepository(t)
				mockConvRepo.EXPECT().
					GetConversation(mock.Anything, fixedUUID).
					Return(assistant.Conversation{}, false, errors.New("database error"))

				scope := transaction.NewMockScope(t)
				scope.EXPECT().Conversation().Return(mockConvRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					})
			},
			expectedConv: assistant.Conversation{},
			expectedErr:  errors.New("database error"),
		},
		"error-validation-empty-title": {
			conversationID: fixedUUID,
			title:          "",
			setExpectations: func(uow *transaction.MockUnitOfWork, timeProvider *core.MockCurrentTimeProvider) {
				mockConvRepo := assistant.NewMockConversationRepository(t)
				mockConvRepo.EXPECT().
					GetConversation(mock.Anything, fixedUUID).
					Return(assistant.Conversation{
						ID:          fixedUUID,
						Title:       "Old Title",
						TitleSource: assistant.ConversationTitleSource_User,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					}, true, nil)

				scope := transaction.NewMockScope(t)
				scope.EXPECT().Conversation().Return(mockConvRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					})
			},
			expectedConv: assistant.Conversation{},
			expectedErr:  core.NewValidationErr("conversation title cannot be empty"),
		},
		"error-update-conversation-failure": {
			conversationID: fixedUUID,
			title:          newTitle,
			setExpectations: func(uow *transaction.MockUnitOfWork, timeProvider *core.MockCurrentTimeProvider) {
				mockConvRepo := assistant.NewMockConversationRepository(t)
				mockConvRepo.EXPECT().
					GetConversation(mock.Anything, fixedUUID).
					Return(assistant.Conversation{
						ID:          fixedUUID,
						Title:       "Old Title",
						TitleSource: assistant.ConversationTitleSource_User,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					}, true, nil)

				mockConvRepo.EXPECT().
					UpdateConversation(mock.Anything, mock.Anything).
					Return(errors.New("update failed"))
				timeProvider.EXPECT().Now().Return(fixedTime).Once()

				scope := transaction.NewMockScope(t)
				scope.EXPECT().Conversation().Return(mockConvRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					})
			},
			expectedConv: assistant.Conversation{},
			expectedErr:  errors.New("update failed"),
		},
		"error-uow-execute-failure": {
			conversationID: fixedUUID,
			title:          newTitle,
			setExpectations: func(uow *transaction.MockUnitOfWork, timeProvider *core.MockCurrentTimeProvider) {
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					Return(errors.New("transaction failed"))
			},
			expectedConv: assistant.Conversation{},
			expectedErr:  errors.New("transaction failed"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := transaction.NewMockUnitOfWork(t)
			timeProvider := core.NewMockCurrentTimeProvider(t)
			if tt.setExpectations != nil {
				tt.setExpectations(uow, timeProvider)
			}

			uc := NewUpdateConversationImpl(uow, timeProvider)

			got, gotErr := uc.Execute(t.Context(), tt.conversationID, tt.title)
			if tt.expectedErr != nil {
				assert.Error(t, gotErr)
				assert.Equal(t, tt.expectedErr.Error(), gotErr.Error())
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tt.expectedConv.ID, got.ID)
				assert.Equal(t, tt.expectedConv.Title, got.Title)
				assert.Equal(t, tt.expectedConv.TitleSource, got.TitleSource)
			}
		})
	}
}
