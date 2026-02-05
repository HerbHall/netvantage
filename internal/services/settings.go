package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

// Setting represents a key-value configuration entry.
type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SettingsRepository provides access to application settings.
type SettingsRepository interface {
	// Get returns a single setting by key.
	Get(ctx context.Context, key string) (*Setting, error)

	// GetAll returns all settings.
	GetAll(ctx context.Context) ([]Setting, error)

	// Set creates or updates a setting.
	Set(ctx context.Context, key, value string) error

	// Delete removes a setting by key.
	Delete(ctx context.Context, key string) error
}

// Compile-time interface guard.
var _ SettingsRepository = (*SQLiteSettingsRepository)(nil)

// SQLiteSettingsRepository implements SettingsRepository using SQLite.
type SQLiteSettingsRepository struct {
	db *sql.DB
}

// NewSQLiteSettingsRepository creates a SettingsRepository and runs the
// core_settings migration.
func NewSQLiteSettingsRepository(ctx context.Context, store plugin.Store) (*SQLiteSettingsRepository, error) {
	if err := store.Migrate(ctx, "core", settingsMigrations); err != nil {
		return nil, fmt.Errorf("core settings migrations: %w", err)
	}
	return &SQLiteSettingsRepository{db: store.DB()}, nil
}

func (r *SQLiteSettingsRepository) Get(ctx context.Context, key string) (*Setting, error) {
	var s Setting
	err := r.db.QueryRowContext(ctx,
		`SELECT key, value, updated_at FROM core_settings WHERE key = ?`, key,
	).Scan(&s.Key, &s.Value, &s.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get setting %q: %w", key, err)
	}
	return &s, nil
}

func (r *SQLiteSettingsRepository) GetAll(ctx context.Context) ([]Setting, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT key, value, updated_at FROM core_settings ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	defer rows.Close()

	var settings []Setting
	for rows.Next() {
		var s Setting
		if err := rows.Scan(&s.Key, &s.Value, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan setting row: %w", err)
		}
		settings = append(settings, s)
	}
	return settings, rows.Err()
}

func (r *SQLiteSettingsRepository) Set(ctx context.Context, key, value string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO core_settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, now,
	)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}

func (r *SQLiteSettingsRepository) Delete(ctx context.Context, key string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM core_settings WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("delete setting %q: %w", key, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// settingsMigrations defines the database schema for core_settings.
var settingsMigrations = []plugin.Migration{
	{
		Version:     1,
		Description: "create core_settings table",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE core_settings (
					key        TEXT PRIMARY KEY,
					value      TEXT NOT NULL,
					updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
				)`)
			return err
		},
	},
}
