package app

import (
	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/graphql"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/workers"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/actionregistry/composite"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/actionregistry/local"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/actionregistry/mcp"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/approvaldispatcher"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/config"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/log"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/md"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/modelrunner"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/postgres"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/pubsub"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/time"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/tokenizer"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/board"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/chat"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
)

// NewMonolithic builds the all-in-one deployable.
// It hosts the HTTP server (REST API + embedded webapp static files), GraphQL API,
// action approval dispatcher, message relay, board summary generator,
// and conversation title generator in a single process.
// Optional initializers are executed before the default wiring initializers.
func NewMonolithic(initializers ...symbiont.Initializer) *symbiont.App {
	return symbiont.NewApp().
		Initialize(initializers...).
		Initialize(
			&log.InitLogger{},
			&telemetry.InitOpenTelemetry{},
			&telemetry.InitHttpClient{},
			&config.InitVaultProvider{},
			&postgres.InitDB{},
			&modelrunner.InitAssistantClient{},
			&modelrunner.InitEncoderClient{},
			&pubsub.InitClient{},
			&postgres.InitUnitOfWork{},
			&postgres.InitTodoRepository{},
			&postgres.InitBoardSummaryRepository{},
			&postgres.InitChatMessageRepository{},
			&postgres.InitConversationRepository{},
			&postgres.InitLocker{},
			&postgres.InitConversationSummaryRepository{},
			&time.InitCurrentTimeProvider{},
			&tokenizer.InitTokenizer{},
			&approvaldispatcher.InitDispatcher{},
			&pubsub.InitPublisher{},
			&md.InitSkillRegistry{},
			&todo.InitCreator{},
			&todo.InitDeleter{},
			&todo.InitUpdater{},
			&local.InitActionRegistry{},
			&mcp.InitActionRegistry{},
			&composite.InitActionRegistry{},
			&todo.InitListTodos{},
			&todo.InitCreateTodo{},
			&todo.InitUpdateTodo{},
			&todo.InitDeleteTodo{},
			&board.InitGenerateBoardSummary{},
			&chat.InitConversationCompactor{},
			&chat.InitConversationTranscriptWriter{},
			&chat.InitActionPipeline{},
			&chat.InitTurnRunner{},
			&chat.InitTurnStateBuilder{},
			&chat.InitGenerateConversationTitle{},
			&board.InitGetBoardSummary{},
			&chat.InitListConversations{},
			&chat.InitUpdateConversation{},
			&chat.InitListChatMessages{},
			&chat.InitSubmitActionApproval{},
			&chat.InitDeleteConversation{},
			&chat.InitStreamChat{},
			&chat.InitListAvailableModels{},
			&chat.InitListAvailableSkills{},
			&outbox.InitRelay{},
		).
		Host(
			&http.TodoAppServer{},
			&graphql.TodoGraphQLServer{},
			&workers.BoardSummaryGenerator{},
			&workers.ConversationTitleGenerator{},
			&workers.ActionApprovalDispatcher{},
			&workers.MessageRelay{},
		)
}

// NewHTTPAPI builds the HTTP API deployable.
// It hosts the HTTP server (REST API + embedded webapp static files)
// and action approval dispatcher in one process.
func NewHTTPAPI() *symbiont.App {
	return symbiont.NewApp().
		Initialize(
			&log.InitLogger{},
			&telemetry.InitOpenTelemetry{},
			&telemetry.InitHttpClient{},
			&config.InitVaultProvider{},
			&postgres.InitDB{},
			&modelrunner.InitAssistantClient{},
			&modelrunner.InitEncoderClient{},
			&pubsub.InitClient{},
			&postgres.InitUnitOfWork{},
			&postgres.InitTodoRepository{},
			&postgres.InitBoardSummaryRepository{},
			&postgres.InitChatMessageRepository{},
			&postgres.InitConversationRepository{},
			&postgres.InitConversationSummaryRepository{},
			&time.InitCurrentTimeProvider{},
			&tokenizer.InitTokenizer{},
			&approvaldispatcher.InitDispatcher{},
			&pubsub.InitPublisher{},
			&md.InitSkillRegistry{},
			&todo.InitCreator{},
			&todo.InitDeleter{},
			&todo.InitUpdater{},
			&local.InitActionRegistry{},
			&mcp.InitActionRegistry{},
			&composite.InitActionRegistry{},
			&todo.InitListTodos{},
			&todo.InitCreateTodo{},
			&todo.InitUpdateTodo{},
			&todo.InitDeleteTodo{},
			&board.InitGetBoardSummary{},
			&chat.InitConversationCompactor{},
			&chat.InitConversationTranscriptWriter{},
			&chat.InitActionPipeline{},
			&chat.InitTurnRunner{},
			&chat.InitTurnStateBuilder{},
			&chat.InitListConversations{},
			&chat.InitUpdateConversation{},
			&chat.InitListChatMessages{},
			&chat.InitSubmitActionApproval{},
			&chat.InitDeleteConversation{},
			&chat.InitStreamChat{},
			&chat.InitListAvailableModels{},
			&chat.InitListAvailableSkills{},
		).
		Host(
			&http.TodoAppServer{},
			&workers.ActionApprovalDispatcher{},
		)
}

// NewGraphQLAPI builds the GraphQL API deployable.
// It hosts only the GraphQL server in a dedicated process.
func NewGraphQLAPI() *symbiont.App {
	return symbiont.NewApp().
		Initialize(
			&log.InitLogger{},
			&telemetry.InitOpenTelemetry{},
			&telemetry.InitHttpClient{},
			&config.InitVaultProvider{},
			&postgres.InitDB{SkipMigration: true},
			&modelrunner.InitEncoderClient{},
			&postgres.InitUnitOfWork{},
			&postgres.InitTodoRepository{},
			&time.InitCurrentTimeProvider{},
			&todo.InitDeleter{},
			&todo.InitUpdater{},
			&todo.InitListTodos{},
			&todo.InitUpdateTodo{},
			&todo.InitDeleteTodo{},
		).
		Host(
			&graphql.TodoGraphQLServer{},
		)
}

// NewMessageRelay builds the outbox relay worker deployable.
// It hosts the message relay worker in a dedicated process.
func NewMessageRelay() *symbiont.App {
	return symbiont.NewApp().
		Initialize(
			&log.InitLogger{},
			&telemetry.InitOpenTelemetry{},
			&config.InitVaultProvider{},
			&postgres.InitDB{SkipMigration: true},
			&pubsub.InitClient{},
			&postgres.InitUnitOfWork{},
			&pubsub.InitPublisher{},
			&outbox.InitRelay{},
		).
		Host(
			&workers.MessageRelay{},
		)
}

// NewBoardSummaryGenerator builds the board summary generator deployable.
// It hosts the board summary generator in a dedicated process.
func NewBoardSummaryGenerator() *symbiont.App {
	return symbiont.NewApp().
		Initialize(
			&log.InitLogger{},
			&telemetry.InitOpenTelemetry{},
			&telemetry.InitHttpClient{},
			&config.InitVaultProvider{},
			&postgres.InitDB{SkipMigration: true},
			&postgres.InitLocker{},
			&modelrunner.InitAssistantClient{},
			&pubsub.InitClient{},
			&postgres.InitBoardSummaryRepository{},
			&time.InitCurrentTimeProvider{},
			&board.InitGenerateBoardSummary{},
		).
		Host(
			&workers.BoardSummaryGenerator{},
		)
}

// NewConversationTitleGenerator builds the conversation title generator deployable.
// It hosts the conversation title generator in a dedicated process.
func NewConversationTitleGenerator() *symbiont.App {
	return symbiont.NewApp().
		Initialize(
			&log.InitLogger{},
			&telemetry.InitOpenTelemetry{},
			&telemetry.InitHttpClient{},
			&config.InitVaultProvider{},
			&postgres.InitDB{SkipMigration: true},
			&modelrunner.InitAssistantClient{},
			&pubsub.InitClient{},
			&postgres.InitChatMessageRepository{},
			&postgres.InitConversationRepository{},
			&postgres.InitLocker{},
			&postgres.InitConversationSummaryRepository{},
			&time.InitCurrentTimeProvider{},
			&chat.InitGenerateConversationTitle{},
		).
		Host(
			&workers.ConversationTitleGenerator{},
		)
}
