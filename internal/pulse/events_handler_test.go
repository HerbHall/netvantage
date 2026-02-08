package pulse

import (
	"context"
	"testing"

	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

func newTestModule(t *testing.T) (*Module, *PulseStore) {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "pulse", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ps := NewPulseStore(db.DB())
	m := &Module{
		logger: zap.NewNop(),
		store:  ps,
		cfg:    DefaultConfig(),
	}
	return m, ps
}

func TestHandleDeviceDiscovered_CreatesCheck(t *testing.T) {
	m, ps := newTestModule(t)
	ctx := context.Background()

	device := &models.Device{
		ID:          "device-001",
		Hostname:    "test-host",
		IPAddresses: []string{"192.168.1.100"},
	}

	event := plugin.Event{
		Topic: "recon.device.discovered",
		Payload: &recon.DeviceEvent{
			ScanID: "scan-123",
			Device: device,
		},
	}

	m.handleDeviceDiscovered(ctx, event)

	check, err := ps.GetCheckByDeviceID(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetCheckByDeviceID() error = %v", err)
	}
	if check == nil {
		t.Fatal("expected check to be created, got nil")
	}

	expectedID := "pulse-device-001"
	if check.ID != expectedID {
		t.Errorf("check.ID = %q, want %q", check.ID, expectedID)
	}
	if check.DeviceID != device.ID {
		t.Errorf("check.DeviceID = %q, want %q", check.DeviceID, device.ID)
	}
	if check.CheckType != "icmp" {
		t.Errorf("check.CheckType = %q, want %q", check.CheckType, "icmp")
	}
	if check.Target != device.IPAddresses[0] {
		t.Errorf("check.Target = %q, want %q", check.Target, device.IPAddresses[0])
	}
	if !check.Enabled {
		t.Error("check.Enabled = false, want true")
	}
	if check.IntervalSeconds != int(m.cfg.CheckInterval.Seconds()) {
		t.Errorf("check.IntervalSeconds = %d, want %d", check.IntervalSeconds, int(m.cfg.CheckInterval.Seconds()))
	}
}

func TestHandleDeviceDiscovered_DuplicateIgnored(t *testing.T) {
	m, ps := newTestModule(t)
	ctx := context.Background()

	device := &models.Device{
		ID:          "device-002",
		Hostname:    "test-host",
		IPAddresses: []string{"192.168.1.101", "192.168.1.102"},
	}

	event := plugin.Event{
		Topic: "recon.device.discovered",
		Payload: &recon.DeviceEvent{
			ScanID: "scan-123",
			Device: device,
		},
	}

	// First event should create a check
	m.handleDeviceDiscovered(ctx, event)

	firstCheck, err := ps.GetCheckByDeviceID(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetCheckByDeviceID() error = %v", err)
	}
	if firstCheck == nil {
		t.Fatal("expected check to be created on first event")
	}

	// Second event should be ignored (no duplicate)
	updatedDevice := &models.Device{
		ID:          device.ID,
		Hostname:    "test-host-updated",
		IPAddresses: []string{"192.168.1.103"}, // Different IP
	}
	event.Payload = &recon.DeviceEvent{
		ScanID: "scan-456",
		Device: updatedDevice,
	}

	m.handleDeviceDiscovered(ctx, event)

	secondCheck, err := ps.GetCheckByDeviceID(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetCheckByDeviceID() error = %v", err)
	}

	// Should still have the original check with original target
	if secondCheck.Target != firstCheck.Target {
		t.Errorf("check.Target changed from %q to %q, expected no change", firstCheck.Target, secondCheck.Target)
	}
	if secondCheck.CreatedAt != firstCheck.CreatedAt {
		t.Error("check.CreatedAt changed, expected check to be unchanged")
	}
}

func TestHandleDeviceDiscovered_InvalidPayload(t *testing.T) {
	m, ps := newTestModule(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		payload interface{}
	}{
		{
			name:    "nil payload",
			payload: nil,
		},
		{
			name:    "wrong type - string",
			payload: "not a device event",
		},
		{
			name:    "wrong type - map",
			payload: map[string]string{"key": "value"},
		},
		{
			name: "wrong struct type",
			payload: &models.Device{
				ID:          "device-003",
				IPAddresses: []string{"192.168.1.100"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := plugin.Event{
				Topic:   "recon.device.discovered",
				Payload: tt.payload,
			}

			// Should not panic or create any checks
			m.handleDeviceDiscovered(ctx, event)

			// Verify no checks were created
			// We can't query by device ID here since we don't have a valid device
			// but we can check that ListEnabledChecks returns empty
			checks, err := ps.ListEnabledChecks(ctx)
			if err != nil {
				t.Fatalf("ListEnabledChecks() error = %v", err)
			}
			if len(checks) > 0 {
				t.Errorf("expected no checks to be created, got %d", len(checks))
			}
		})
	}
}

func TestHandleDeviceDiscovered_NoIPAddresses(t *testing.T) {
	m, ps := newTestModule(t)
	ctx := context.Background()

	tests := []struct {
		name   string
		device *models.Device
	}{
		{
			name: "empty IP addresses",
			device: &models.Device{
				ID:          "device-004",
				Hostname:    "no-ip-host",
				IPAddresses: []string{},
			},
		},
		{
			name: "nil IP addresses",
			device: &models.Device{
				ID:          "device-005",
				Hostname:    "nil-ip-host",
				IPAddresses: nil,
			},
		},
		{
			name:   "nil device",
			device: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := plugin.Event{
				Topic: "recon.device.discovered",
				Payload: &recon.DeviceEvent{
					ScanID: "scan-123",
					Device: tt.device,
				},
			}

			m.handleDeviceDiscovered(ctx, event)

			// Verify no check was created
			if tt.device != nil && tt.device.ID != "" {
				check, err := ps.GetCheckByDeviceID(ctx, tt.device.ID)
				if err != nil {
					t.Fatalf("GetCheckByDeviceID() error = %v", err)
				}
				if check != nil {
					t.Errorf("expected no check for device without IP, got check ID %q", check.ID)
				}
			}
		})
	}
}

func TestHandleDeviceDiscovered_NilStore(t *testing.T) {
	// Module with nil store should return early without panicking
	m := &Module{
		logger: zap.NewNop(),
		store:  nil,
		cfg:    DefaultConfig(),
	}

	ctx := context.Background()
	device := &models.Device{
		ID:          "device-006",
		Hostname:    "test-host",
		IPAddresses: []string{"192.168.1.100"},
	}

	event := plugin.Event{
		Topic: "recon.device.discovered",
		Payload: &recon.DeviceEvent{
			ScanID: "scan-123",
			Device: device,
		},
	}

	// Should not panic
	m.handleDeviceDiscovered(ctx, event)
}
