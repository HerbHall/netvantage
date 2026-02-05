package services_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/services"
	"github.com/HerbHall/subnetree/internal/testutil"
	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/google/uuid"
)

func newScanRepo(t *testing.T) (services.ScanRepository, *sql.DB) {
	t.Helper()
	store := testutil.NewStore(t)
	if err := store.Migrate(context.Background(), "recon", reconMigrations); err != nil {
		t.Fatalf("recon migrations: %v", err)
	}
	return services.NewSQLiteScanRepository(store.DB()), store.DB()
}

func TestSQLiteScanRepository_CreateAndGet(t *testing.T) {
	repo, _ := newScanRepo(t)
	ctx := context.Background()

	scan := &models.ScanResult{
		ID:     uuid.New().String(),
		Subnet: "192.168.1.0/24",
		Status: "running",
	}

	if err := repo.Create(ctx, scan); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if scan.StartedAt == "" {
		t.Error("StartedAt not set by Create")
	}

	got, err := repo.Get(ctx, scan.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Subnet != "192.168.1.0/24" {
		t.Errorf("Subnet = %q, want %q", got.Subnet, "192.168.1.0/24")
	}
	if got.Status != "running" {
		t.Errorf("Status = %q, want %q", got.Status, "running")
	}
}

func TestSQLiteScanRepository_CreateGeneratesID(t *testing.T) {
	repo, _ := newScanRepo(t)
	ctx := context.Background()

	scan := &models.ScanResult{
		Subnet: "10.0.0.0/16",
	}
	if err := repo.Create(ctx, scan); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if scan.ID == "" {
		t.Error("Create did not generate an ID")
	}
	if scan.Status != "running" {
		t.Errorf("Status = %q, want %q", scan.Status, "running")
	}
}

func TestSQLiteScanRepository_GetNotFound(t *testing.T) {
	repo, _ := newScanRepo(t)
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent-id")
	if err != services.ErrNotFound {
		t.Errorf("Get nonexistent = %v, want ErrNotFound", err)
	}
}

func TestSQLiteScanRepository_UpdateStatus(t *testing.T) {
	repo, _ := newScanRepo(t)
	ctx := context.Background()

	scan := &models.ScanResult{
		Subnet: "172.16.0.0/12",
	}
	if err := repo.Create(ctx, scan); err != nil {
		t.Fatalf("Create: %v", err)
	}

	endedAt := time.Now().UTC().Format(time.RFC3339)
	if err := repo.UpdateStatus(ctx, scan.ID, "completed", &endedAt); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, err := repo.Get(ctx, scan.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("Status = %q, want %q", got.Status, "completed")
	}
	if got.EndedAt == "" {
		t.Error("EndedAt is empty after UpdateStatus")
	}
}

func TestSQLiteScanRepository_UpdateStatusWithoutEndedAt(t *testing.T) {
	repo, _ := newScanRepo(t)
	ctx := context.Background()

	scan := &models.ScanResult{
		Subnet: "192.168.0.0/24",
	}
	if err := repo.Create(ctx, scan); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdateStatus(ctx, scan.ID, "failed", nil); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, err := repo.Get(ctx, scan.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != "failed" {
		t.Errorf("Status = %q, want %q", got.Status, "failed")
	}
}

func TestSQLiteScanRepository_ListPagination(t *testing.T) {
	repo, _ := newScanRepo(t)
	ctx := context.Background()

	// Create 5 scans.
	for i := 0; i < 5; i++ {
		scan := &models.ScanResult{
			Subnet:    "10.0.0.0/24",
			StartedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
		}
		if err := repo.Create(ctx, scan); err != nil {
			t.Fatalf("Create scan %d: %v", i, err)
		}
	}

	// Page 1.
	result, err := repo.List(ctx, services.ListOptions{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("Total = %d, want 5", result.Total)
	}
	if len(result.Items) != 2 {
		t.Errorf("Page 1 items = %d, want 2", len(result.Items))
	}

	// Beyond end.
	result, err = repo.List(ctx, services.ListOptions{Limit: 10, Offset: 5})
	if err != nil {
		t.Fatalf("List beyond end: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("Beyond end items = %d, want 0", len(result.Items))
	}
}

func TestSQLiteScanRepository_ListEmpty(t *testing.T) {
	repo, _ := newScanRepo(t)
	ctx := context.Background()

	result, err := repo.List(ctx, services.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
	if result.Items == nil {
		t.Error("Items is nil, want empty slice")
	}
}
