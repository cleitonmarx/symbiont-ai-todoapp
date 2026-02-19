package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGenerateConversationTitleImpl_Execute(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	chatMessageID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		event           domain.ChatMessageEvent
		model           string
		setExpectations func(
			*domain.MockConversationRepository,
			*domain.MockConversationSummaryRepository,
			*domain.MockChatMessageRepository,
			*domain.MockCurrentTimeProvider,
			*domain.MockAssistant,
		)
		expectedErr error
	}{
		"invalid-event-type": {
			model: "title-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_TODO_CREATED,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
			},
			expectedErr: domain.NewValidationErr("invalid event type for conversation title generation"),
		},
		"empty-conversation-id": {
			model: "title-model",
			event: domain.ChatMessageEvent{
				Type:          domain.EventType_CHAT_MESSAGE_SENT,
				ChatMessageID: chatMessageID,
				ChatRole:      domain.ChatRole_Assistant,
			},
			expectedErr: domain.NewValidationErr("conversation id cannot be empty"),
		},
		"ignore-non-assistant-events": {
			model: "title-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
				ChatRole:       domain.ChatRole_User,
			},
			expectedErr: nil,
		},
		"conversation-not-found-noop": {
			model: "title-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
				ChatRole:       domain.ChatRole_Assistant,
			},
			setExpectations: func(
				conversationRepo *domain.MockConversationRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{}, false, nil).
					Once()
			},
			expectedErr: nil,
		},
		"skip-non-auto-title-source": {
			model: "title-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
				ChatRole:       domain.ChatRole_Assistant,
			},
			setExpectations: func(
				conversationRepo *domain.MockConversationRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID:          conversationID,
						Title:       "Manually renamed",
						TitleSource: domain.ConversationTitleSource_User,
					}, true, nil).
					Once()
			},
			expectedErr: nil,
		},
		"llm-error": {
			model: "title-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
				ChatRole:       domain.ChatRole_Assistant,
			},
			setExpectations: func(
				conversationRepo *domain.MockConversationRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID:          conversationID,
						Title:       "Show my tasks",
						TitleSource: domain.ConversationTitleSource_Auto,
					}, true, nil).
					Once()

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_MESSAGES_FOR_TITLE).
					Return([]domain.ChatMessage{
						{ChatRole: domain.ChatRole_User, Content: "Plan my week", MessageState: domain.ChatMessageState_Completed},
						{ChatRole: domain.ChatRole_Assistant, Content: "Sure, let's plan your week.", MessageState: domain.ChatMessageState_Completed},
					}, false, nil).
					Once()

				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{
						ConversationID:      conversationID,
						CurrentStateSummary: "User planning weekly tasks and timeline",
					}, true, nil).
					Once()

				assistant.EXPECT().
					RunTurnSync(mock.Anything, mock.Anything).
					Return(domain.AssistantTurnResponse{}, errors.New("llm unavailable")).
					Once()
			},
			expectedErr: errors.New("failed to generate conversation title: llm unavailable"),
		},
		"success-update-title": {
			model: "title-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
				ChatRole:       domain.ChatRole_Assistant,
			},
			setExpectations: func(
				conversationRepo *domain.MockConversationRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID:          conversationID,
						Title:       "Show my tasks",
						TitleSource: domain.ConversationTitleSource_Auto,
					}, true, nil).
					Once()

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_MESSAGES_FOR_TITLE).
					Return([]domain.ChatMessage{
						{ChatRole: domain.ChatRole_User, Content: "Break down spring cleaning", MessageState: domain.ChatMessageState_Completed},
						{ChatRole: domain.ChatRole_Assistant, Content: "I split this into room-based tasks.", MessageState: domain.ChatMessageState_Completed},
					}, false, nil).
					Once()

				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{
						ConversationID:      conversationID,
						CurrentStateSummary: "Spring cleaning plan with room-based todo breakdown and due-date schedule",
					}, true, nil).
					Once()

				assistant.EXPECT().
					RunTurnSync(mock.Anything, mock.MatchedBy(func(req domain.AssistantTurnRequest) bool {
						require.GreaterOrEqual(t, len(req.Messages), 1)
						assert.Equal(t, domain.ChatRole_System, req.Messages[0].Role)
						assert.Contains(t, req.Messages[0].Content, "Current title:")
						assert.Contains(t, req.Messages[0].Content, "Focused summary:")
						assert.Contains(t, req.Messages[0].Content, "Recent conversation context:")
						assert.Contains(t, req.Messages[0].Content, "Spring cleaning plan with room-based todo breakdown")
						assert.NotContains(t, req.Messages[0].Content, "**")
						return req.Model == "title-model" &&
							req.Stream == false &&
							assert.Equal(t, common.Ptr(CHAT_TITLE_MAX_TOKENS), req.MaxTokens) &&
							assert.Equal(t, common.Ptr(CHAT_TITLE_TEMPERATURE), req.Temperature) &&
							assert.Equal(t, common.Ptr(CHAT_TITLE_TOP_P), req.TopP)
					})).
					Return(domain.AssistantTurnResponse{
						Content: "\"Spring Cleaning Task Breakdown\"",
						Usage: domain.AssistantUsage{
							PromptTokens:     10,
							CompletionTokens: 4,
							TotalTokens:      14,
						},
					}, nil).
					Once()

				timeProvider.EXPECT().Now().Return(fixedTime).Once()

				conversationRepo.EXPECT().
					UpdateConversation(mock.Anything, mock.MatchedBy(func(c domain.Conversation) bool {
						return c.ID == conversationID &&
							c.Title == "Spring Cleaning Task Breakdown" &&
							c.TitleSource == domain.ConversationTitleSource_LLM &&
							c.UpdatedAt.Equal(fixedTime)
					})).
					Return(nil).
					Once()
			},
			expectedErr: nil,
		},
		"off-topic-title-noop": {
			model: "title-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
				ChatRole:       domain.ChatRole_Assistant,
			},
			setExpectations: func(
				conversationRepo *domain.MockConversationRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID:          conversationID,
						Title:       "I invited my Friend to...",
						TitleSource: domain.ConversationTitleSource_Auto,
					}, true, nil).
					Once()

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_MESSAGES_FOR_TITLE).
					Return([]domain.ChatMessage{
						{ChatRole: domain.ChatRole_User, Content: "I invited my friend to a dinner and need hosting tasks.", MessageState: domain.ChatMessageState_Completed},
						{ChatRole: domain.ChatRole_Assistant, Content: "I created tasks for your dinner plan.", MessageState: domain.ChatMessageState_Completed},
					}, false, nil).
					Once()

				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{
						ConversationID: conversationID,
						CurrentStateSummary: "current_intent: Create tasks for hosting a dinner with Friend on Feb 20, ensuring all needs are met\n" +
							"user_nuances: Use \"Dinner with Friend:\" as title prefix, focus on food and hosting\n" +
							"tasks: Dinner with Friend: Research restaurants near me|O|2026-02-15; Dinner with Friend: Confirm menu options with restaurant|O|2026-02-16\n" +
							"last_action: create_task -> Success -> Task ID 23d8eed5-1afd-4ecb-a9f6-b2c4b8d5f8a1 created\n" +
							"output_format: list",
					}, true, nil).
					Once()

				assistant.EXPECT().
					RunTurnSync(mock.Anything, mock.Anything).
					Return(domain.AssistantTurnResponse{
						Content: "Prepare weekly meeting agenda and send to team",
					}, nil).
					Once()
			},
			expectedErr: nil,
		},
		"empty-or-placeholder-generated-title-noop": {
			model: "title-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
				ChatRole:       domain.ChatRole_Assistant,
			},
			setExpectations: func(
				conversationRepo *domain.MockConversationRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID:          conversationID,
						Title:       "Show my tasks",
						TitleSource: domain.ConversationTitleSource_Auto,
					}, true, nil).
					Once()

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_MESSAGES_FOR_TITLE).
					Return([]domain.ChatMessage{
						{ChatRole: domain.ChatRole_User, Content: "What should I do first?", MessageState: domain.ChatMessageState_Completed},
						{ChatRole: domain.ChatRole_Assistant, Content: "Let's prioritize by due date.", MessageState: domain.ChatMessageState_Completed},
					}, false, nil).
					Once()

				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{
						ConversationID:      conversationID,
						CurrentStateSummary: "User asked for prioritization by due date",
					}, true, nil).
					Once()

				assistant.EXPECT().
					RunTurnSync(mock.Anything, mock.Anything).
					Return(domain.AssistantTurnResponse{Content: "New Conversation"}, nil).
					Once()
			},
			expectedErr: nil,
		},
		"sanitizes-single-line-verbose-title": {
			model: "title-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
				ChatRole:       domain.ChatRole_Assistant,
			},
			setExpectations: func(
				conversationRepo *domain.MockConversationRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID:          conversationID,
						Title:       "I want to plan a...",
						TitleSource: domain.ConversationTitleSource_Auto,
					}, true, nil).
					Once()

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_MESSAGES_FOR_TITLE).
					Return([]domain.ChatMessage{
						{ChatRole: domain.ChatRole_User, Content: "I want to plan a trip to Japan on April 4.", MessageState: domain.ChatMessageState_Completed},
						{ChatRole: domain.ChatRole_Assistant, Content: "I've created tasks for your Japan trip.", MessageState: domain.ChatMessageState_Completed},
					}, false, nil).
					Once()

				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{
						ConversationID:      conversationID,
						CurrentStateSummary: "Planning Japan trip timeline with prep tasks before April 4",
					}, true, nil).
					Once()

				assistant.EXPECT().
					RunTurnSync(mock.Anything, mock.Anything).
					Return(domain.AssistantTurnResponse{
						Content: "Japan Trip planning with research flights accommodation visa checklist and packing",
					}, nil).
					Once()

				timeProvider.EXPECT().Now().Return(fixedTime).Once()

				conversationRepo.EXPECT().
					UpdateConversation(mock.Anything, mock.MatchedBy(func(c domain.Conversation) bool {
						return c.ID == conversationID &&
							c.Title == "Japan Trip planning with research flights accommodation" &&
							c.TitleSource == domain.ConversationTitleSource_LLM
					})).
					Return(nil).
					Once()
			},
			expectedErr: nil,
		},
		"rejects-multiline-list-title-output": {
			model: "title-model",
			event: domain.ChatMessageEvent{
				Type:           domain.EventType_CHAT_MESSAGE_SENT,
				ConversationID: conversationID,
				ChatMessageID:  chatMessageID,
				ChatRole:       domain.ChatRole_Assistant,
			},
			setExpectations: func(
				conversationRepo *domain.MockConversationRepository,
				summaryRepo *domain.MockConversationSummaryRepository,
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				assistant *domain.MockAssistant,
			) {
				conversationRepo.EXPECT().
					GetConversation(mock.Anything, conversationID).
					Return(domain.Conversation{
						ID:          conversationID,
						Title:       "I want to plan a...",
						TitleSource: domain.ConversationTitleSource_Auto,
					}, true, nil).
					Once()

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_MESSAGES_FOR_TITLE).
					Return([]domain.ChatMessage{
						{ChatRole: domain.ChatRole_User, Content: "I want to plan a trip to Japan on April 4.", MessageState: domain.ChatMessageState_Completed},
						{ChatRole: domain.ChatRole_Assistant, Content: "I've created tasks for your Japan trip.", MessageState: domain.ChatMessageState_Completed},
					}, false, nil).
					Once()

				summaryRepo.EXPECT().
					GetConversationSummary(mock.Anything, conversationID).
					Return(domain.ConversationSummary{
						ConversationID:      conversationID,
						CurrentStateSummary: "Planning Japan trip timeline with prep tasks before April 4",
					}, true, nil).
					Once()

				assistant.EXPECT().
					RunTurnSync(mock.Anything, mock.Anything).
					Return(domain.AssistantTurnResponse{
						Content: "Review project timeline\nUpdate client contact info\nPrepare meeting agenda",
					}, nil).
					Once()
			},
			expectedErr: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			conversationRepo := domain.NewMockConversationRepository(t)
			summaryRepo := domain.NewMockConversationSummaryRepository(t)
			chatRepo := domain.NewMockChatMessageRepository(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			assistant := domain.NewMockAssistant(t)
			if tt.setExpectations != nil {
				tt.setExpectations(conversationRepo, summaryRepo, chatRepo, timeProvider, assistant)
			}

			uc := NewGenerateConversationTitleImpl(
				conversationRepo,
				summaryRepo,
				chatRepo,
				timeProvider,
				assistant,
				tt.model,
				nil,
			)

			err := uc.Execute(context.Background(), tt.event)
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitGenerateConversationTitle_Initialize(t *testing.T) {
	i := InitGenerateConversationTitle{}

	ctx, err := i.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredUseCase, err := depend.Resolve[GenerateConversationTitle]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredUseCase)
}
