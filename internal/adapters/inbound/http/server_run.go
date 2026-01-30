package http

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var _ gen.ServerInterface = (*TodoAppServer)(nil)

// TodoAppServer is the HTTP server adapter for the TodoApp application.
//
// It implements the OpenAPI-generated ServerInterface and serves both the REST API
// endpoints and the embedded web application static files. The server is instrumented
// with OpenTelemetry for distributed tracing and configured via environment variables
// or configuration providers through the symbiont framework.
//
// Dependencies are automatically resolved and injected at initialization time.
type TodoAppServer struct {
	Port                      int                         `config:"HTTP_PORT" default:"8080"`
	Logger                    *log.Logger                 `resolve:""`
	ListTodosUseCase          usecases.ListTodos          `resolve:""`
	CreateTodoUseCase         usecases.CreateTodo         `resolve:""`
	UpdateTodoUseCase         usecases.UpdateTodo         `resolve:""`
	DeleteTodoUseCase         usecases.DeleteTodo         `resolve:""`
	GetBoardSummaryUseCase    usecases.GetBoardSummary    `resolve:""`
	ListChatMessagesUseCase   usecases.ListChatMessages   `resolve:""`
	DeleteConversationUseCase usecases.DeleteConversation `resolve:""`
	StreamChatUseCase         usecases.StreamChat         `resolve:""`
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

	// get an `http.Handler` that we can use
	h := gen.HandlerWithOptions(api, gen.StdHTTPServerOptions{
		BaseRouter: mux,
		Middlewares: []gen.MiddlewareFunc{
			otelhttp.NewMiddleware(
				"todoapp-api",
				otelhttp.WithSpanNameFormatter(tracing.SpanNameFormatter),
			)},
	})

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
		api.Logger.Print("TodoAppServer: Shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.Shutdown(shutdownCtx)
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
