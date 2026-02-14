//go:build !windows

package recon

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"go.uber.org/zap"
)

func TestNewMDNSListener(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	s := NewReconStore(db.DB())
	bus := &mockEventBus{}
	logger := zap.NewNop()

	listener := NewMDNSListener(s, bus, logger, 30*time.Second)
	if listener == nil {
		t.Fatal("NewMDNSListener returned nil")
	}
	if listener.store != s {
		t.Error("store not set")
	}
	if listener.bus != bus {
		t.Error("bus not set")
	}
	if listener.interval != 30*time.Second {
		t.Errorf("interval = %v, want 30s", listener.interval)
	}
	if listener.seen == nil {
		t.Error("seen map not initialized")
	}
}

func TestMDNSListener_RunStopsOnCancel(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	s := NewReconStore(db.DB())
	bus := &mockEventBus{}
	logger := zap.NewNop()

	// Use a very long interval so we only test cancellation, not ticks.
	listener := NewMDNSListener(s, bus, logger, 10*time.Minute)

	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		listener.Run(runCtx)
		close(done)
	}()

	// Cancel and verify the goroutine exits promptly.
	cancel()

	select {
	case <-done:
		// Run exited cleanly.
	case <-time.After(5 * time.Second):
		t.Fatal("MDNSListener.Run did not stop within 5 seconds after cancellation")
	}
}

func TestMDNSListener_Deduplication(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	s := NewReconStore(db.DB())
	bus := &mockEventBus{}
	logger := zap.NewNop()

	listener := NewMDNSListener(s, bus, logger, 60*time.Second)

	ip := "192.168.1.100"

	// First time: not recently seen.
	if listener.recentlySeen(ip) {
		t.Error("IP should not be recently seen before marking")
	}

	// Mark and check.
	listener.markSeen(ip)
	if !listener.recentlySeen(ip) {
		t.Error("IP should be recently seen after marking")
	}

	// Second check: still recently seen.
	if !listener.recentlySeen(ip) {
		t.Error("IP should still be recently seen")
	}
}

func TestMDNSListener_CleanSeen(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	s := NewReconStore(db.DB())
	logger := zap.NewNop()

	// Very short interval so entries expire quickly.
	listener := NewMDNSListener(s, nil, logger, 10*time.Millisecond)

	listener.markSeen("10.0.0.1")
	listener.markSeen("10.0.0.2")

	// Both should exist.
	listener.mu.Lock()
	if len(listener.seen) != 2 {
		t.Errorf("seen map has %d entries, want 2", len(listener.seen))
	}
	listener.mu.Unlock()

	// Wait for entries to expire (2x interval = 20ms).
	time.Sleep(30 * time.Millisecond)

	listener.cleanSeen()

	listener.mu.Lock()
	if len(listener.seen) != 0 {
		t.Errorf("seen map has %d entries after clean, want 0", len(listener.seen))
	}
	listener.mu.Unlock()
}

func TestInferDeviceTypeFromService(t *testing.T) {
	tests := []struct {
		service string
		want    string
	}{
		{"_ipp._tcp", "printer"},
		{"_printer._tcp", "printer"},
		{"_airplay._tcp", "iot"},
		{"_raop._tcp", "iot"},
		{"_googlecast._tcp", "iot"},
		{"_homekit._tcp", "iot"},
		{"_hap._tcp", "iot"},
		{"_mqtt._tcp", "iot"},
		{"_http._tcp", "unknown"},
		{"_ssh._tcp", "unknown"},
		{"_workstation._tcp", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			got := string(inferDeviceTypeFromService(tt.service))
			if got != tt.want {
				t.Errorf("inferDeviceTypeFromService(%q) = %q, want %q", tt.service, got, tt.want)
			}
		})
	}
}
