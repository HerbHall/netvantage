package recon

import (
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
)

func TestDeviceToCSVRow_ColumnCount(t *testing.T) {
	d := models.Device{
		ID:              "abc-123",
		Hostname:        "web-01",
		IPAddresses:     []string{"192.168.1.1", "10.0.0.1"},
		MACAddress:      "AA:BB:CC:DD:EE:FF",
		Manufacturer:    "TestCorp",
		DeviceType:      models.DeviceTypeServer,
		OS:              "Ubuntu 22.04",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
		LastSeen:        time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		FirstSeen:       time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC),
		Notes:           "Production server",
		Tags:            []string{"prod", "web", "critical"},
		Location:        "Rack A3",
		Category:        "production",
		PrimaryRole:     "web-server",
		Owner:           "platform-team",
	}

	row := deviceToCSVRow(d)

	if len(row) != len(csvHeaders()) {
		t.Fatalf("expected %d columns, got %d", len(csvHeaders()), len(row))
	}

	// Spot-check values.
	if row[0] != "abc-123" {
		t.Errorf("id: got %q, want %q", row[0], "abc-123")
	}
	if row[1] != "web-01" {
		t.Errorf("hostname: got %q, want %q", row[1], "web-01")
	}
	if row[2] != "192.168.1.1;10.0.0.1" {
		t.Errorf("ip_addresses: got %q, want %q", row[2], "192.168.1.1;10.0.0.1")
	}
	if row[12] != "prod;web;critical" {
		t.Errorf("tags: got %q, want %q", row[12], "prod;web;critical")
	}
}

func TestCSVRowToDevice_ValidRow(t *testing.T) {
	row := []string{
		"abc-123", "web-01", "192.168.1.1;10.0.0.1", "AA:BB:CC:DD:EE:FF",
		"TestCorp", "server", "Ubuntu 22.04", "online", "icmp",
		"2026-01-15T10:30:00Z", "2026-01-10T08:00:00Z",
		"Production server", "prod;web;critical",
		"Rack A3", "production", "web-server", "platform-team",
	}

	d, err := csvRowToDevice(row)
	if err != nil {
		t.Fatalf("csvRowToDevice: %v", err)
	}

	if d.ID != "abc-123" {
		t.Errorf("ID: got %q, want %q", d.ID, "abc-123")
	}
	if d.Hostname != "web-01" {
		t.Errorf("Hostname: got %q, want %q", d.Hostname, "web-01")
	}
	if len(d.IPAddresses) != 2 {
		t.Fatalf("IPAddresses: got %d, want 2", len(d.IPAddresses))
	}
	if d.IPAddresses[0] != "192.168.1.1" || d.IPAddresses[1] != "10.0.0.1" {
		t.Errorf("IPAddresses: got %v", d.IPAddresses)
	}
	if d.DeviceType != models.DeviceTypeServer {
		t.Errorf("DeviceType: got %q, want %q", d.DeviceType, models.DeviceTypeServer)
	}
	if d.Status != models.DeviceStatusOnline {
		t.Errorf("Status: got %q, want %q", d.Status, models.DeviceStatusOnline)
	}
	if len(d.Tags) != 3 {
		t.Fatalf("Tags: got %d, want 3", len(d.Tags))
	}
	if d.Tags[0] != "prod" || d.Tags[1] != "web" || d.Tags[2] != "critical" {
		t.Errorf("Tags: got %v", d.Tags)
	}
}

func TestCSVRowToDevice_SemicolonSeparatedIPsAndTags(t *testing.T) {
	row := []string{
		"", "host-1", "10.0.0.1;10.0.0.2;10.0.0.3", "",
		"", "", "", "", "",
		"", "",
		"", "tag-a;tag-b",
		"", "", "", "",
	}

	d, err := csvRowToDevice(row)
	if err != nil {
		t.Fatalf("csvRowToDevice: %v", err)
	}

	if len(d.IPAddresses) != 3 {
		t.Fatalf("IPAddresses: got %d, want 3", len(d.IPAddresses))
	}
	if len(d.Tags) != 2 {
		t.Fatalf("Tags: got %d, want 2", len(d.Tags))
	}
}

func TestCSVRowToDevice_TooFewColumns(t *testing.T) {
	row := []string{"id-only", "hostname"}

	_, err := csvRowToDevice(row)
	if err == nil {
		t.Fatal("expected error for too few columns")
	}
}

func TestCSVRoundTrip(t *testing.T) {
	original := models.Device{
		ID:              "rt-001",
		Hostname:        "round-trip-host",
		IPAddresses:     []string{"172.16.0.1", "172.16.0.2"},
		MACAddress:      "11:22:33:44:55:66",
		Manufacturer:    "Acme",
		DeviceType:      models.DeviceTypeRouter,
		OS:              "RouterOS",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoverySNMP,
		LastSeen:        time.Date(2026, 2, 10, 14, 0, 0, 0, time.UTC),
		FirstSeen:       time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
		Notes:           "Core router",
		Tags:            []string{"core", "network"},
		Location:        "DC-1",
		Category:        "infrastructure",
		PrimaryRole:     "core-router",
		Owner:           "netops",
	}

	row := deviceToCSVRow(original)
	parsed, err := csvRowToDevice(row)
	if err != nil {
		t.Fatalf("csvRowToDevice: %v", err)
	}

	if parsed.ID != original.ID {
		t.Errorf("ID: got %q, want %q", parsed.ID, original.ID)
	}
	if parsed.Hostname != original.Hostname {
		t.Errorf("Hostname: got %q, want %q", parsed.Hostname, original.Hostname)
	}
	if len(parsed.IPAddresses) != len(original.IPAddresses) {
		t.Fatalf("IPAddresses len: got %d, want %d", len(parsed.IPAddresses), len(original.IPAddresses))
	}
	for i := range original.IPAddresses {
		if parsed.IPAddresses[i] != original.IPAddresses[i] {
			t.Errorf("IPAddresses[%d]: got %q, want %q", i, parsed.IPAddresses[i], original.IPAddresses[i])
		}
	}
	if parsed.MACAddress != original.MACAddress {
		t.Errorf("MACAddress: got %q, want %q", parsed.MACAddress, original.MACAddress)
	}
	if parsed.DeviceType != original.DeviceType {
		t.Errorf("DeviceType: got %q, want %q", parsed.DeviceType, original.DeviceType)
	}
	if parsed.Status != original.Status {
		t.Errorf("Status: got %q, want %q", parsed.Status, original.Status)
	}
	if parsed.DiscoveryMethod != original.DiscoveryMethod {
		t.Errorf("DiscoveryMethod: got %q, want %q", parsed.DiscoveryMethod, original.DiscoveryMethod)
	}
	if !parsed.LastSeen.Equal(original.LastSeen) {
		t.Errorf("LastSeen: got %v, want %v", parsed.LastSeen, original.LastSeen)
	}
	if !parsed.FirstSeen.Equal(original.FirstSeen) {
		t.Errorf("FirstSeen: got %v, want %v", parsed.FirstSeen, original.FirstSeen)
	}
	if parsed.Notes != original.Notes {
		t.Errorf("Notes: got %q, want %q", parsed.Notes, original.Notes)
	}
	if len(parsed.Tags) != len(original.Tags) {
		t.Fatalf("Tags len: got %d, want %d", len(parsed.Tags), len(original.Tags))
	}
	for i := range original.Tags {
		if parsed.Tags[i] != original.Tags[i] {
			t.Errorf("Tags[%d]: got %q, want %q", i, parsed.Tags[i], original.Tags[i])
		}
	}
	if parsed.Location != original.Location {
		t.Errorf("Location: got %q, want %q", parsed.Location, original.Location)
	}
	if parsed.Category != original.Category {
		t.Errorf("Category: got %q, want %q", parsed.Category, original.Category)
	}
	if parsed.PrimaryRole != original.PrimaryRole {
		t.Errorf("PrimaryRole: got %q, want %q", parsed.PrimaryRole, original.PrimaryRole)
	}
	if parsed.Owner != original.Owner {
		t.Errorf("Owner: got %q, want %q", parsed.Owner, original.Owner)
	}
}
