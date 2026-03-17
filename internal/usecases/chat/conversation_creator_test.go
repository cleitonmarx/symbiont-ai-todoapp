package chat

import (
	"context"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
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
			Content:         "Sorry, I could not process your request. Please try again.",
			MessageState:    assistant.ChatMessageState_Failed,
			ErrorMessage:    &expectedErr,
			ActionCallsLen:  0,
			HasActionCallID: false,
		},
	})

	creator := NewConversationCreatorImpl(uow, nil)
	state := NewTurnState(conversation, false, nil, assistant.TurnRequest{Model: "test-model"}, 7)

	userMessage := assistant.ChatMessage{
		ID:             uuid.New(),
		ConversationID: conversation.ID,
		TurnID:         state.TurnID(),
		TurnSequence:   state.NextTurnSequence(),
		ChatRole:       assistant.ChatRole_User,
		Content:        "Hello",
		Model:          "test-model",
		MessageState:   assistant.ChatMessageState_Completed,
		CreatedAt:      fixedTime,
		UpdatedAt:      fixedTime,
	}
	if err := creator.CreateMessage(t.Context(), conversation, userMessage); err != nil {
		t.Fatalf("CreateMessage for user returned error: %v", err)
	}

	failureMessage := assistant.ChatMessage{
		ID:             uuid.New(),
		ConversationID: conversation.ID,
		TurnID:         state.TurnID(),
		TurnSequence:   state.NextTurnSequence(),
		ChatRole:       assistant.ChatRole_Assistant,
		Content:        "Sorry, I could not process your request. Please try again.",
		Model:          state.Model(),
		MessageState:   assistant.ChatMessageState_Failed,
		ErrorMessage:   &expectedErr,
		CreatedAt:      fixedTime,
		UpdatedAt:      fixedTime,
	}
	if err := creator.CreateMessage(t.Context(), conversation, failureMessage); err != nil {
		t.Fatalf("CreateMessage for failure returned error: %v", err)
	}
}

func TestConversationCreator_RepairTurn(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	turnID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	userMessageID := uuid.MustParse("00000000-0000-0000-0000-000000000011")
	danglingAssistantMessageID := uuid.MustParse("00000000-0000-0000-0000-000000000012")
	otherTurnMessageID := uuid.MustParse("00000000-0000-0000-0000-000000000013")
	actionCallID := "func-123"
	userCreatedAt := time.Date(2026, 3, 14, 10, 0, 0, 0, time.UTC)
	danglingCreatedAt := userCreatedAt.Add(5 * time.Second)
	otherCreatedAt := danglingCreatedAt.Add(5 * time.Second)

	chatRepo := assistant.NewMockChatMessageRepository(t)
	conversationRepo := assistant.NewMockConversationRepository(t)
	uow := transaction.NewMockUnitOfWork(t)
	scope := transaction.NewMockScope(t)

	uow.EXPECT().
		Execute(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
			return fn(ctx, scope)
		}).
		Once()

	scope.EXPECT().ChatMessage().Return(chatRepo).Twice()
	scope.EXPECT().Conversation().Return(conversationRepo).Twice()

	chatRepo.EXPECT().
		ListChatMessages(mock.Anything, conversationID, 1, 0).
		Return([]assistant.ChatMessage{
			{
				ID:             userMessageID,
				ConversationID: conversationID,
				TurnID:         turnID,
				ChatRole:       assistant.ChatRole_User,
				Content:        "hello",
				CreatedAt:      userCreatedAt,
			},
			{
				ID:             danglingAssistantMessageID,
				ConversationID: conversationID,
				TurnID:         turnID,
				ChatRole:       assistant.ChatRole_Assistant,
				ActionCalls: []assistant.ActionCall{
					{ID: actionCallID, Name: "list_todos"},
				},
				CreatedAt: danglingCreatedAt,
			},
			{
				ID:             otherTurnMessageID,
				ConversationID: conversationID,
				TurnID:         uuid.MustParse("00000000-0000-0000-0000-000000000020"),
				ChatRole:       assistant.ChatRole_Assistant,
				Content:        "previous turn",
				CreatedAt:      otherCreatedAt,
			},
		}, false, nil).
		Once()

	chatRepo.EXPECT().
		DeleteChatMessages(mock.Anything, []uuid.UUID{danglingAssistantMessageID}).
		Return(nil).
		Once()

	conversationRepo.EXPECT().
		GetConversation(mock.Anything, conversationID).
		Return(assistant.Conversation{
			ID:          conversationID,
			Title:       "Fresh title",
			TitleSource: assistant.ConversationTitleSource_LLM,
			CreatedAt:   userCreatedAt.Add(-time.Minute),
			UpdatedAt:   danglingCreatedAt,
		}, true, nil).
		Once()

	conversationRepo.EXPECT().
		UpdateConversation(mock.Anything, mock.MatchedBy(func(conv assistant.Conversation) bool {
			return conv.ID == conversationID &&
				conv.Title == "Fresh title" &&
				conv.TitleSource == assistant.ConversationTitleSource_LLM &&
				conv.LastMessageAt != nil &&
				conv.LastMessageAt.Equal(otherCreatedAt) &&
				conv.UpdatedAt.Equal(otherCreatedAt)
		})).
		Return(nil).
		Once()

	creator := NewConversationCreatorImpl(uow, nil)
	err := creator.RepairTurn(t.Context(), assistant.Conversation{
		ID:        conversationID,
		CreatedAt: userCreatedAt.Add(-time.Minute),
		UpdatedAt: danglingCreatedAt,
	}, turnID)
	if err != nil {
		t.Fatalf("RepairTurn returned error: %v", err)
	}
}
