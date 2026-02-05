package services_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/services"
	"github.com/HerbHall/subnetree/internal/testutil"
	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
)

// reconMigrations creates the recon_devices and recon_scan_devices tables
// needed by the device repository tests.
var reconMigrations = []plugin.Migration{
	{
		Version:     1,
		Description: "create recon tables for testing",
		Up: func(tx *sql.Tx) error {
			stmts := []string{
				`CREATE TABLE recon_devices (
					id               TEXT PRIMARY KEY,
					hostname         TEXT NOT NULL DEFAULT '',
					ip_addresses     TEXT NOT NULL DEFAULT '[]',
					mac_address      TEXT NOT NULL DEFAULT '',
					manufacturer     TEXT NOT NULL DEFAULT '',
					device_type      TEXT NOT NULL DEFAULT 'unknown',
					os               TEXT NOT NULL DEFAULT '',
					status           TEXT NOT NULL DEFAULT 'unknown',
					discovery_method TEXT NOT NULL DEFAULT 'icmp',
					agent_id         TEXT NOT NULL DEFAULT '',
					first_seen       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					last_seen        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					notes            TEXT NOT NULL DEFAULT '',
					tags             TEXT NOT NULL DEFAULT '[]',
					custom_fields    TEXT NOT NULL DEFAULT '{}'
				)`,
				`CREATE INDEX idx_recon_devices_mac ON recon_devices(mac_address)`,
				`CREATE INDEX idx_recon_devices_status ON recon_devices(status)`,
				`CREATE INDEX idx_recon_devices_last_seen ON recon_devices(last_seen)`,
				`CREATE TABLE recon_scans (
					id         TEXT PRIMARY KEY,
					subnet     TEXT NOT NULL,
					started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					ended_at   DATETIME,
					status     TEXT NOT NULL DEFAULT 'pending',
					total      INTEGER NOT NULL DEFAULT 0,
					online     INTEGER NOT NULL DEFAULT 0,
					error_msg  TEXT NOT NULL DEFAULT ''
				)`,
				`CREATE INDEX idx_recon_scans_status ON recon_scans(status)`,
				`CREATE TABLE recon_scan_devices (
					scan_id   TEXT NOT NULL REFERENCES recon_scans(id) ON DELETE CASCADE,
					device_id TEXT NOT NULL REFERENCES recon_devices(id) ON DELETE CASCADE,
					PRIMARY KEY (scan_id, device_id)
				)`,
			}
			for _, stmt := range stmts {
				if _, err := tx.Exec(stmt); err != nil {
					return err
				}
			}
			return nil
		},
	},
}

func newDeviceRepo(t *testing.T) (services.DeviceRepository, *sql.DB) {
	t.Helper()
	store := testutil.NewStore(t)
	if err := store.Migrate(context.Background(), "recon", reconMigrations); err != nil {
		t.Fatalf("recon migrations: %v", err)
	}
	return services.NewSQLiteDeviceRepository(store.DB()), store.DB()
}

func TestSQLiteDeviceRepository_CreateAndGet(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	d := testutil.NewDevice(
		testutil.WithHostname("server-01"),
		testutil.WithIP("10.0.0.1"),
		testutil.WithMAC("AA:BB:CC:DD:EE:FF"),
	)

	if err := repo.Create(ctx, &d); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, d.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.ID != d.ID {
		t.Errorf("ID = %q, want %q", got.ID, d.ID)
	}
	if got.Hostname != "server-01" {
		t.Errorf("Hostname = %q, want %q", got.Hostname, "server-01")
	}
	if got.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MACAddress = %q, want %q", got.MACAddress, "AA:BB:CC:DD:EE:FF")
	}
	if len(got.IPAddresses) != 1 || got.IPAddresses[0] != "10.0.0.1" {
		t.Errorf("IPAddresses = %v, want [10.0.0.1]", got.IPAddresses)
	}
}

func TestSQLiteDeviceRepository_CreateGeneratesID(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	d := testutil.NewDevice()
	d.ID = "" // Force ID generation.

	if err := repo.Create(ctx, &d); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if d.ID == "" {
		t.Error("Create did not generate an ID")
	}
}

func TestSQLiteDeviceRepository_GetNotFound(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent-id")
	if err != services.ErrNotFound {
		t.Errorf("Get nonexistent = %v, want ErrNotFound", err)
	}
}

func TestSQLiteDeviceRepository_Update(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	d := testutil.NewDevice(testutil.WithHostname("old-name"))
	if err := repo.Create(ctx, &d); err != nil {
		t.Fatalf("Create: %v", err)
	}

	d.Hostname = "new-name"
	d.Status = models.DeviceStatusOffline
	d.LastSeen = time.Now().UTC()

	if err := repo.Update(ctx, &d); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.Get(ctx, d.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Hostname != "new-name" {
		t.Errorf("Hostname = %q, want %q", got.Hostname, "new-name")
	}
	if got.Status != models.DeviceStatusOffline {
		t.Errorf("Status = %q, want %q", got.Status, models.DeviceStatusOffline)
	}
}

func TestSQLiteDeviceRepository_UpdateNotFound(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	d := testutil.NewDevice()
	d.ID = "nonexistent-id"

	err := repo.Update(ctx, &d)
	if err != services.ErrNotFound {
		t.Errorf("Update nonexistent = %v, want ErrNotFound", err)
	}
}

func TestSQLiteDeviceRepository_Delete(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	d := testutil.NewDevice()
	if err := repo.Create(ctx, &d); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Delete(ctx, d.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.Get(ctx, d.ID)
	if err != services.ErrNotFound {
		t.Errorf("Get after delete = %v, want ErrNotFound", err)
	}
}

func TestSQLiteDeviceRepository_DeleteNotFound(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent-id")
	if err != services.ErrNotFound {
		t.Errorf("Delete nonexistent = %v, want ErrNotFound", err)
	}
}

func TestSQLiteDeviceRepository_ListPagination(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	// Create 5 devices.
	for i := 0; i < 5; i++ {
		d := testutil.NewDevice(testutil.WithHostname(
			"device-" + string(rune('A'+i)),
		))
		if err := repo.Create(ctx, &d); err != nil {
			t.Fatalf("Create device %d: %v", i, err)
		}
	}

	// Page 1: limit 2, offset 0.
	result, err := repo.List(ctx, services.DeviceFilter{}, services.ListOptions{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("Total = %d, want 5", result.Total)
	}
	if len(result.Items) != 2 {
		t.Errorf("Page 1 items = %d, want 2", len(result.Items))
	}

	// Page 3: limit 2, offset 4.
	result, err = repo.List(ctx, services.DeviceFilter{}, services.ListOptions{Limit: 2, Offset: 4})
	if err != nil {
		t.Fatalf("List page 3: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("Page 3 items = %d, want 1", len(result.Items))
	}
}

func TestSQLiteDeviceRepository_ListFilterStatus(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	// Create devices with different statuses.
	for _, s := range []models.DeviceStatus{
		models.DeviceStatusOnline,
		models.DeviceStatusOnline,
		models.DeviceStatusOffline,
	} {
		d := testutil.NewDevice(testutil.WithStatus(s))
		if err := repo.Create(ctx, &d); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	result, err := repo.List(ctx, services.DeviceFilter{Status: "online"}, services.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total online = %d, want 2", result.Total)
	}
	if len(result.Items) != 2 {
		t.Errorf("Items online = %d, want 2", len(result.Items))
	}
}

func TestSQLiteDeviceRepository_ListFilterDeviceType(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	d1 := testutil.NewDevice(testutil.WithDeviceType(models.DeviceTypeRouter))
	d2 := testutil.NewDevice(testutil.WithDeviceType(models.DeviceTypeServer))
	d3 := testutil.NewDevice(testutil.WithDeviceType(models.DeviceTypeRouter))
	for _, d := range []*models.Device{&d1, &d2, &d3} {
		if err := repo.Create(ctx, d); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	result, err := repo.List(ctx, services.DeviceFilter{DeviceType: "router"}, services.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total routers = %d, want 2", result.Total)
	}
}

func TestSQLiteDeviceRepository_ListSearch(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	d1 := testutil.NewDevice(
		testutil.WithHostname("web-server"),
		testutil.WithIP("10.0.0.1"),
	)
	d2 := testutil.NewDevice(
		testutil.WithHostname("db-server"),
		testutil.WithIP("10.0.0.2"),
	)
	d3 := testutil.NewDevice(
		testutil.WithHostname("printer"),
		testutil.WithIP("10.0.1.1"),
	)
	for _, d := range []*models.Device{&d1, &d2, &d3} {
		if err := repo.Create(ctx, d); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	// Search by hostname substring.
	result, err := repo.List(ctx, services.DeviceFilter{Search: "server"}, services.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total matching 'server' = %d, want 2", result.Total)
	}

	// Search by IP substring.
	result, err = repo.List(ctx, services.DeviceFilter{Search: "10.0.1"}, services.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total matching '10.0.1' = %d, want 1", result.Total)
	}
}

func TestSQLiteDeviceRepository_ListSortAsc(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	now := time.Now().UTC()
	d1 := testutil.NewDevice(testutil.WithHostname("alpha"))
	d1.LastSeen = now.Add(-2 * time.Hour)
	d2 := testutil.NewDevice(testutil.WithHostname("beta"))
	d2.LastSeen = now.Add(-1 * time.Hour)
	d3 := testutil.NewDevice(testutil.WithHostname("gamma"))
	d3.LastSeen = now

	for _, d := range []*models.Device{&d1, &d2, &d3} {
		if err := repo.Create(ctx, d); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	// Sort ascending by last_seen.
	result, err := repo.List(ctx, services.DeviceFilter{}, services.ListOptions{
		SortBy: "last_seen", SortOrder: "asc",
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("Items = %d, want 3", len(result.Items))
	}
	if result.Items[0].Hostname != "alpha" {
		t.Errorf("First = %q, want %q", result.Items[0].Hostname, "alpha")
	}
	if result.Items[2].Hostname != "gamma" {
		t.Errorf("Last = %q, want %q", result.Items[2].Hostname, "gamma")
	}
}

func TestSQLiteDeviceRepository_ListEmpty(t *testing.T) {
	repo, _ := newDeviceRepo(t)
	ctx := context.Background()

	result, err := repo.List(ctx, services.DeviceFilter{}, services.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
	if result.Items == nil {
		t.Error("Items is nil, want empty slice")
	}
	if len(result.Items) != 0 {
		t.Errorf("Items = %d, want 0", len(result.Items))
	}
}
