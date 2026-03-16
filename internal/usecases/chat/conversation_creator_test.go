package chat

import (
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
)

func TestConversationCreator_CreateMessage(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 14, 10, 0, 0, 0, time.UTC)
	conversation := assistant.Conversation{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}
	chatRepo := assistant.NewMockChatMessageRepository(t)
	conversationRepo := assistant.NewMockConversationRepository(t)
	uow := transaction.NewMockUnitOfWork(t)
	outboxRepo := outbox.NewMockRepository(t)

	expectedErr := "stream failed"
	expectPersistSequence(t, chatRepo, conversationRepo, uow, outboxRepo, fixedTime, []persistCallExpectation{
		{
			Role:            assistant.ChatRole_User,
			Content:         "Hello",
			ActionCallsLen:  0,
			HasActionCallID: false,
		},
		{
			Role:            assistant.ChatRole_Assistant,
			Content:         "",
			MessageState:    assistant.ChatMessageState_Failed,
			ErrorMessage:    &expectedErr,
			ActionCallsLen:  0,
			HasActionCallID: false,
		},
	})

	creator := newConversationCreator(uow, nil)
	session := NewTurnSession(conversation, false, "Hello", nil, assistant.TurnRequest{Model: "test-model"}, 7)

	userMessage, ok := session.TryBuildUserMessage(uuid.Nil, fixedTime)
	if !ok {
		t.Fatal("TryBuildUserMessage returned false")
	}
	if err := creator.CreateMessage(t.Context(), conversation, userMessage); err != nil {
		t.Fatalf("CreateMessage for user returned error: %v", err)
	}

	failureMessage := session.BuildFailureMessage(fixedTime, errors.New(expectedErr))
	if err := creator.CreateMessage(t.Context(), conversation, failureMessage); err != nil {
		t.Fatalf("CreateMessage for failure returned error: %v", err)
	}
}
