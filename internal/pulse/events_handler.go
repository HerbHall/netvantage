package pulse

import (
	"context"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// handleDeviceDiscovered auto-creates an ICMP check when Recon discovers a new device.
func (m *Module) handleDeviceDiscovered(ctx context.Context, event plugin.Event) {
	if m.store == nil {
		return
	}

	de, ok := event.Payload.(*recon.DeviceEvent)
	if !ok {
		m.logger.Warn("unexpected payload type for device discovered event")
		return
	}

	if de.Device == nil {
		m.logger.Debug("device discovered event has nil device")
		return
	}

	if len(de.Device.IPAddresses) == 0 {
		m.logger.Debug("device discovered event has no IP addresses",
			zap.String("device_id", de.Device.ID),
		)
		return
	}

	// Check if a pulse check already exists for this device.
	existing, err := m.store.GetCheckByDeviceID(ctx, de.Device.ID)
	if err != nil {
		m.logger.Warn("failed to check existing pulse check",
			zap.String("device_id", de.Device.ID),
			zap.Error(err),
		)
		return
	}
	if existing != nil {
		return // Already monitored.
	}

	now := time.Now().UTC()
	check := &Check{
		ID:              fmt.Sprintf("pulse-%s", de.Device.ID),
		DeviceID:        de.Device.ID,
		CheckType:       "icmp",
		Target:          de.Device.IPAddresses[0],
		IntervalSeconds: int(m.cfg.CheckInterval.Seconds()),
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := m.store.InsertCheck(ctx, check); err != nil {
		m.logger.Warn("failed to auto-create pulse check",
			zap.String("device_id", de.Device.ID),
			zap.String("target", check.Target),
			zap.Error(err),
		)
		return
	}

	m.logger.Info("auto-created pulse check for discovered device",
		zap.String("check_id", check.ID),
		zap.String("device_id", de.Device.ID),
		zap.String("target", check.Target),
	)
}
