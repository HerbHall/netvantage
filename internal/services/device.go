package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/google/uuid"
)

// DeviceFilter controls which devices are returned by List.
type DeviceFilter struct {
	Status     string // Filter by DeviceStatus value.
	DeviceType string // Filter by DeviceType value.
	Search     string // Search hostname, IP addresses, or MAC address.
	ScanID     string // Filter to devices linked to a specific scan.
}

// DeviceRepository provides CRUD access to network devices.
type DeviceRepository interface {
	// Get returns a single device by ID.
	Get(ctx context.Context, id string) (*models.Device, error)

	// List returns a filtered, paginated list of devices.
	List(ctx context.Context, filter DeviceFilter, opts ListOptions) (*ListResult[models.Device], error)

	// Create inserts a new device. If device.ID is empty, a UUID is generated.
	Create(ctx context.Context, device *models.Device) error

	// Update modifies an existing device's mutable fields.
	Update(ctx context.Context, device *models.Device) error

	// Delete removes a device by ID.
	Delete(ctx context.Context, id string) error
}

// Compile-time interface guard.
var _ DeviceRepository = (*SQLiteDeviceRepository)(nil)

// SQLiteDeviceRepository implements DeviceRepository using SQLite.
// It queries the recon_devices table directly.
type SQLiteDeviceRepository struct {
	db *sql.DB
}

// NewSQLiteDeviceRepository creates a DeviceRepository.
// The recon_devices table must already exist (created by the recon module's migrations).
func NewSQLiteDeviceRepository(db *sql.DB) *SQLiteDeviceRepository {
	return &SQLiteDeviceRepository{db: db}
}

// deviceColumns is the shared column list for device queries.
const deviceColumns = `id, hostname, ip_addresses, mac_address, manufacturer,
	device_type, os, status, discovery_method, agent_id,
	first_seen, last_seen, notes, tags, custom_fields`

func (r *SQLiteDeviceRepository) Get(ctx context.Context, id string) (*models.Device, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+deviceColumns+` FROM recon_devices WHERE id = ?`, id)
	d, err := scanDevice(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get device %q: %w", id, err)
	}
	return d, nil
}

func (r *SQLiteDeviceRepository) List(ctx context.Context, filter DeviceFilter, opts ListOptions) (*ListResult[models.Device], error) {
	opts = normalizeListOptions(opts)

	// Validate sortBy against allowed columns.
	sortCol := "last_seen"
	allowedSorts := map[string]string{
		"hostname":  "hostname",
		"status":    "status",
		"last_seen": "last_seen",
		"first_seen": "first_seen",
		"device_type": "device_type",
	}
	if opts.SortBy != "" {
		if col, ok := allowedSorts[opts.SortBy]; ok {
			sortCol = col
		}
	}

	// Build WHERE clause with parameterized placeholders.
	where := "1=1"
	var args []any

	if filter.Status != "" {
		where += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.DeviceType != "" {
		where += " AND device_type = ?"
		args = append(args, filter.DeviceType)
	}
	if filter.Search != "" {
		where += " AND (hostname LIKE ? OR ip_addresses LIKE ? OR mac_address LIKE ?)"
		pattern := "%" + filter.Search + "%"
		args = append(args, pattern, pattern, pattern)
	}
	if filter.ScanID != "" {
		where += " AND id IN (SELECT device_id FROM recon_scan_devices WHERE scan_id = ?)"
		args = append(args, filter.ScanID)
	}

	// Count total matching rows.
	var total int
	//nolint:gosec // where uses parameterized placeholders only
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM recon_devices WHERE "+where, args...,
	).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count devices: %w", err)
	}

	// Query with pagination and sorting.
	queryArgs := make([]any, 0, len(args)+2)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, opts.Limit, opts.Offset)

	orderDir := "DESC"
	if opts.SortOrder == "asc" {
		orderDir = "ASC"
	}

	//nolint:gosec // where and sortCol are validated above, not user input
	query := fmt.Sprintf(
		"SELECT %s FROM recon_devices WHERE %s ORDER BY %s %s LIMIT ? OFFSET ?",
		deviceColumns, where, sortCol, orderDir,
	)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		d, err := scanDeviceRow(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate devices: %w", err)
	}
	if devices == nil {
		devices = []models.Device{}
	}

	return &ListResult[models.Device]{Items: devices, Total: total}, nil
}

func (r *SQLiteDeviceRepository) Create(ctx context.Context, device *models.Device) error {
	if device.ID == "" {
		device.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if device.FirstSeen.IsZero() {
		device.FirstSeen = now
	}
	if device.LastSeen.IsZero() {
		device.LastSeen = now
	}

	ipsJSON, _ := json.Marshal(device.IPAddresses)
	if device.IPAddresses == nil {
		ipsJSON = []byte("[]")
	}
	tagsJSON, _ := json.Marshal(device.Tags)
	if device.Tags == nil {
		tagsJSON = []byte("[]")
	}
	cfJSON, _ := json.Marshal(device.CustomFields)
	if device.CustomFields == nil {
		cfJSON = []byte("{}")
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO recon_devices (
			id, hostname, ip_addresses, mac_address, manufacturer,
			device_type, os, status, discovery_method, agent_id,
			first_seen, last_seen, notes, tags, custom_fields
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		device.ID, device.Hostname, string(ipsJSON), device.MACAddress, device.Manufacturer,
		string(device.DeviceType), device.OS, string(device.Status), string(device.DiscoveryMethod), device.AgentID,
		device.FirstSeen, device.LastSeen, device.Notes, string(tagsJSON), string(cfJSON),
	)
	if err != nil {
		return fmt.Errorf("create device: %w", err)
	}
	return nil
}

func (r *SQLiteDeviceRepository) Update(ctx context.Context, device *models.Device) error {
	ipsJSON, _ := json.Marshal(device.IPAddresses)
	if device.IPAddresses == nil {
		ipsJSON = []byte("[]")
	}
	tagsJSON, _ := json.Marshal(device.Tags)
	if device.Tags == nil {
		tagsJSON = []byte("[]")
	}
	cfJSON, _ := json.Marshal(device.CustomFields)
	if device.CustomFields == nil {
		cfJSON = []byte("{}")
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE recon_devices SET
			hostname = ?, ip_addresses = ?, mac_address = ?, manufacturer = ?,
			device_type = ?, os = ?, status = ?, discovery_method = ?, agent_id = ?,
			last_seen = ?, notes = ?, tags = ?, custom_fields = ?
		WHERE id = ?`,
		device.Hostname, string(ipsJSON), device.MACAddress, device.Manufacturer,
		string(device.DeviceType), device.OS, string(device.Status), string(device.DiscoveryMethod), device.AgentID,
		device.LastSeen, device.Notes, string(tagsJSON), string(cfJSON),
		device.ID,
	)
	if err != nil {
		return fmt.Errorf("update device: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLiteDeviceRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM recon_devices WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// scanDevice scans a single *sql.Row into a Device.
func scanDevice(row *sql.Row) (*models.Device, error) {
	var d models.Device
	var ipsJSON, tagsJSON, cfJSON string
	var dt, status, method string
	err := row.Scan(
		&d.ID, &d.Hostname, &ipsJSON, &d.MACAddress, &d.Manufacturer,
		&dt, &d.OS, &status, &method, &d.AgentID,
		&d.FirstSeen, &d.LastSeen, &d.Notes, &tagsJSON, &cfJSON,
	)
	if err != nil {
		return nil, err
	}
	d.DeviceType = models.DeviceType(dt)
	d.Status = models.DeviceStatus(status)
	d.DiscoveryMethod = models.DiscoveryMethod(method)
	_ = json.Unmarshal([]byte(ipsJSON), &d.IPAddresses)
	_ = json.Unmarshal([]byte(tagsJSON), &d.Tags)
	_ = json.Unmarshal([]byte(cfJSON), &d.CustomFields)
	return &d, nil
}

// scanDeviceRow scans a *sql.Rows row into a Device.
func scanDeviceRow(rows *sql.Rows) (*models.Device, error) {
	var d models.Device
	var ipsJSON, tagsJSON, cfJSON string
	var dt, status, method string
	err := rows.Scan(
		&d.ID, &d.Hostname, &ipsJSON, &d.MACAddress, &d.Manufacturer,
		&dt, &d.OS, &status, &method, &d.AgentID,
		&d.FirstSeen, &d.LastSeen, &d.Notes, &tagsJSON, &cfJSON,
	)
	if err != nil {
		return nil, err
	}
	d.DeviceType = models.DeviceType(dt)
	d.Status = models.DeviceStatus(status)
	d.DiscoveryMethod = models.DiscoveryMethod(method)
	_ = json.Unmarshal([]byte(ipsJSON), &d.IPAddresses)
	_ = json.Unmarshal([]byte(tagsJSON), &d.Tags)
	_ = json.Unmarshal([]byte(cfJSON), &d.CustomFields)
	return &d, nil
}
