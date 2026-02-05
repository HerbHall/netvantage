package store

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/HerbHall/subnetree/pkg/plugin"
	_ "modernc.org/sqlite" // Pure-Go SQLite driver
)

// Compile-time interface guard.
var _ plugin.Store = (*SQLiteStore)(nil)

// SQLiteStore implements plugin.Store backed by SQLite via modernc.org/sqlite.
type SQLiteStore struct {
	db   *sql.DB
	mu   sync.Mutex // Serialize migrations
	once sync.Once  // Ensure _migrations table created once
}

// New opens (or creates) a SQLite database at the given path and applies
// recommended pragmas for WAL mode, foreign keys, and performance.
// Returns the concrete type; callers assign to plugin.Store where needed.
func New(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}

	// SQLite performs best with a single write connection. WAL enables concurrent readers.
	db.SetMaxOpenConns(1)

	// Verify the connection works.
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite %q: %w", path, err)
	}

	// Apply recommended pragmas (modernc.org/sqlite requires SQL statements, not DSN params).
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA cache_size=-20000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("exec %q: %w", p, err)
		}
	}

	return &SQLiteStore{db: db}, nil
}

// DB returns the underlying *sql.DB for direct queries.
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// Tx executes fn within a database transaction. The transaction is
// committed if fn returns nil, rolled back otherwise.
func (s *SQLiteStore) Tx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original: %w)", rbErr, err)
		}
		return err
	}

	return tx.Commit()
}

// Migrate runs pending migrations for the named plugin. Already-applied
// migrations (tracked in the shared _migrations table) are skipped.
// Migrations must be provided in ascending Version order.
func (s *SQLiteStore) Migrate(ctx context.Context, pluginName string, migrations []plugin.Migration) error {
	if err := s.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range migrations {
		applied, err := s.isMigrationApplied(ctx, pluginName, m.Version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := s.applyMigration(ctx, pluginName, m); err != nil {
			return fmt.Errorf("migration %s/%d (%s): %w", pluginName, m.Version, m.Description, err)
		}
	}

	return nil
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// ensureMigrationsTable creates the shared _migrations tracking table if it
// doesn't already exist. Safe to call multiple times (uses sync.Once).
func (s *SQLiteStore) ensureMigrationsTable(ctx context.Context) error {
	var err error
	s.once.Do(func() {
		_, err = s.db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS _migrations (
				plugin_name TEXT    NOT NULL,
				version     INTEGER NOT NULL,
				description TEXT    NOT NULL,
				applied_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (plugin_name, version)
			)
		`)
	})
	return err
}

func (s *SQLiteStore) isMigrationApplied(ctx context.Context, pluginName string, version int) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM _migrations WHERE plugin_name = ? AND version = ?",
		pluginName, version,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check migration %s/%d: %w", pluginName, version, err)
	}
	return count > 0, nil
}

func (s *SQLiteStore) applyMigration(ctx context.Context, pluginName string, m plugin.Migration) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		if err := m.Up(tx); err != nil {
			return err
		}

		_, err := tx.ExecContext(ctx,
			"INSERT INTO _migrations (plugin_name, version, description) VALUES (?, ?, ?)",
			pluginName, m.Version, m.Description,
		)
		return err
	})
}
