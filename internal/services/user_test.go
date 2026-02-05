package services_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/HerbHall/subnetree/internal/services"
	"github.com/HerbHall/subnetree/internal/testutil"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/google/uuid"
)

// authMigrations creates the auth_users table needed by user repository tests.
// This mirrors the auth module's migrations (versions 1-3).
var authMigrations = []plugin.Migration{
	{
		Version:     1,
		Description: "create auth_users table",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE auth_users (
					id            TEXT PRIMARY KEY,
					username      TEXT NOT NULL UNIQUE,
					email         TEXT NOT NULL UNIQUE,
					password_hash TEXT,
					role          TEXT NOT NULL DEFAULT 'viewer',
					auth_provider TEXT NOT NULL DEFAULT 'local',
					oidc_subject  TEXT,
					created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					last_login    DATETIME,
					disabled      INTEGER NOT NULL DEFAULT 0
				)`)
			return err
		},
	},
	{
		Version:     2,
		Description: "create auth_refresh_tokens table",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE auth_refresh_tokens (
					id         TEXT PRIMARY KEY,
					user_id    TEXT NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
					token_hash TEXT NOT NULL UNIQUE,
					expires_at DATETIME NOT NULL,
					created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					revoked    INTEGER NOT NULL DEFAULT 0
				)`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE INDEX idx_refresh_tokens_user ON auth_refresh_tokens(user_id)`)
			return err
		},
	},
	{
		Version:     3,
		Description: "add account lockout columns",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`ALTER TABLE auth_users ADD COLUMN failed_login_attempts INTEGER NOT NULL DEFAULT 0`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`ALTER TABLE auth_users ADD COLUMN locked_until DATETIME`)
			return err
		},
	},
}

func newUserRepo(t *testing.T) services.UserRepository {
	t.Helper()
	store := testutil.NewStore(t)
	if err := store.Migrate(context.Background(), "auth", authMigrations); err != nil {
		t.Fatalf("auth migrations: %v", err)
	}
	return services.NewSQLiteUserRepository(store.DB())
}

func makeUser(username, email, role string) *services.User {
	return &services.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: "$2a$10$fakehash",
		Role:         role,
		AuthProvider: "local",
	}
}

func TestSQLiteUserRepository_CreateAndGet(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	u := makeUser("admin", "admin@example.com", "admin")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, u.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Username != "admin" {
		t.Errorf("Username = %q, want %q", got.Username, "admin")
	}
	if got.Email != "admin@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "admin@example.com")
	}
	if got.Role != "admin" {
		t.Errorf("Role = %q, want %q", got.Role, "admin")
	}
	if got.PasswordHash != "$2a$10$fakehash" {
		t.Errorf("PasswordHash not stored correctly")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestSQLiteUserRepository_CreateGeneratesID(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	u := &services.User{
		Username: "newuser",
		Email:    "new@example.com",
		Role:     "viewer",
	}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.ID == "" {
		t.Error("Create did not generate an ID")
	}
}

func TestSQLiteUserRepository_GetNotFound(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent-id")
	if err != services.ErrNotFound {
		t.Errorf("Get nonexistent = %v, want ErrNotFound", err)
	}
}

func TestSQLiteUserRepository_GetByUsername(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	u := makeUser("findme", "findme@example.com", "operator")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByUsername(ctx, "findme")
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("ID = %q, want %q", got.ID, u.ID)
	}
}

func TestSQLiteUserRepository_GetByUsernameNotFound(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	_, err := repo.GetByUsername(ctx, "nonexistent")
	if err != services.ErrNotFound {
		t.Errorf("GetByUsername nonexistent = %v, want ErrNotFound", err)
	}
}

func TestSQLiteUserRepository_List(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	// Empty initially.
	users, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("List empty = %d, want 0", len(users))
	}

	// Create users.
	for _, name := range []string{"alice", "bob", "charlie"} {
		u := makeUser(name, name+"@example.com", "viewer")
		if err := repo.Create(ctx, u); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	users, err = repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("List = %d, want 3", len(users))
	}
}

func TestSQLiteUserRepository_Update(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	u := makeUser("updatable", "old@example.com", "viewer")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	u.Email = "new@example.com"
	u.Role = "admin"
	u.Disabled = true

	if err := repo.Update(ctx, u); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.Get(ctx, u.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Email != "new@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "new@example.com")
	}
	if got.Role != "admin" {
		t.Errorf("Role = %q, want %q", got.Role, "admin")
	}
	if !got.Disabled {
		t.Error("Disabled = false, want true")
	}
}

func TestSQLiteUserRepository_UpdateNotFound(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	u := &services.User{ID: "nonexistent-id", Email: "x@y.com", Role: "viewer"}
	err := repo.Update(ctx, u)
	if err != services.ErrNotFound {
		t.Errorf("Update nonexistent = %v, want ErrNotFound", err)
	}
}

func TestSQLiteUserRepository_UpdatePassword(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	u := makeUser("pwduser", "pwd@example.com", "viewer")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	newHash := "$2a$10$newhash"
	if err := repo.UpdatePassword(ctx, u.ID, newHash); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}

	got, err := repo.Get(ctx, u.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.PasswordHash != newHash {
		t.Errorf("PasswordHash = %q, want %q", got.PasswordHash, newHash)
	}
}

func TestSQLiteUserRepository_UpdatePasswordNotFound(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	err := repo.UpdatePassword(ctx, "nonexistent-id", "hash")
	if err != services.ErrNotFound {
		t.Errorf("UpdatePassword nonexistent = %v, want ErrNotFound", err)
	}
}

func TestSQLiteUserRepository_Delete(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	u := makeUser("deleteme", "del@example.com", "viewer")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Delete(ctx, u.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.Get(ctx, u.ID)
	if err != services.ErrNotFound {
		t.Errorf("Get after delete = %v, want ErrNotFound", err)
	}
}

func TestSQLiteUserRepository_DeleteNotFound(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent-id")
	if err != services.ErrNotFound {
		t.Errorf("Delete nonexistent = %v, want ErrNotFound", err)
	}
}

func TestSQLiteUserRepository_Count(t *testing.T) {
	repo := newUserRepo(t)
	ctx := context.Background()

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count empty: %v", err)
	}
	if count != 0 {
		t.Errorf("Count empty = %d, want 0", count)
	}

	for _, name := range []string{"a", "b", "c"} {
		u := makeUser(name, name+"@example.com", "viewer")
		if err := repo.Create(ctx, u); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 3 {
		t.Errorf("Count = %d, want 3", count)
	}
}
