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
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/modelrunner"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/postgres"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/pubsub"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/skillregistry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/time"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/board"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/chat"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
)

// NewMonolithic builds the all-in-one deployable.
// It hosts the HTTP server (REST API + embedded webapp static files), GraphQL API,
// action approval dispatcher, message relay, board summary generator,
// chat summary generator, and conversation title generator in a single process.
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
			&postgres.InitConversationSummaryRepository{},
			&time.InitCurrentTimeProvider{},
			&approvaldispatcher.InitDispatcher{},
			&pubsub.InitPublisher{},
			&skillregistry.InitLocalSkillRegistry{},
			&todo.InitCreator{},
			&todo.InitDeleter{},
			&todo.InitUpdater{},
			&local.InitLocalActionRegistry{},
			&mcp.InitMCPActionRegistry{},
			&composite.InitCompositeActionRegistry{},
			&todo.InitListTodos{},
			&todo.InitCreateTodo{},
			&todo.InitUpdateTodo{},
			&todo.InitDeleteTodo{},
			&board.InitGenerateBoardSummary{},
			&chat.InitGenerateChatSummary{},
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
			&workers.ChatSummaryGenerator{},
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
			&approvaldispatcher.InitDispatcher{},
			&pubsub.InitPublisher{},
			&skillregistry.InitLocalSkillRegistry{},
			&todo.InitCreator{},
			&todo.InitDeleter{},
			&todo.InitUpdater{},
			&local.InitLocalActionRegistry{},
			&mcp.InitMCPActionRegistry{},
			&composite.InitCompositeActionRegistry{},
			&todo.InitListTodos{},
			&todo.InitCreateTodo{},
			&todo.InitUpdateTodo{},
			&todo.InitDeleteTodo{},
			&board.InitGetBoardSummary{},
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

// NewChatSummaryGenerator builds the chat summary generator deployable.
// It hosts the chat summary generator in a dedicated process.
func NewChatSummaryGenerator() *symbiont.App {
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
			&postgres.InitConversationSummaryRepository{},
			&time.InitCurrentTimeProvider{},
			&chat.InitGenerateChatSummary{},
		).
		Host(
			&workers.ChatSummaryGenerator{},
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
			&postgres.InitConversationSummaryRepository{},
			&time.InitCurrentTimeProvider{},
			&chat.InitGenerateConversationTitle{},
		).
		Host(
			&workers.ConversationTitleGenerator{},
		)
}
