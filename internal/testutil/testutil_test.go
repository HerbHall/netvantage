package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/netvantage/pkg/models"
	"github.com/HerbHall/netvantage/pkg/plugin"
)

func TestLogger_NotNil(t *testing.T) {
	l := Logger()
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewStore_Usable(t *testing.T) {
	db := NewStore(t)
	if db == nil {
		t.Fatal("expected non-nil store")
	}
	if err := db.DB().PingContext(context.Background()); err != nil {
		t.Fatalf("PingContext: %v", err)
	}
}

func TestMockBus_RecordsEvents(t *testing.T) {
	bus := NewMockBus()

	ev := plugin.Event{Topic: "test.topic", Source: "test"}
	if err := bus.Publish(context.Background(), ev); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	bus.PublishAsync(context.Background(), plugin.Event{Topic: "test.async", Source: "test"})

	events := bus.Events()
	if len(events) != 2 {
		t.Fatalf("Events len = %d, want 2", len(events))
	}
	if events[0].Topic != "test.topic" {
		t.Errorf("events[0].Topic = %q, want test.topic", events[0].Topic)
	}
	if events[1].Topic != "test.async" {
		t.Errorf("events[1].Topic = %q, want test.async", events[1].Topic)
	}
}

func TestMockBus_Reset(t *testing.T) {
	bus := NewMockBus()
	_ = bus.Publish(context.Background(), plugin.Event{Topic: "a"})
	bus.Reset()
	if len(bus.Events()) != 0 {
		t.Error("expected empty events after Reset")
	}
}

func TestClock_Advance(t *testing.T) {
	c := NewClock()
	start := c.Now()
	c.Advance(5 * time.Minute)
	if got := c.Now().Sub(start); got != 5*time.Minute {
		t.Errorf("Advance: elapsed = %v, want 5m", got)
	}
}

func TestClock_Set(t *testing.T) {
	c := NewClock()
	target := time.Date(2030, 6, 15, 12, 0, 0, 0, time.UTC)
	c.Set(target)
	if !c.Now().Equal(target) {
		t.Errorf("Set: got %v, want %v", c.Now(), target)
	}
}

func TestNewDevice_Defaults(t *testing.T) {
	d := NewDevice()
	if d.ID == "" {
		t.Error("expected non-empty ID")
	}
	if d.Status != models.DeviceStatusOnline {
		t.Errorf("Status = %q, want online", d.Status)
	}
	if d.Hostname != "test-device" {
		t.Errorf("Hostname = %q, want test-device", d.Hostname)
	}
}

func TestNewDevice_WithOptions(t *testing.T) {
	d := NewDevice(
		WithHostname("myhost"),
		WithIP("10.0.0.1"),
		WithStatus(models.DeviceStatusOffline),
	)
	if d.Hostname != "myhost" {
		t.Errorf("Hostname = %q, want myhost", d.Hostname)
	}
	if len(d.IPAddresses) != 1 || d.IPAddresses[0] != "10.0.0.1" {
		t.Errorf("IPAddresses = %v, want [10.0.0.1]", d.IPAddresses)
	}
	if d.Status != models.DeviceStatusOffline {
		t.Errorf("Status = %q, want offline", d.Status)
	}
}
