package chat

import (
	"context"
	"log"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitDeleteConversation is the initializer for the DeleteConversation usecase
type InitDeleteConversation struct {
	Uow transaction.UnitOfWork `resolve:""`
}

// Initialize registers the DeleteConversation use case in the dependency container.
func (i InitDeleteConversation) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[DeleteConversation](NewDeleteConversationImpl(i.Uow))
	return ctx, nil
}

// InitGenerateChatSummary is the initializer for the GenerateChatSummary use case
type InitGenerateChatSummary struct {
	ChatMessageRepo         assistant.ChatMessageRepository         `resolve:""`
	ConversationSummaryRepo assistant.ConversationSummaryRepository `resolve:""`
	TimeProvider            core.CurrentTimeProvider                `resolve:""`
	Assistant               assistant.Assistant                     `resolve:""`
	Model                   string                                  `config:"LLM_CHAT_SUMMARY_MODEL"`
}

// Initialize registers the GenerateChatSummary use case in the dependency container.
func (i InitGenerateChatSummary) Initialize(ctx context.Context) (context.Context, error) {
	queue, _ := depend.Resolve[CompletedConversationSummaryChannel]()
	depend.Register[GenerateChatSummary](NewGenerateChatSummaryImpl(
		i.ChatMessageRepo,
		i.ConversationSummaryRepo,
		i.TimeProvider,
		i.Assistant,
		i.Model,
		queue,
	))
	return ctx, nil
}

// InitGenerateConversationTitle is the initializer for the GenerateConversationTitle use case
type InitGenerateConversationTitle struct {
	ConversationRepo        assistant.ConversationRepository        `resolve:""`
	ConversationSummaryRepo assistant.ConversationSummaryRepository `resolve:""`
	ChatMessageRepo         assistant.ChatMessageRepository         `resolve:""`
	TimeProvider            core.CurrentTimeProvider                `resolve:""`
	Assistant               assistant.Assistant                     `resolve:""`
	Model                   string                                  `config:"LLM_CHAT_TITLE_MODEL"`
}

// Initialize registers the GenerateConversationTitle use case in the dependency container.
func (i InitGenerateConversationTitle) Initialize(ctx context.Context) (context.Context, error) {
	queue, _ := depend.Resolve[CompletedConversationTitleUpdateChannel]()
	depend.Register[GenerateConversationTitle](NewGenerateConversationTitleImpl(
		i.ConversationRepo,
		i.ConversationSummaryRepo,
		i.ChatMessageRepo,
		i.TimeProvider,
		i.Assistant,
		i.Model,
		queue,
	))
	return ctx, nil
}

// InitListAvailableModels is the initializer for the ListAvailableModels use case
type InitListAvailableModels struct {
	AssistantCatalog assistant.ModelCatalog `resolve:""`
}

// Initialize registers the ListAvailableModels use case in the dependency container.
func (i InitListAvailableModels) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListAvailableModels](NewListAvailableModelsImpl(
		i.AssistantCatalog,
	))
	return ctx, nil
}

// InitListChatMessages is the initializer for the ListChatMessages use case
type InitListChatMessages struct {
	Repo assistant.ChatMessageRepository `resolve:""`
}

// Initialize registers the ListChatMessages use case in the dependency container.
func (i InitListChatMessages) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListChatMessages](NewListChatMessagesImpl(i.Repo))
	return ctx, nil
}

// InitListConversations is the initializer for the ListConversations use case
type InitListConversations struct {
	ConversationRepo assistant.ConversationRepository `resolve:""`
}

// Initialize registers the ListConversations use case in the dependency container.
func (init InitListConversations) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListConversations](NewListConversationsImpl(init.ConversationRepo))
	return ctx, nil
}

// InitStreamChat is the initializer for the StreamChat use case
type InitStreamChat struct {
	Logger                  *log.Logger                             `resolve:""`
	ChatMessageRepo         assistant.ChatMessageRepository         `resolve:""`
	ConversationSummaryRepo assistant.ConversationSummaryRepository `resolve:""`
	ConversationRepo        assistant.ConversationRepository        `resolve:""`
	Uow                     transaction.UnitOfWork                  `resolve:""`
	TimeProvider            core.CurrentTimeProvider                `resolve:""`
	ActionRegistry          assistant.ActionRegistry                `resolve:""`
	SkillRegistry           assistant.SkillRegistry                 `resolve:""`
	ApprovalDispatcher      assistant.ActionApprovalDispatcher      `resolve:""`
	Assistant               assistant.Assistant                     `resolve:""`
	EmbeddingModel          string                                  `config:"LLM_EMBEDDING_MODEL"`
	MaxActionCycles         int                                     `config:"LLM_MAX_ACTION_CYCLES" default:"50"`
}

// Initialize registers the StreamChat use case in the dependency container.
func (i InitStreamChat) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[StreamChat](NewStreamChatImpl(
		i.Logger,
		i.ChatMessageRepo,
		i.ConversationSummaryRepo,
		i.ConversationRepo,
		i.TimeProvider,
		i.Assistant,
		i.ActionRegistry,
		i.SkillRegistry,
		i.ApprovalDispatcher,
		i.Uow,
		i.EmbeddingModel,
		i.MaxActionCycles,
	))
	return ctx, nil
}

// InitSubmitActionApproval is the initializer for the SubmitActionApproval use case.
type InitSubmitActionApproval struct {
	Publisher outbox.EventPublisher `resolve:""`
}

// Initialize registers the SubmitActionApproval use case in the dependency container.
func (i InitSubmitActionApproval) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[SubmitActionApproval](NewSubmitActionApprovalImpl(i.Publisher))
	return ctx, nil
}

// InitUpdateConversation initializes the UpdateConversation use case and registers it in the dependency container.
type InitUpdateConversation struct {
	Uow          transaction.UnitOfWork   `resolve:""`
	TimeProvider core.CurrentTimeProvider `resolve:""`
}

// Initialize registers the UpdateConversation use case in the dependency container.
func (i InitUpdateConversation) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[UpdateConversation](NewUpdateConversationImpl(i.Uow, i.TimeProvider))
	return ctx, nil
}
