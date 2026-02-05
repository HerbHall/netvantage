package testutil

import (
	"time"

	"github.com/google/uuid"

	"github.com/HerbHall/subnetree/pkg/models"
)

// NewDevice returns a Device with sensible defaults, suitable for test fixtures.
// Override individual fields after creation as needed.
func NewDevice(opts ...func(*models.Device)) models.Device {
	d := models.Device{
		ID:              uuid.New().String(),
		Hostname:        "test-device",
		IPAddresses:     []string{"192.168.1.100"},
		MACAddress:      "00:11:22:33:44:55",
		DeviceType:      models.DeviceTypeDesktop,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
		FirstSeen:       time.Now().UTC(),
		LastSeen:        time.Now().UTC(),
	}
	for _, opt := range opts {
		opt(&d)
	}
	return d
}

// WithHostname sets the device hostname.
func WithHostname(name string) func(*models.Device) {
	return func(d *models.Device) { d.Hostname = name }
}

// WithIP sets the device's IP address list.
func WithIP(ips ...string) func(*models.Device) {
	return func(d *models.Device) { d.IPAddresses = ips }
}

// WithMAC sets the device's MAC address.
func WithMAC(mac string) func(*models.Device) {
	return func(d *models.Device) { d.MACAddress = mac }
}

// WithStatus sets the device status.
func WithStatus(s models.DeviceStatus) func(*models.Device) {
	return func(d *models.Device) { d.Status = s }
}

// WithLastSeen sets the device's last_seen timestamp.
func WithLastSeen(t time.Time) func(*models.Device) {
	return func(d *models.Device) { d.LastSeen = t }
}

// WithDeviceType sets the device type.
func WithDeviceType(dt models.DeviceType) func(*models.Device) {
	return func(d *models.Device) { d.DeviceType = dt }
}
