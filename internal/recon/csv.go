package recon

import (
	"fmt"
	"strings"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
)

// csvHeaders returns the CSV column headers.
func csvHeaders() []string {
	return []string{
		"id", "hostname", "ip_addresses", "mac_address", "manufacturer",
		"device_type", "os", "status", "discovery_method", "last_seen",
		"first_seen", "notes", "tags", "location", "category",
		"primary_role", "owner",
	}
}

// deviceToCSVRow converts a device to a CSV row (matching csvHeaders order).
func deviceToCSVRow(d models.Device) []string {
	return []string{
		d.ID,
		d.Hostname,
		strings.Join(d.IPAddresses, ";"),
		d.MACAddress,
		d.Manufacturer,
		string(d.DeviceType),
		d.OS,
		string(d.Status),
		string(d.DiscoveryMethod),
		d.LastSeen.Format(time.RFC3339),
		d.FirstSeen.Format(time.RFC3339),
		d.Notes,
		strings.Join(d.Tags, ";"),
		d.Location,
		d.Category,
		d.PrimaryRole,
		d.Owner,
	}
}

// csvColumnCount is the number of columns in the CSV format.
const csvColumnCount = 17

// csvRowToDevice parses a CSV row into a Device. Returns error for invalid data.
func csvRowToDevice(row []string) (models.Device, error) {
	if len(row) < csvColumnCount {
		return models.Device{}, fmt.Errorf("expected %d columns, got %d", csvColumnCount, len(row))
	}

	// Re-slice to exactly csvColumnCount so gosec can verify bounds statically.
	r := row[:csvColumnCount]

	var d models.Device
	d.ID = r[0]
	d.Hostname = r[1]

	if r[2] != "" {
		d.IPAddresses = strings.Split(r[2], ";")
	}

	d.MACAddress = r[3]
	d.Manufacturer = r[4]
	d.DeviceType = models.DeviceType(r[5])
	d.OS = r[6]
	d.Status = models.DeviceStatus(r[7])
	d.DiscoveryMethod = models.DiscoveryMethod(r[8])

	if r[9] != "" {
		t, err := time.Parse(time.RFC3339, r[9])
		if err != nil {
			return models.Device{}, fmt.Errorf("invalid last_seen: %w", err)
		}
		d.LastSeen = t
	}
	if r[10] != "" {
		t, err := time.Parse(time.RFC3339, r[10])
		if err != nil {
			return models.Device{}, fmt.Errorf("invalid first_seen: %w", err)
		}
		d.FirstSeen = t
	}

	d.Notes = r[11]

	if r[12] != "" {
		d.Tags = strings.Split(r[12], ";")
	}

	d.Location = r[13]
	d.Category = r[14]
	d.PrimaryRole = r[15]
	d.Owner = r[16]

	return d, nil
}
