package log

import (
	"context"
	"log"
	"os"

	"github.com/cleitonmarx/symbiont/depend"
)

// InitLogger is the initializer for the logger dependency.
type InitLogger struct{}

// Initialize registers the logger in the dependency container.
func (il InitLogger) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register(log.New(os.Stdout, "", log.Lmsgprefix))
	return ctx, nil
}
