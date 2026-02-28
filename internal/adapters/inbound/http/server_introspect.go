package http

import (
	"context"

	"github.com/cleitonmarx/symbiont/introspection"
)

// Introspect is the implementation of the Introspector interface for the TodoAppServer,
// which receives the introspection report and stores it in the server struct for later use in the introspection handler.
func (api *TodoAppServer) Introspect(_ context.Context, r introspection.Report) error {
	api.introspectionReport = r
	return nil
}
