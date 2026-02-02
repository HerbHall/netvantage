package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/HerbHall/netvantage/pkg/models"
	"github.com/google/uuid"
)

// ScanRepository provides access to network scan records.
type ScanRepository interface {
	// Get returns a single scan by ID.
	Get(ctx context.Context, id string) (*models.ScanResult, error)

	// List returns a paginated list of scans ordered by start time.
	List(ctx context.Context, opts ListOptions) (*ListResult[models.ScanResult], error)

	// Create inserts a new scan record. If scan.ID is empty, a UUID is generated.
	Create(ctx context.Context, scan *models.ScanResult) error

	// UpdateStatus updates a scan's status and optional end time.
	UpdateStatus(ctx context.Context, id, status string, endedAt *string) error
}

// Compile-time interface guard.
var _ ScanRepository = (*SQLiteScanRepository)(nil)

// SQLiteScanRepository implements ScanRepository using SQLite.
// It queries the recon_scans table directly.
type SQLiteScanRepository struct {
	db *sql.DB
}

// NewSQLiteScanRepository creates a ScanRepository.
// The recon_scans table must already exist (created by the recon module's migrations).
func NewSQLiteScanRepository(db *sql.DB) *SQLiteScanRepository {
	return &SQLiteScanRepository{db: db}
}

func (r *SQLiteScanRepository) Get(ctx context.Context, id string) (*models.ScanResult, error) {
	var scan models.ScanResult
	var endedAt sql.NullString
	var errorMsg string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, subnet, started_at, ended_at, status, total, online, error_msg
		FROM recon_scans WHERE id = ?`, id,
	).Scan(&scan.ID, &scan.Subnet, &scan.StartedAt, &endedAt, &scan.Status,
		&scan.Total, &scan.Online, &errorMsg)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get scan %q: %w", id, err)
	}
	if endedAt.Valid {
		scan.EndedAt = endedAt.String
	}
	return &scan, nil
}

func (r *SQLiteScanRepository) List(ctx context.Context, opts ListOptions) (*ListResult[models.ScanResult], error) {
	opts = normalizeListOptions(opts)

	// Count total scans.
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM recon_scans`,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count scans: %w", err)
	}

	// Query with pagination. Scans are always ordered by started_at.
	orderDir := "DESC"
	if opts.SortOrder == "asc" {
		orderDir = "ASC"
	}

	//nolint:gosec // orderDir is validated above
	query := fmt.Sprintf(
		`SELECT id, subnet, started_at, ended_at, status, total, online, error_msg
		FROM recon_scans ORDER BY started_at %s LIMIT ? OFFSET ?`, orderDir)

	rows, err := r.db.QueryContext(ctx, query, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("list scans: %w", err)
	}
	defer rows.Close()

	var scans []models.ScanResult
	for rows.Next() {
		var scan models.ScanResult
		var endedAt sql.NullString
		var errorMsg string
		if err := rows.Scan(&scan.ID, &scan.Subnet, &scan.StartedAt, &endedAt,
			&scan.Status, &scan.Total, &scan.Online, &errorMsg); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		if endedAt.Valid {
			scan.EndedAt = endedAt.String
		}
		scans = append(scans, scan)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scans: %w", err)
	}
	if scans == nil {
		scans = []models.ScanResult{}
	}

	return &ListResult[models.ScanResult]{Items: scans, Total: total}, nil
}

func (r *SQLiteScanRepository) Create(ctx context.Context, scan *models.ScanResult) error {
	if scan.ID == "" {
		scan.ID = uuid.New().String()
	}
	if scan.StartedAt == "" {
		scan.StartedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if scan.Status == "" {
		scan.Status = "running"
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO recon_scans (id, subnet, started_at, status)
		VALUES (?, ?, ?, ?)`,
		scan.ID, scan.Subnet, scan.StartedAt, scan.Status,
	)
	if err != nil {
		return fmt.Errorf("create scan: %w", err)
	}
	return nil
}

func (r *SQLiteScanRepository) UpdateStatus(ctx context.Context, id, status string, endedAt *string) error {
	var err error
	if endedAt != nil {
		_, err = r.db.ExecContext(ctx,
			`UPDATE recon_scans SET status = ?, ended_at = ? WHERE id = ?`,
			status, *endedAt, id)
	} else {
		_, err = r.db.ExecContext(ctx,
			`UPDATE recon_scans SET status = ? WHERE id = ?`,
			status, id)
	}
	if err != nil {
		return fmt.Errorf("update scan status: %w", err)
	}
	return nil
}
