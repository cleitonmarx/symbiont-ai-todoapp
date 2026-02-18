package app

import (
	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/graphql"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/workers"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/config"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/log"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/modelrunner"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/postgres"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/pubsub"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/outbound/time"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
)

// NewTodoApp creates and returns a new instance of the TodoApp application.
func NewTodoApp(initializers ...symbiont.Initializer) *symbiont.App {
	return symbiont.NewApp().
		Initialize(initializers...).
		Initialize(
			&log.InitLogger{},
			&telemetry.InitOpenTelemetry{},
			&telemetry.InitHttpClient{},
			&config.InitVaultProvider{},
			&postgres.InitDB{},
			&postgres.InitUnitOfWork{},
			&postgres.InitTodoRepository{},
			&postgres.InitBoardSummaryRepository{},
			&postgres.InitChatMessageRepository{},
			&postgres.InitConversationRepository{},
			&postgres.InitConversationSummaryRepository{},
			&time.InitCurrentTimeProvider{},
			&pubsub.InitClient{},
			&pubsub.InitPublisher{},
			&modelrunner.InitAssistantClient{},

			&usecases.InitTodoCreator{},
			&usecases.InitTodoDeleter{},
			&usecases.InitTodoUpdater{},
			&assistant.InitAssistantActionRegistry{},

			&usecases.InitListTodos{},
			&usecases.InitCreateTodo{},
			&usecases.InitUpdateTodo{},
			&usecases.InitDeleteTodo{},
			&usecases.InitGenerateBoardSummary{},
			&usecases.InitGenerateChatSummary{},
			&usecases.InitGenerateConversationTitle{},
			&usecases.InitGetBoardSummary{},
			&usecases.InitListConversations{},
			&usecases.InitUpdateConversation{},
			&usecases.InitListChatMessages{},
			&usecases.InitDeleteConversation{},
			&usecases.InitStreamChat{},
			&usecases.InitListAvailableModels{},
			&usecases.InitRelayOutbox{},
		).
		Host(
			&http.TodoAppServer{},
			&graphql.TodoGraphQLServer{},
			&workers.BoardSummaryGenerator{},
			&workers.ChatSummaryGenerator{},
			&workers.ConversationTitleGenerator{},
			&workers.MessageRelay{},
		).
		Introspect(&MermaidGraphIntrospector{})
}
