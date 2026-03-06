package outbox

import (
	"context"
	"log"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitRelay is used to initialize the Relay in the dependency container
type InitRelay struct {
	Uow       transaction.UnitOfWork `resolve:""`
	Logger    *log.Logger            `resolve:""`
	Publisher outbox.EventPublisher  `resolve:""`
}

// Initialize registers the outbox relay use case in the dependency container.
func (iro InitRelay) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[Relay](NewRelayImpl(iro.Uow, iro.Publisher, iro.Logger))
	return ctx, nil
}
