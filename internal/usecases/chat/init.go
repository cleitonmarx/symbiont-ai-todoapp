package chat

import (
	"context"
	"log"
	"time"

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

// InitConversationCompactor initializes the synchronous conversation compactor.
type InitConversationCompactor struct {
	ChatMessageRepo         assistant.ChatMessageRepository         `resolve:""`
	ConversationSummaryRepo assistant.ConversationSummaryRepository `resolve:""`
	TimeProvider            core.CurrentTimeProvider                `resolve:""`
	Assistant               assistant.Assistant                     `resolve:""`
	Model                   string                                  `config:"LLM_CHAT_SUMMARY_MODEL"`
}

// Initialize registers the ConversationCompactor in the dependency container.
func (i InitConversationCompactor) Initialize(ctx context.Context) (context.Context, error) {
	compactor := NewConversationCompactorImpl(
		i.ChatMessageRepo,
		i.ConversationSummaryRepo,
		i.TimeProvider,
		i.Assistant,
		i.Model,
	)
	depend.Register[ConversationCompactor](compactor)
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
	lock, _ := depend.Resolve[core.Locker]()
	depend.Register[GenerateConversationTitle](NewGenerateConversationTitleImpl(
		i.ConversationRepo,
		i.ConversationSummaryRepo,
		i.ChatMessageRepo,
		lock,
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

// InitListAvailableSkills is the initializer for the ListAvailableSkills use case.
type InitListAvailableSkills struct {
	SkillRegistry assistant.SkillRegistry `resolve:""`
}

// Initialize registers the ListAvailableSkills use case in the dependency container.
func (i InitListAvailableSkills) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListAvailableSkills](NewListAvailableSkillsImpl(
		i.SkillRegistry,
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
	Logger                  *log.Logger                      `resolve:""`
	TimeProvider            core.CurrentTimeProvider         `resolve:""`
	ConversationRepo        assistant.ConversationRepository `resolve:""`
	ConversationCompactor   ConversationCompactor            `resolve:""`
	CompactionTriggerTokens int                              `config:"CHAT_COMPACTION_TRIGGER_TOKENS"`
	CompactionTimeout       time.Duration                    `config:"CHAT_COMPACTION_TIMEOUT" default:"20s"`
	StateBuilder            TurnStateBuilder                 `resolve:""`
	TurnRunner              TurnRunner                       `resolve:""`
	TranscriptWriter        ConversationTranscriptWriter     `resolve:""`
	MaxActionCycles         int                              `config:"LLM_MAX_ACTION_CYCLES" default:"50"`
}

// Initialize registers the StreamChat use case in the dependency container.
func (i InitStreamChat) Initialize(ctx context.Context) (context.Context, error) {
	useCase := NewStreamChatImpl(
		i.Logger,
		i.TimeProvider,
		i.ConversationRepo,
		i.ConversationCompactor,
		assistant.CompactionPolicy{TriggerTokenCount: i.CompactionTriggerTokens},
		i.CompactionTimeout,
		i.MaxActionCycles,
		i.StateBuilder,
		i.TurnRunner,
		i.TranscriptWriter,
	)
	depend.Register[StreamChat](useCase)
	return ctx, nil
}

// InitConversationTranscriptWriter is the initializer for the ConversationTranscriptWriter component.
type InitConversationTranscriptWriter struct {
	Uow       transaction.UnitOfWork `resolve:""`
	Tokenizer assistant.Tokenizer    `resolve:""`
}

// Initialize registers the ConversationTranscriptWriter component in the dependency container.
func (i InitConversationTranscriptWriter) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ConversationTranscriptWriter](NewConversationTranscriptWriterImpl(
		i.Uow,
		i.Tokenizer,
	))
	return ctx, nil
}

// InitActionPipeline is the initializer for the ActionPipeline component.
type InitActionPipeline struct {
	ActionRegistry     assistant.ActionRegistry           `resolve:""`
	ApprovalDispatcher assistant.ActionApprovalDispatcher `resolve:""`
	TranscriptWriter   ConversationTranscriptWriter       `resolve:""`
	TimeProvider       core.CurrentTimeProvider           `resolve:""`
}

// Initialize registers the ActionPipeline component in the dependency container.
func (i InitActionPipeline) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ActionPipeline](NewActionPipelineImpl(
		i.ActionRegistry,
		i.ApprovalDispatcher,
		i.TranscriptWriter,
		i.TimeProvider,
	))
	return ctx, nil
}

// InitTurnRunner is the initializer for the TurnRunner component.
type InitTurnRunner struct {
	Logger         *log.Logger         `resolve:""`
	Assistant      assistant.Assistant `resolve:""`
	ActionPipeline ActionPipeline      `resolve:""`
}

// Initialize registers the TurnRunner component in the dependency container.
func (i InitTurnRunner) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[TurnRunner](NewTurnRunnerImpl(
		i.Logger,
		i.Assistant,
		i.ActionPipeline,
	))
	return ctx, nil
}

// InitTurnStateBuilder is the initializer for the TurnStateBuilder component.
type InitTurnStateBuilder struct {
	ConversationSummaryRepo assistant.ConversationSummaryRepository `resolve:""`
	ChatMessageRepo         assistant.ChatMessageRepository         `resolve:""`
	TimeProvider            core.CurrentTimeProvider                `resolve:""`
	SkillRegistry           assistant.SkillRegistry                 `resolve:""`
	ActionRegistry          assistant.ActionRegistry                `resolve:""`
}

// Initialize registers the TurnStateBuilder component in the dependency container.
func (i InitTurnStateBuilder) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[TurnStateBuilder](NewTurnStateBuilderImpl(
		i.ConversationSummaryRepo,
		i.ChatMessageRepo,
		i.TimeProvider,
		i.SkillRegistry,
		i.ActionRegistry,
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
