package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	domain_mocks "github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRelayOutboxImpl_Execute(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	eventID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	todoID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174000")

	tests := map[string]struct {
		setExpectations func(uow *domain_mocks.MockUnitOfWork, publisher *domain_mocks.MockTodoEventPublisher)
		expectedErr     error
	}{
		"success-relay-and-delete": {
			setExpectations: func(uow *domain_mocks.MockUnitOfWork, publisher *domain_mocks.MockTodoEventPublisher) {
				outbox := domain_mocks.NewMockOutboxRepository(t)

				uow.EXPECT().Outbox().Return(outbox)
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
						return fn(uow)
					})

				oe := domain.OutboxEvent{
					ID:         eventID,
					EventType:  string(domain.TodoEventType_TODO_CREATED),
					EntityID:   todoID,
					CreatedAt:  fixedTime,
					RetryCount: 0,
					MaxRetries: 3,
				}

				outbox.EXPECT().FetchPendingEvents(
					mock.Anything,
					100,
				).Return([]domain.OutboxEvent{oe}, nil)

				publisher.EXPECT().PublishEvent(
					mock.Anything,
					oe,
				).Return(nil)

				outbox.EXPECT().DeleteEvent(
					mock.Anything,
					eventID,
				).Return(nil)
			},
			expectedErr: nil,
		},
		"success-relay-multiple-events": {
			setExpectations: func(uow *domain_mocks.MockUnitOfWork, publisher *domain_mocks.MockTodoEventPublisher) {
				outbox := domain_mocks.NewMockOutboxRepository(t)

				uow.EXPECT().Outbox().Return(outbox)
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
						return fn(uow)
					})

				eventID2 := uuid.MustParse("323e4567-e89b-12d3-a456-426614174000")
				todoID2 := uuid.MustParse("423e4567-e89b-12d3-a456-426614174000")

				events := []domain.OutboxEvent{
					{
						ID:         eventID,
						EventType:  string(domain.TodoEventType_TODO_CREATED),
						EntityID:   todoID,
						CreatedAt:  fixedTime,
						RetryCount: 0,
						MaxRetries: 3,
					},
					{
						ID:         eventID2,
						EventType:  string(domain.TodoEventType_TODO_UPDATED),
						EntityID:   todoID2,
						CreatedAt:  fixedTime,
						RetryCount: 0,
						MaxRetries: 3,
					},
				}

				outbox.EXPECT().FetchPendingEvents(
					mock.Anything,
					100,
				).Return(events, nil)

				for _, event := range events {
					publisher.EXPECT().PublishEvent(
						mock.Anything,
						event,
					).Return(nil)

					outbox.EXPECT().DeleteEvent(
						mock.Anything,
						event.ID,
					).Return(nil)
				}
			},
			expectedErr: nil,
		},
		"publish-error-retry": {
			setExpectations: func(uow *domain_mocks.MockUnitOfWork, publisher *domain_mocks.MockTodoEventPublisher) {
				outbox := domain_mocks.NewMockOutboxRepository(t)

				uow.EXPECT().Outbox().Return(outbox)
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
						return fn(uow)
					})

				outbox.EXPECT().FetchPendingEvents(
					mock.Anything,
					100,
				).Return([]domain.OutboxEvent{
					{
						ID:         eventID,
						EventType:  string(domain.TodoEventType_TODO_CREATED),
						EntityID:   todoID,
						CreatedAt:  fixedTime,
						RetryCount: 0,
						MaxRetries: 3,
					},
				}, nil)

				publisher.EXPECT().PublishEvent(
					mock.Anything,
					mock.Anything,
				).Return(errors.New("publish error"))

				outbox.EXPECT().UpdateEvent(
					mock.Anything,
					eventID,
					"PENDING",
					1,
					"publish error",
				).Return(nil)
			},
			expectedErr: nil,
		},
		"publish-error-max-retries-exceeded": {
			setExpectations: func(uow *domain_mocks.MockUnitOfWork, publisher *domain_mocks.MockTodoEventPublisher) {
				outbox := domain_mocks.NewMockOutboxRepository(t)

				uow.EXPECT().Outbox().Return(outbox)
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
						return fn(uow)
					})

				outbox.EXPECT().FetchPendingEvents(
					mock.Anything,
					100,
				).Return([]domain.OutboxEvent{
					{
						ID:         eventID,
						EventType:  string(domain.TodoEventType_TODO_CREATED),
						EntityID:   todoID,
						CreatedAt:  fixedTime,
						RetryCount: 2,
						MaxRetries: 3,
					},
				}, nil)

				publisher.EXPECT().PublishEvent(
					mock.Anything,
					mock.Anything,
				).Return(errors.New("publish error"))

				outbox.EXPECT().UpdateEvent(
					mock.Anything,
					eventID,
					"FAILED",
					3,
					"publish error",
				).Return(nil)
			},
			expectedErr: nil,
		},
		"fetch-pending-events-error": {
			setExpectations: func(uow *domain_mocks.MockUnitOfWork, publisher *domain_mocks.MockTodoEventPublisher) {
				outbox := domain_mocks.NewMockOutboxRepository(t)

				uow.EXPECT().Outbox().Return(outbox)
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
						return fn(uow)
					})

				outbox.EXPECT().FetchPendingEvents(
					mock.Anything,
					100,
				).Return(nil, errors.New("database error"))
			},
			expectedErr: errors.New("database error"),
		},
		"empty-batch": {
			setExpectations: func(uow *domain_mocks.MockUnitOfWork, publisher *domain_mocks.MockTodoEventPublisher) {
				outbox := domain_mocks.NewMockOutboxRepository(t)

				uow.EXPECT().Outbox().Return(outbox)
				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
						return fn(uow)
					})

				outbox.EXPECT().FetchPendingEvents(
					mock.Anything,
					100,
				).Return([]domain.OutboxEvent{}, nil)
			},
			expectedErr: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := domain_mocks.NewMockUnitOfWork(t)
			publisher := domain_mocks.NewMockTodoEventPublisher(t)

			if tt.setExpectations != nil {
				tt.setExpectations(uow, publisher)
			}

			relay := NewRelayOutboxImpl(uow, publisher, nil)
			gotErr := relay.Execute(context.Background())

			assert.Equal(t, tt.expectedErr, gotErr)
		})
	}
}

func TestInitRelayOutbox_Initialize(t *testing.T) {
	iro := InitRelayOutbox{}

	ctx, err := iro.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredRelay, err := depend.Resolve[RelayOutbox]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredRelay)
}
