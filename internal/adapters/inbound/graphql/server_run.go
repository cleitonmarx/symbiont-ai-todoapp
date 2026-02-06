package graphql

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"github.com/rs/cors"
)

type TodoGraphQLServer struct {
	Logger            *log.Logger         `resolve:""`
	ListTodosUsecase  usecases.ListTodos  `resolve:""`
	DeleteTodoUsecase usecases.DeleteTodo `resolve:""`
	UpdateTodoUsecase usecases.UpdateTodo `resolve:""`
	Port              int                 `config:"GRAPHQL_SERVER_PORT" default:"8085"`
}

func (s *TodoGraphQLServer) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	h := handler.New(
		gen.NewExecutableSchema(gen.Config{Resolvers: s}),
	)
	h.AddTransport(transport.POST{})
	h.AddTransport(transport.GET{})
	h.Use(extension.Introspection{})

	corsHandler := cors.AllowAll()

	mux.Handle("/v1/query", corsHandler.Handler(
		telemetry.HttpHandler(h, "todoapp-graphql"),
	))

	mux.Handle("/", playground.Handler("TodoApp GraphQL", "/v1/query"))

	svr := &http.Server{
		Handler: mux,
		Addr:    fmt.Sprintf(":%d", s.Port),
	}

	errCh := make(chan error, 1)
	go func() {
		s.Logger.Printf("TodoGraphQLServer: Listening on port %d", s.Port)
		errCh <- svr.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		s.Logger.Print("TodoGraphQLServer: Shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return svr.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *TodoGraphQLServer) IsReady(ctx context.Context) error {
	resp, err := http.Get(fmt.Sprintf("http://:%d", s.Port))
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
