package approvaldispatcher

import (
	"context"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatcher_WaitAndDispatch(t *testing.T) {
	t.Parallel()

	dispatcher := NewDispatcher()
	key := domain.AssistantActionApprovalKey{
		ConversationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		TurnID:         uuid.MustParse("10000000-0000-0000-0000-000000000001"),
		ActionCallID:   "call-1",
	}
	expected := domain.AssistantActionApprovalDecision{
		Key:        key,
		ActionName: "delete_todos",
		Status:     domain.ChatMessageApprovalStatus_Approved,
		Reason:     common.Ptr("approved"),
		DecidedAt:  time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC),
	}

	waitResult := make(chan domain.AssistantActionApprovalDecision, 1)
	waitErr := make(chan error, 1)

	go func() {
		decision, err := dispatcher.Wait(t.Context(), key)
		waitResult <- decision
		waitErr <- err
	}()

	time.Sleep(10 * time.Millisecond)
	dispatched := dispatcher.Dispatch(t.Context(), expected)
	assert.True(t, dispatched)

	gotDecision := <-waitResult
	gotErr := <-waitErr
	require.NoError(t, gotErr)
	assert.Equal(t, expected, gotDecision)
}

func TestDispatcher_WaitCanceled(t *testing.T) {
	t.Parallel()

	dispatcher := NewDispatcher()
	key := domain.AssistantActionApprovalKey{
		ConversationID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		TurnID:         uuid.MustParse("20000000-0000-0000-0000-000000000002"),
		ActionCallID:   "call-2",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := dispatcher.Wait(ctx, key)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDispatcher_DispatchWithoutWaiter(t *testing.T) {
	t.Parallel()

	dispatcher := NewDispatcher()
	dispatched := dispatcher.Dispatch(t.Context(), domain.AssistantActionApprovalDecision{
		Key: domain.AssistantActionApprovalKey{
			ConversationID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			TurnID:         uuid.MustParse("30000000-0000-0000-0000-000000000003"),
			ActionCallID:   "call-3",
		},
		ActionName: "delete_todos",
		Status:     domain.ChatMessageApprovalStatus_Rejected,
		DecidedAt:  time.Now().UTC(),
	})

	assert.False(t, dispatched)
}
