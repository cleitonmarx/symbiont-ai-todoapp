package approvaldispatcher

import (
	"context"
	"sync"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// Dispatcher coordinates action approval decisions using in-memory Go channels.
type Dispatcher struct {
	mu      sync.Mutex
	waiters map[assistant.ActionApprovalKey]chan assistant.ActionApprovalDecision
}

// NewDispatcher creates a new in-memory channel-backed approval dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		waiters: make(map[assistant.ActionApprovalKey]chan assistant.ActionApprovalDecision),
	}
}

// Wait blocks until a decision is dispatched for the given key, or context is canceled.
func (d *Dispatcher) Wait(ctx context.Context, key assistant.ActionApprovalKey) (assistant.ActionApprovalDecision, error) {
	_, span := telemetry.Start(ctx)
	defer span.End()

	ch := d.registerWaiter(key)
	defer d.unregisterWaiter(key, ch)

	select {
	case decision := <-ch:
		return decision, nil
	case <-ctx.Done():
		return assistant.ActionApprovalDecision{}, ctx.Err()
	}
}

// Dispatch sends a decision to an active waiter. Returns false when no waiter exists.
func (d *Dispatcher) Dispatch(ctx context.Context, decision assistant.ActionApprovalDecision) bool {
	_, span := telemetry.Start(ctx)
	defer span.End()

	ch := d.takeWaiter(decision.Key)
	if ch == nil {
		return false
	}

	ch <- decision
	return true
}

// registerWaiter creates and registers a new channel for the given key, returning the channel to wait on.
func (d *Dispatcher) registerWaiter(key assistant.ActionApprovalKey) chan assistant.ActionApprovalDecision {
	d.mu.Lock()
	defer d.mu.Unlock()

	ch := make(chan assistant.ActionApprovalDecision, 1)
	d.waiters[key] = ch
	return ch
}

// unregisterWaiter removes the channel for the given key if it matches the expected channel, preventing leaks.
func (d *Dispatcher) unregisterWaiter(key assistant.ActionApprovalKey, expected chan assistant.ActionApprovalDecision) {
	d.mu.Lock()
	defer d.mu.Unlock()

	current := d.waiters[key]
	if current == expected {
		delete(d.waiters, key)
	}
}

// takeWaiter atomically retrieves and removes the channel for the given key, returning nil if no waiter exists.
func (d *Dispatcher) takeWaiter(key assistant.ActionApprovalKey) chan assistant.ActionApprovalDecision {
	d.mu.Lock()
	defer d.mu.Unlock()

	ch, found := d.waiters[key]
	if !found {
		return nil
	}
	delete(d.waiters, key)
	return ch
}
