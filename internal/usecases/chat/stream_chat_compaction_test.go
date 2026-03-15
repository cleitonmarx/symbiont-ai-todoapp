package chat

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStreamChatImpl_compactConversationContext(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	fixedTime := time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC)
	compactionPolicy := assistant.ContextCompactionPolicy{
		TriggerTokenCount: 8000,
	}

	tests := map[string]struct {
		setExpectations func(*MockConversationCompactor)
		wantEvents     []assistant.EventType
		wantTimeCalled bool
		timeout        time.Duration
	}{
		"skips-when-not-triggered": {
			setExpectations: func(compactor *MockConversationCompactor) {
				compactor.EXPECT().EvaluateConversationCompaction(mock.Anything, conversationID, compactionPolicy).Return(assistant.ContextCompactionDecision{
					ShouldGenerate: false,
					Reason:         assistant.ContextCompactionReasonNone,
				}, nil).Once()
			},
			wantEvents:     []assistant.EventType{},
			wantTimeCalled: false,
		},
		"emits-started-and-completed-when-triggered": {
			setExpectations: func(compactor *MockConversationCompactor) {
				compactor.EXPECT().EvaluateConversationCompaction(mock.Anything, conversationID, compactionPolicy).Return(assistant.ContextCompactionDecision{
					ShouldGenerate: true,
					Reason:         assistant.ContextCompactionReasonTokenCountThreshold,
					MessageCount:   7,
					TotalTokens:    1200,
				}, nil).Once()
				compactor.EXPECT().CompactConversation(mock.Anything, conversationID).Return(nil).Once()
			},
			wantEvents:     []assistant.EventType{assistant.EventType_ContextCompactionStarted, assistant.EventType_ContextCompactionCompleted},
			wantTimeCalled: true,
		},
		"emits-failed-when-evaluation-errors": {
			setExpectations: func(compactor *MockConversationCompactor) {
				compactor.EXPECT().EvaluateConversationCompaction(mock.Anything, conversationID, compactionPolicy).Return(assistant.ContextCompactionDecision{}, assert.AnError).Once()
			},
			wantEvents:     []assistant.EventType{assistant.EventType_ContextCompactionFailed},
			wantTimeCalled: false,
		},
		"emits-started-then-failed-when-compaction-errors": {
			setExpectations: func(compactor *MockConversationCompactor) {
				compactor.EXPECT().EvaluateConversationCompaction(mock.Anything, conversationID, compactionPolicy).Return(assistant.ContextCompactionDecision{
					ShouldGenerate: true,
					Reason:         assistant.ContextCompactionReasonTokenCountThreshold,
					MessageCount:   2,
					TotalTokens:    990,
				}, nil).Once()
				compactor.EXPECT().CompactConversation(mock.Anything, conversationID).Return(assert.AnError).Once()
			},
			wantEvents:     []assistant.EventType{assistant.EventType_ContextCompactionStarted, assistant.EventType_ContextCompactionFailed},
			wantTimeCalled: false,
		},
		"emits-started-then-failed-when-compaction-times-out": {
			setExpectations: func(compactor *MockConversationCompactor) {
				compactor.EXPECT().EvaluateConversationCompaction(mock.Anything, conversationID, compactionPolicy).Return(assistant.ContextCompactionDecision{
					ShouldGenerate: true,
					Reason:         assistant.ContextCompactionReasonTokenCountThreshold,
					MessageCount:   5,
					TotalTokens:    980,
				}, nil).Once()
				compactor.EXPECT().CompactConversation(mock.Anything, conversationID).RunAndReturn(func(ctx context.Context, _ uuid.UUID) error {
					<-ctx.Done()
					return ctx.Err()
				}).Once()
			},
			wantEvents:     []assistant.EventType{assistant.EventType_ContextCompactionStarted, assistant.EventType_ContextCompactionFailed},
			wantTimeCalled: false,
			timeout:        10 * time.Millisecond,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			compactor := NewMockConversationCompactor(t)
			tt.setExpectations(compactor)

			timeProvider := core.NewMockCurrentTimeProvider(t)
			if tt.wantTimeCalled {
				timeProvider.EXPECT().Now().Return(fixedTime).Once()
			}

			useCase := StreamChatImpl{
				logger:                log.New(io.Discard, "", 0),
				conversationCompactor: compactor,
				timeProvider:          timeProvider,
				compactionPolicy:      compactionPolicy,
				compactionTimeout: tt.timeout,
			}

			gotEvents := make([]assistant.EventType, 0, 2)
			err := useCase.compactConversationContext(t.Context(), conversationID, func(_ context.Context, eventType assistant.EventType, _ any) error {
				gotEvents = append(gotEvents, eventType)
				return nil
			})

			assert.NoError(t, err)
			assert.Equal(t, tt.wantEvents, gotEvents)
		})
	}
}
