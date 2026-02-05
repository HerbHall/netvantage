package testutil

import (
	"context"
	"sync"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

// Compile-time interface check.
var _ plugin.EventBus = (*MockBus)(nil)

// MockBus is a thread-safe in-memory event bus that records all published
// events for later inspection.
type MockBus struct {
	mu     sync.Mutex
	events []plugin.Event
}

// NewMockBus returns a new MockBus.
func NewMockBus() *MockBus {
	return &MockBus{}
}

// Publish records an event synchronously.
func (b *MockBus) Publish(_ context.Context, event plugin.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
	return nil
}

// PublishAsync records an event (same as Publish in tests).
func (b *MockBus) PublishAsync(_ context.Context, event plugin.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
}

// Subscribe is a no-op that returns a no-op unsubscribe function.
func (b *MockBus) Subscribe(_ string, _ plugin.EventHandler) func() {
	return func() {}
}

// SubscribeAll is a no-op that returns a no-op unsubscribe function.
func (b *MockBus) SubscribeAll(_ plugin.EventHandler) func() {
	return func() {}
}

// Events returns a copy of all recorded events.
func (b *MockBus) Events() []plugin.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]plugin.Event, len(b.events))
	copy(out, b.events)
	return out
}

// Reset clears all recorded events.
func (b *MockBus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = nil
}
