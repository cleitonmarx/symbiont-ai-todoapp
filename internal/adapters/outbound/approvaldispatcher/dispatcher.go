package approvaldispatcher

import (
	"context"
	"sync"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
)

// Dispatcher coordinates action approval decisions using in-memory Go channels.
type Dispatcher struct {
	mu      sync.Mutex
	waiters map[domain.AssistantActionApprovalKey]chan domain.AssistantActionApprovalDecision
}

// NewDispatcher creates a new in-memory channel-backed approval dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		waiters: make(map[domain.AssistantActionApprovalKey]chan domain.AssistantActionApprovalDecision),
	}
}

// Wait blocks until a decision is dispatched for the given key, or context is canceled.
func (d *Dispatcher) Wait(ctx context.Context, key domain.AssistantActionApprovalKey) (domain.AssistantActionApprovalDecision, error) {
	_, span := telemetry.Start(ctx)
	defer span.End()

	ch := d.registerWaiter(key)
	defer d.unregisterWaiter(key, ch)

	select {
	case decision := <-ch:
		return decision, nil
	case <-ctx.Done():
		return domain.AssistantActionApprovalDecision{}, ctx.Err()
	}
}

// Dispatch sends a decision to an active waiter. Returns false when no waiter exists.
func (d *Dispatcher) Dispatch(ctx context.Context, decision domain.AssistantActionApprovalDecision) bool {
	_, span := telemetry.Start(ctx)
	defer span.End()

	ch := d.takeWaiter(decision.Key)
	if ch == nil {
		return false
	}

	ch <- decision
	return true
}

func (d *Dispatcher) registerWaiter(key domain.AssistantActionApprovalKey) chan domain.AssistantActionApprovalDecision {
	d.mu.Lock()
	defer d.mu.Unlock()

	ch := make(chan domain.AssistantActionApprovalDecision, 1)
	d.waiters[key] = ch
	return ch
}

func (d *Dispatcher) unregisterWaiter(key domain.AssistantActionApprovalKey, expected chan domain.AssistantActionApprovalDecision) {
	d.mu.Lock()
	defer d.mu.Unlock()

	current := d.waiters[key]
	if current == expected {
		delete(d.waiters, key)
	}
}

func (d *Dispatcher) takeWaiter(key domain.AssistantActionApprovalKey) chan domain.AssistantActionApprovalDecision {
	d.mu.Lock()
	defer d.mu.Unlock()

	ch, found := d.waiters[key]
	if !found {
		return nil
	}
	delete(d.waiters, key)
	return ch
}

// InitDispatcher is used to initialize and register the dispatcher.
type InitDispatcher struct{}

func (i InitDispatcher) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.AssistantActionApprovalDispatcher](NewDispatcher())
	return ctx, nil
}
