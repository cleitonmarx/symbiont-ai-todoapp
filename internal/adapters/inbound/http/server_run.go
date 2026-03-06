package http

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/board"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/chat"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/cleitonmarx/symbiont/introspection/mermaid"
	"github.com/rs/cors"
)

var _ gen.ServerInterface = (*TodoAppServer)(nil)

// TodoAppServer is the REST API and UI HTTP server for the TodoApp application.
type TodoAppServer struct {
	Port                        int                       `config:"HTTP_PORT" default:"8080"`
	Logger                      *log.Logger               `resolve:""`
	ListTodosUseCase            todo.List                 `resolve:""`
	CreateTodoUseCase           todo.Create               `resolve:""`
	UpdateTodoUseCase           todo.Update               `resolve:""`
	DeleteTodoUseCase           todo.Delete               `resolve:""`
	GetBoardSummaryUseCase      board.GetBoardSummary     `resolve:""`
	ListConversationsUseCase    chat.ListConversations    `resolve:""`
	UpdateConversationUseCase   chat.UpdateConversation   `resolve:""`
	ListChatMessagesUseCase     chat.ListChatMessages     `resolve:""`
	SubmitActionApprovalUseCase chat.SubmitActionApproval `resolve:""`
	DeleteConversationUseCase   chat.DeleteConversation   `resolve:""`
	ListAvailableModelsUseCase  chat.ListAvailableModels  `resolve:""`
	StreamChatUseCase           chat.StreamChat           `resolve:""`
	introspectionReport         introspection.Report
}

//go:embed webappdist/*
var embedFS embed.FS

// Run starts the HTTP server for the TodoAppServer.
func (api TodoAppServer) Run(ctx context.Context) error {

	mux := http.NewServeMux()

	// Serve webapp static files
	sub, err := fs.Sub(embedFS, "webappdist")
	if err != nil {
		return fmt.Errorf("failed to create sub filesystem for webapp: %w", err)
	}
	mux.Handle("/", http.FileServerFS(sub))

	// Register introspection endpoint for debugging and testing purposes
	mux.Handle("/introspect/", mermaid.NewGraphHandler("TodoApp", api.introspectionReport))

	// Create the OpenAPI handler with telemetry middleware
	h := gen.HandlerWithOptions(api, gen.StdHTTPServerOptions{
		BaseRouter: mux,
		Middlewares: []gen.MiddlewareFunc{
			telemetry.Middleware("todoapp-api"),
		},
	})

	// Apply CORS at the top-level so preflight requests hit it, too.
	h = cors.AllowAll().Handler(h)

	s := &http.Server{
		Handler: h,
		Addr:    fmt.Sprintf(":%d", api.Port),
	}

	errCh := make(chan error, 1)
	go func() {
		api.Logger.Printf("TodoAppServer: Listening on port %d", api.Port)
		errCh <- s.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := s.Shutdown(shutdownCtx)
		if err != nil {
			api.Logger.Printf("TodoAppServer: error during shutdown: %v", err)
		} else {
			api.Logger.Println("TodoAppServer: stopped")
		}
		return err
	case err := <-errCh:
		return err
	}
}

// IsReady checks if the TodoAppServer is ready by performing a health check.
func (api TodoAppServer) IsReady(ctx context.Context) error {
	resp, err := http.Get(fmt.Sprintf("http://:%d", api.Port))
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
