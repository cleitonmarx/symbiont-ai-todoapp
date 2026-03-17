package chat

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTurnRunner_Run_RetriesAfterRecovery(t *testing.T) {
	t.Parallel()

	assistantClient := assistant.NewMockAssistant(t)
	actionPipeline := NewMockActionPipeline(t)
	callCount := 0

	assistantClient.EXPECT().
		RunTurn(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, req assistant.TurnRequest, _ assistant.EventCallback) error {
			callCount++
			if callCount == 1 {
				return errors.New("stream failed")
			}
			assert.NotEmpty(t, req.Messages)
			assert.Equal(t, assistant.ChatRole_System, req.Messages[len(req.Messages)-1].Role)
			assert.True(t, strings.Contains(req.Messages[len(req.Messages)-1].Content, "stream failed"))
			return nil
		}).
		Twice()

	runner := NewTurnRunnerImpl(
		log.New(io.Discard, "", 0),
		assistantClient,
		actionPipeline,
	)

	state := NewTurnState(assistant.Conversation{}, false, nil, assistant.TurnRequest{
		Model:    "test-model",
		Messages: []assistant.Message{{Role: assistant.ChatRole_User, Content: "Hello"}},
	}, 7)

	err := runner.Run(t.Context(), state, func(context.Context, assistant.EventType, any) error { return nil })
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestTurnRunner_Run_ProcessesStreamEvents(t *testing.T) {
	t.Parallel()

	assistantClient := assistant.NewMockAssistant(t)
	actionPipeline := NewMockActionPipeline(t)
	runner := NewTurnRunnerImpl(
		log.New(io.Discard, "", 0),
		assistantClient,
		actionPipeline,
	)

	state := NewTurnState(
		assistant.Conversation{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
		true,
		nil,
		assistant.TurnRequest{Model: "test-model"},
		7,
	)

	actionPipeline.EXPECT().
		Handle(mock.Anything, assistant.ActionCall{ID: "call-1", Name: "list_todos"}, state, mock.Anything).
		Return(false, nil).
		Once()
	request := state.Request()
	assistantClient.EXPECT().
		RunTurn(mock.Anything, request, mock.Anything).
		RunAndReturn(func(ctx context.Context, _ assistant.TurnRequest, onEvent assistant.EventCallback) error {
			if err := onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "Hello back"}); err != nil {
				return err
			}
			if err := onEvent(ctx, assistant.EventType_ActionRequested, assistant.ActionCall{ID: "call-1", Name: "list_todos"}); err != nil {
				return err
			}
			return onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
				Usage: assistant.Usage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8},
			})
		}).
		Once()

	var turnStarted assistant.TurnStarted
	var eventTypes []assistant.EventType
	err := runner.Run(t.Context(), state, func(_ context.Context, eventType assistant.EventType, data any) error {
		eventTypes = append(eventTypes, eventType)
		if eventType == assistant.EventType_TurnStarted {
			turnStarted = data.(assistant.TurnStarted)
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, state.TurnID(), turnStarted.TurnID)
	assert.Equal(t, "Hello back", state.AssistantContent())
	assert.Equal(t, 3, state.TokenUsage().PromptTokens)
	assert.Equal(t, 5, state.TokenUsage().CompletionTokens)
	assert.Equal(t, 8, state.TokenUsage().TotalTokens)
	assert.Equal(t, []assistant.EventType{
		assistant.EventType_TurnStarted,
		assistant.EventType_MessageDelta,
	}, eventTypes)
}
