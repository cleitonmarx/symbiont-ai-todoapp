package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpdateConversationImpl_Execute(t *testing.T) {
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	newTitle := "Updated Conversation Title"

	expectedConversation := domain.Conversation{
		ID:          fixedUUID,
		Title:       newTitle,
		TitleSource: domain.ConversationTitleSource_User,
		CreatedAt:   fixedTime,
		UpdatedAt:   fixedTime,
	}

	tests := map[string]struct {
		conversationID  uuid.UUID
		title           string
		setExpectations func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider)
		expectedConv    domain.Conversation
		expectedErr     error
	}{
		"success-update-title": {
			conversationID: fixedUUID,
			title:          newTitle,
			setExpectations: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider) {
				mockConvRepo := domain.NewMockConversationRepository(t)
				mockConvRepo.EXPECT().
					GetConversation(mock.Anything, fixedUUID).
					Return(domain.Conversation{
						ID:          fixedUUID,
						Title:       "Old Title",
						TitleSource: domain.ConversationTitleSource_Auto,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					}, true, nil)

				mockConvRepo.EXPECT().
					UpdateConversation(mock.Anything, mock.MatchedBy(func(c domain.Conversation) bool {
						return c.Title == newTitle &&
							c.TitleSource == domain.ConversationTitleSource_User &&
							c.UpdatedAt.Equal(fixedTime)
					})).
					Return(nil)
				timeProvider.EXPECT().Now().Return(fixedTime).Once()

				uow.EXPECT().
					Conversation().
					Return(mockConvRepo)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})
			},
			expectedConv: expectedConversation,
			expectedErr:  nil,
		},
		"error-conversation-not-found": {
			conversationID: fixedUUID,
			title:          newTitle,
			setExpectations: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider) {
				mockConvRepo := domain.NewMockConversationRepository(t)
				mockConvRepo.EXPECT().
					GetConversation(mock.Anything, fixedUUID).
					Return(domain.Conversation{}, false, nil)

				uow.EXPECT().
					Conversation().
					Return(mockConvRepo)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})
			},
			expectedConv: domain.Conversation{},
			expectedErr:  domain.NewNotFoundErr("conversation with ID 123e4567-e89b-12d3-a456-426614174000 not found"),
		},
		"error-get-conversation-failure": {
			conversationID: fixedUUID,
			title:          newTitle,
			setExpectations: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider) {
				mockConvRepo := domain.NewMockConversationRepository(t)
				mockConvRepo.EXPECT().
					GetConversation(mock.Anything, fixedUUID).
					Return(domain.Conversation{}, false, errors.New("database error"))

				uow.EXPECT().
					Conversation().
					Return(mockConvRepo)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})
			},
			expectedConv: domain.Conversation{},
			expectedErr:  errors.New("database error"),
		},
		"error-validation-empty-title": {
			conversationID: fixedUUID,
			title:          "",
			setExpectations: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider) {
				mockConvRepo := domain.NewMockConversationRepository(t)
				mockConvRepo.EXPECT().
					GetConversation(mock.Anything, fixedUUID).
					Return(domain.Conversation{
						ID:          fixedUUID,
						Title:       "Old Title",
						TitleSource: domain.ConversationTitleSource_User,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					}, true, nil)

				uow.EXPECT().
					Conversation().
					Return(mockConvRepo)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})
			},
			expectedConv: domain.Conversation{},
			expectedErr:  domain.NewValidationErr("conversation title cannot be empty"),
		},
		"error-update-conversation-failure": {
			conversationID: fixedUUID,
			title:          newTitle,
			setExpectations: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider) {
				mockConvRepo := domain.NewMockConversationRepository(t)
				mockConvRepo.EXPECT().
					GetConversation(mock.Anything, fixedUUID).
					Return(domain.Conversation{
						ID:          fixedUUID,
						Title:       "Old Title",
						TitleSource: domain.ConversationTitleSource_User,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					}, true, nil)

				mockConvRepo.EXPECT().
					UpdateConversation(mock.Anything, mock.Anything).
					Return(errors.New("update failed"))
				timeProvider.EXPECT().Now().Return(fixedTime).Once()

				uow.EXPECT().
					Conversation().
					Return(mockConvRepo)

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(_ domain.UnitOfWork) error) error {
						return fn(uow)
					})
			},
			expectedConv: domain.Conversation{},
			expectedErr:  errors.New("update failed"),
		},
		"error-uow-execute-failure": {
			conversationID: fixedUUID,
			title:          newTitle,
			setExpectations: func(uow *domain.MockUnitOfWork, timeProvider *domain.MockCurrentTimeProvider) {
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					Return(errors.New("transaction failed"))
			},
			expectedConv: domain.Conversation{},
			expectedErr:  errors.New("transaction failed"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain.NewMockUnitOfWork(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			if tt.setExpectations != nil {
				tt.setExpectations(uow, timeProvider)
			}

			uc := NewUpdateConversationImpl(uow, timeProvider)

			got, gotErr := uc.Execute(context.Background(), tt.conversationID, tt.title)
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

func TestInitUpdateConversation_Initialize(t *testing.T) {
	iuc := InitUpdateConversation{}

	ctx, err := iuc.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredUpdateConversation, err := depend.Resolve[UpdateConversation]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredUpdateConversation)
}
