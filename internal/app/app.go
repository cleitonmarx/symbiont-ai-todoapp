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
			&approvaldispatcher.InitDispatcher{},
			&pubsub.InitClient{},
			&pubsub.InitPublisher{},
			&modelrunner.InitAssistantClient{},
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
