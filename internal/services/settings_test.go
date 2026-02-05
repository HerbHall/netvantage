package services_test

import (
	"context"
	"testing"

	"github.com/HerbHall/subnetree/internal/services"
	"github.com/HerbHall/subnetree/internal/testutil"
)

func newSettingsRepo(t *testing.T) services.SettingsRepository {
	t.Helper()
	store := testutil.NewStore(t)
	repo, err := services.NewSQLiteSettingsRepository(context.Background(), store)
	if err != nil {
		t.Fatalf("NewSQLiteSettingsRepository: %v", err)
	}
	return repo
}

func TestSQLiteSettingsRepository_SetAndGet(t *testing.T) {
	repo := newSettingsRepo(t)
	ctx := context.Background()

	if err := repo.Set(ctx, "site_name", "SubNetree"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	s, err := repo.Get(ctx, "site_name")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.Key != "site_name" {
		t.Errorf("Key = %q, want %q", s.Key, "site_name")
	}
	if s.Value != "SubNetree" {
		t.Errorf("Value = %q, want %q", s.Value, "SubNetree")
	}
	if s.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}
}

func TestSQLiteSettingsRepository_SetOverwrite(t *testing.T) {
	repo := newSettingsRepo(t)
	ctx := context.Background()

	if err := repo.Set(ctx, "theme", "light"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := repo.Set(ctx, "theme", "dark"); err != nil {
		t.Fatalf("Set overwrite: %v", err)
	}

	s, err := repo.Get(ctx, "theme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.Value != "dark" {
		t.Errorf("Value = %q, want %q", s.Value, "dark")
	}
}

func TestSQLiteSettingsRepository_GetNotFound(t *testing.T) {
	repo := newSettingsRepo(t)
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent")
	if err != services.ErrNotFound {
		t.Errorf("Get nonexistent = %v, want ErrNotFound", err)
	}
}

func TestSQLiteSettingsRepository_GetAll(t *testing.T) {
	repo := newSettingsRepo(t)
	ctx := context.Background()

	// Empty initially.
	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll empty: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("GetAll empty = %d items, want 0", len(all))
	}

	// Add some settings.
	for _, kv := range []struct{ k, v string }{
		{"alpha", "1"},
		{"beta", "2"},
		{"gamma", "3"},
	} {
		if err := repo.Set(ctx, kv.k, kv.v); err != nil {
			t.Fatalf("Set %s: %v", kv.k, err)
		}
	}

	all, err = repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("GetAll = %d items, want 3", len(all))
	}
	// Results are ordered by key.
	if all[0].Key != "alpha" || all[1].Key != "beta" || all[2].Key != "gamma" {
		t.Errorf("GetAll order = [%s, %s, %s], want [alpha, beta, gamma]",
			all[0].Key, all[1].Key, all[2].Key)
	}
}

func TestSQLiteSettingsRepository_Delete(t *testing.T) {
	repo := newSettingsRepo(t)
	ctx := context.Background()

	if err := repo.Set(ctx, "to_delete", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := repo.Delete(ctx, "to_delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.Get(ctx, "to_delete")
	if err != services.ErrNotFound {
		t.Errorf("Get after delete = %v, want ErrNotFound", err)
	}
}

func TestSQLiteSettingsRepository_DeleteNotFound(t *testing.T) {
	repo := newSettingsRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	if err != services.ErrNotFound {
		t.Errorf("Delete nonexistent = %v, want ErrNotFound", err)
	}
}
