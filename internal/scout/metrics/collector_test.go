package metrics

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestNewCollector_ReturnsNonNil(t *testing.T) {
	logger := zap.NewNop()
	c := NewCollector(logger)
	if c == nil {
		t.Fatal("NewCollector returned nil")
	}
}

func TestCollect_ReturnsNonNilMetrics(t *testing.T) {
	logger := zap.NewNop()
	c := NewCollector(logger)

	ctx := context.Background()
	m, err := c.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if m == nil {
		t.Fatal("Collect returned nil metrics")
	}
}

func TestCollect_CancelledContext(t *testing.T) {
	logger := zap.NewNop()
	c := NewCollector(logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	m, err := c.Collect(ctx)
	// On non-Windows the stub ignores context; on Windows CPU collection
	// returns a context error, but the overall Collect still returns a
	// partial result (zero CPU) rather than failing.
	if m == nil {
		t.Fatal("Collect returned nil metrics even with cancelled context")
	}
	// err may or may not be nil depending on platform; just ensure no panic.
	_ = err
}
