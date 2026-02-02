package recon

import "time"

// Event topics published by the Recon module.
const (
	TopicDeviceDiscovered = "recon.device.discovered"
	TopicDeviceUpdated    = "recon.device.updated"
	TopicDeviceLost       = "recon.device.lost"
	TopicScanStarted      = "recon.scan.started"
	TopicScanCompleted    = "recon.scan.completed"
)

// DeviceLostEvent is the payload for TopicDeviceLost events.
type DeviceLostEvent struct {
	DeviceID string    `json:"device_id"`
	IP       string    `json:"ip"`
	LastSeen time.Time `json:"last_seen"`
}
