package app

import (
	"context"
	"encoding/json"
	stdlog "log"

	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/workers"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/outbound/config"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/outbound/log"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/outbound/modelrunner"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/outbound/postgres"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/outbound/pubsub"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/outbound/time"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/cleitonmarx/symbiont/introspection/mermaid"
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
			&time.InitCurrentTimeProvider{},
			&pubsub.InitClient{},
			&pubsub.InitPublisher{},
			&modelrunner.InitLLMClient{},

			&usecases.InitTodoCreator{},
			&usecases.InitTodoDeleter{},
			&usecases.InitTodoUpdater{},
			&usecases.InitLLMToolRegistry{},

			&usecases.InitListTodos{},
			&usecases.InitCreateTodo{},
			&usecases.InitUpdateTodo{},
			&usecases.InitDeleteTodo{},
			&usecases.InitGenerateBoardSummary{},
			&usecases.InitGetBoardSummary{},
			&usecases.InitListChatMessages{},
			&usecases.InitDeleteConversation{},
			&usecases.InitStreamChat{},
			&usecases.InitRelayOutbox{},
		).
		Host(
			&http.TodoAppServer{},
			&graphql.TodoGraphQLServer{},
			&workers.TodoEventSubscriber{},
			&workers.MessageRelay{},
		)
}

// ReportLoggerIntrospector is an implementation of introspection.Introspector that logs the introspection report.
type ReportLoggerIntrospector struct {
	Logger *stdlog.Logger `resolve:""`
}

// Introspect generates and logs the introspection report and a Mermaid graph.
func (i ReportLoggerIntrospector) Introspect(ctx context.Context, r introspection.Report) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	i.Logger.Println("=== TODOAPP INTROSPECTION REPORT ===")
	i.Logger.Println(string(b))
	i.Logger.Println("=== MERMAID GRAPH ===")
	i.Logger.Println(mermaid.GenerateIntrospectionGraph(r))
	i.Logger.Println("=== END OF REPORT ===")
	return nil
}
