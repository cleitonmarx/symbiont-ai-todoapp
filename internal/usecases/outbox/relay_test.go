package outbox

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRelayOutboxImpl_Execute(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	eventID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	todoID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174000")

	tests := map[string]struct {
		setExpectations func(uow *transaction.MockUnitOfWork, publisher *outbox.MockEventPublisher)
		expectedErr     error
	}{
		"success-relay-and-mark-processed": {
			setExpectations: func(uow *transaction.MockUnitOfWork, publisher *outbox.MockEventPublisher) {
				outboxRepo := outbox.NewMockRepository(t)

				scope := transaction.NewMockScope(t)
				scope.EXPECT().Outbox().Return(outboxRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					})

				oe := outbox.Event{
					ID:         eventID,
					EventType:  outbox.EventType_TODO_CREATED,
					EntityID:   todoID,
					CreatedAt:  fixedTime,
					RetryCount: 0,
					MaxRetries: 3,
				}

				outboxRepo.EXPECT().FetchPendingEvents(
					mock.Anything,
					outboxRelayBatchSize,
				).Return([]outbox.Event{oe}, nil).Once()
				outboxRepo.EXPECT().FetchPendingEvents(
					mock.Anything,
					outboxRelayBatchSize,
				).Return([]outbox.Event{}, nil).Once()

				publisher.EXPECT().PublishEvent(
					mock.Anything,
					mock.Anything,
				).Return(nil)

				outboxRepo.EXPECT().UpdateEvent(
					mock.Anything,
					eventID,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(nil)
			},
			expectedErr: nil,
		},
		"success-relay-multiple-events": {
			setExpectations: func(uow *transaction.MockUnitOfWork, publisher *outbox.MockEventPublisher) {
				outboxRepo := outbox.NewMockRepository(t)

				scope := transaction.NewMockScope(t)
				scope.EXPECT().Outbox().Return(outboxRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					})

				eventID2 := uuid.MustParse("323e4567-e89b-12d3-a456-426614174000")
				todoID2 := uuid.MustParse("423e4567-e89b-12d3-a456-426614174000")

				events := []outbox.Event{
					{
						ID:         eventID,
						EventType:  outbox.EventType_TODO_CREATED,
						EntityID:   todoID,
						CreatedAt:  fixedTime,
						RetryCount: 0,
						MaxRetries: 3,
					},
					{
						ID:         eventID2,
						EventType:  outbox.EventType_TODO_UPDATED,
						EntityID:   todoID2,
						CreatedAt:  fixedTime,
						RetryCount: 0,
						MaxRetries: 3,
					},
				}

				outboxRepo.EXPECT().FetchPendingEvents(
					mock.Anything,
					outboxRelayBatchSize,
				).Return(events, nil).Once()
				outboxRepo.EXPECT().FetchPendingEvents(
					mock.Anything,
					outboxRelayBatchSize,
				).Return([]outbox.Event{}, nil).Once()

				for _, event := range events {
					publisher.EXPECT().PublishEvent(
						mock.Anything,
						mock.Anything,
					).Return(nil)

					outboxRepo.EXPECT().UpdateEvent(
						mock.Anything,
						event.ID,
						mock.Anything,
						mock.Anything,
						mock.Anything,
					).Return(nil)
				}
			},
			expectedErr: nil,
		},
		"publish-error-retry": {
			setExpectations: func(uow *transaction.MockUnitOfWork, publisher *outbox.MockEventPublisher) {
				outboxRepo := outbox.NewMockRepository(t)

				scope := transaction.NewMockScope(t)
				scope.EXPECT().Outbox().Return(outboxRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					})

				outboxRepo.EXPECT().FetchPendingEvents(
					mock.Anything,
					outboxRelayBatchSize,
				).Return([]outbox.Event{
					{
						ID:         eventID,
						EventType:  outbox.EventType_TODO_CREATED,
						EntityID:   todoID,
						CreatedAt:  fixedTime,
						RetryCount: 0,
						MaxRetries: 3,
					},
				}, nil).Once()
				outboxRepo.EXPECT().FetchPendingEvents(
					mock.Anything,
					outboxRelayBatchSize,
				).Return([]outbox.Event{}, nil).Once()

				publisher.EXPECT().PublishEvent(
					mock.Anything,
					mock.Anything,
				).Return(errors.New("publish error"))

				outboxRepo.EXPECT().UpdateEvent(
					mock.Anything,
					eventID,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(nil)
			},
			expectedErr: nil,
		},
		"publish-error-max-retries-exceeded": {
			setExpectations: func(uow *transaction.MockUnitOfWork, publisher *outbox.MockEventPublisher) {
				outboxRepo := outbox.NewMockRepository(t)

				scope := transaction.NewMockScope(t)
				scope.EXPECT().Outbox().Return(outboxRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					})

				outboxRepo.EXPECT().FetchPendingEvents(
					mock.Anything,
					outboxRelayBatchSize,
				).Return([]outbox.Event{
					{
						ID:         eventID,
						EventType:  outbox.EventType_TODO_CREATED,
						EntityID:   todoID,
						CreatedAt:  fixedTime,
						RetryCount: 2,
						MaxRetries: 3,
					},
				}, nil).Once()
				outboxRepo.EXPECT().FetchPendingEvents(
					mock.Anything,
					outboxRelayBatchSize,
				).Return([]outbox.Event{}, nil).Once()

				publisher.EXPECT().PublishEvent(
					mock.Anything,
					mock.Anything,
				).Return(errors.New("publish error"))

				outboxRepo.EXPECT().UpdateEvent(
					mock.Anything,
					eventID,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(nil)
			},
			expectedErr: nil,
		},
		"fetch-pending-events-error": {
			setExpectations: func(uow *transaction.MockUnitOfWork, publisher *outbox.MockEventPublisher) {
				outboxRepo := outbox.NewMockRepository(t)

				scope := transaction.NewMockScope(t)
				scope.EXPECT().Outbox().Return(outboxRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					})

				outboxRepo.EXPECT().FetchPendingEvents(
					mock.Anything,
					outboxRelayBatchSize,
				).Return(nil, errors.New("database error")).Once()
			},
			expectedErr: errors.New("database error"),
		},
		"empty-batch": {
			setExpectations: func(uow *transaction.MockUnitOfWork, publisher *outbox.MockEventPublisher) {
				outboxRepo := outbox.NewMockRepository(t)

				scope := transaction.NewMockScope(t)
				scope.EXPECT().Outbox().Return(outboxRepo).Once()

				uow.EXPECT().
					Execute(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
						return fn(ctx, scope)
					})

				outboxRepo.EXPECT().FetchPendingEvents(
					mock.Anything,
					outboxRelayBatchSize,
				).Return([]outbox.Event{}, nil).Once()
			},
			expectedErr: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uow := transaction.NewMockUnitOfWork(t)
			publisher := outbox.NewMockEventPublisher(t)

			if tt.setExpectations != nil {
				tt.setExpectations(uow, publisher)
			}

			relay := NewRelayImpl(uow, publisher, log.New(io.Discard, "", 0))
			gotErr := relay.Execute(t.Context())

			assert.Equal(t, tt.expectedErr, gotErr)
		})
	}
}
