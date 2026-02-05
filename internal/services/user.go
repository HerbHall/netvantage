package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// User represents a user account for the services layer.
// This mirrors the auth_users table but is independent of the internal/auth package.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never serialized to JSON.
	Role         string    `json:"role"`
	AuthProvider string    `json:"auth_provider"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login,omitempty"`
	Disabled     bool      `json:"disabled"`
}

// UserRepository provides access to user accounts.
type UserRepository interface {
	// Get returns a single user by ID.
	Get(ctx context.Context, id string) (*User, error)

	// GetByUsername returns a user by username.
	GetByUsername(ctx context.Context, username string) (*User, error)

	// List returns all users ordered by creation time.
	List(ctx context.Context) ([]User, error)

	// Create inserts a new user. If user.ID is empty, a UUID is generated.
	Create(ctx context.Context, user *User) error

	// Update modifies a user's email, role, and disabled status.
	Update(ctx context.Context, user *User) error

	// UpdatePassword updates a user's password hash.
	UpdatePassword(ctx context.Context, id, passwordHash string) error

	// Delete removes a user by ID.
	Delete(ctx context.Context, id string) error

	// Count returns the total number of users.
	Count(ctx context.Context) (int, error)
}

// Compile-time interface guard.
var _ UserRepository = (*SQLiteUserRepository)(nil)

// SQLiteUserRepository implements UserRepository using SQLite.
// It queries the auth_users table directly.
type SQLiteUserRepository struct {
	db *sql.DB
}

// NewSQLiteUserRepository creates a UserRepository.
// The auth_users table must already exist (created by auth module initialization).
func NewSQLiteUserRepository(db *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
}

// userColumns is the shared SELECT column list for user queries.
const userColumns = `id, username, email, password_hash, role, auth_provider,
	created_at, last_login, disabled`

func (r *SQLiteUserRepository) Get(ctx context.Context, id string) (*User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM auth_users WHERE id = ?`, id)
	u, err := scanServiceUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user %q: %w", id, err)
	}
	return u, nil
}

func (r *SQLiteUserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM auth_users WHERE username = ?`, username)
	u, err := scanServiceUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by username %q: %w", username, err)
	}
	return u, nil
}

func (r *SQLiteUserRepository) List(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+userColumns+` FROM auth_users ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := scanServiceUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (r *SQLiteUserRepository) Create(ctx context.Context, user *User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
	if user.AuthProvider == "" {
		user.AuthProvider = "local"
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO auth_users (id, username, email, password_hash, role, auth_provider, created_at, disabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.Email, user.PasswordHash,
		user.Role, user.AuthProvider, user.CreatedAt, user.Disabled,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *SQLiteUserRepository) Update(ctx context.Context, user *User) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE auth_users SET email = ?, role = ?, disabled = ? WHERE id = ?`,
		user.Email, user.Role, user.Disabled, user.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLiteUserRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE auth_users SET password_hash = ? WHERE id = ?`,
		passwordHash, id,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLiteUserRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM auth_users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLiteUserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM auth_users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

// scanServiceUser scans a single *sql.Row into a User.
func scanServiceUser(row *sql.Row) (*User, error) {
	var u User
	var passwordHash sql.NullString
	var lastLogin sql.NullTime

	err := row.Scan(&u.ID, &u.Username, &u.Email, &passwordHash, &u.Role,
		&u.AuthProvider, &u.CreatedAt, &lastLogin, &u.Disabled)
	if err != nil {
		return nil, err
	}
	if passwordHash.Valid {
		u.PasswordHash = passwordHash.String
	}
	if lastLogin.Valid {
		u.LastLogin = lastLogin.Time
	}
	return &u, nil
}

// scanServiceUserRow scans a *sql.Rows row into a User.
func scanServiceUserRow(rows *sql.Rows) (*User, error) {
	var u User
	var passwordHash sql.NullString
	var lastLogin sql.NullTime

	err := rows.Scan(&u.ID, &u.Username, &u.Email, &passwordHash, &u.Role,
		&u.AuthProvider, &u.CreatedAt, &lastLogin, &u.Disabled)
	if err != nil {
		return nil, err
	}
	if passwordHash.Valid {
		u.PasswordHash = passwordHash.String
	}
	if lastLogin.Valid {
		u.LastLogin = lastLogin.Time
	}
	return &u, nil
}
