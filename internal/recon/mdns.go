//go:build !windows

package recon

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
	"go.uber.org/zap"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
)

// mdnsDefaultServices lists well-known mDNS service types to query.
var mdnsDefaultServices = []string{
	"_http._tcp",
	"_https._tcp",
	"_ssh._tcp",
	"_smb._tcp",
	"_nfs._tcp",
	"_ipp._tcp",
	"_printer._tcp",
	"_airplay._tcp",
	"_raop._tcp",
	"_googlecast._tcp",
	"_homekit._tcp",
	"_hap._tcp",
	"_mqtt._tcp",
	"_workstation._tcp",
}

// MDNSListener passively discovers devices via mDNS/Bonjour service announcements.
type MDNSListener struct {
	store    *ReconStore
	bus      plugin.EventBus
	logger   *zap.Logger
	interval time.Duration

	mu   sync.Mutex
	seen map[string]time.Time // IP -> last seen time (deduplication)
}

// NewMDNSListener creates a new mDNS listener that periodically queries for
// common mDNS services and upserts discovered devices into the store.
func NewMDNSListener(store *ReconStore, bus plugin.EventBus, logger *zap.Logger, interval time.Duration) *MDNSListener {
	return &MDNSListener{
		store:    store,
		bus:      bus,
		logger:   logger,
		interval: interval,
		seen:     make(map[string]time.Time),
	}
}

// Run starts the periodic mDNS query loop. It blocks until ctx is cancelled.
// The caller is responsible for running this in a goroutine and calling wg.Done.
func (l *MDNSListener) Run(ctx context.Context) {
	l.logger.Info("mDNS listener started",
		zap.Duration("interval", l.interval),
		zap.Int("service_count", len(mdnsDefaultServices)),
	)

	// Run an initial scan immediately, then on a ticker.
	l.queryAllServices(ctx)

	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			l.logger.Info("mDNS listener stopped")
			return
		case <-ticker.C:
			l.queryAllServices(ctx)
		}
	}
}

// queryAllServices queries each mDNS service type and processes results.
func (l *MDNSListener) queryAllServices(ctx context.Context) {
	l.logger.Debug("mDNS scan starting")

	var discovered int
	for _, svc := range mdnsDefaultServices {
		if ctx.Err() != nil {
			return
		}
		n := l.queryService(ctx, svc)
		discovered += n
	}

	l.logger.Debug("mDNS scan complete", zap.Int("devices_found", discovered))
	l.cleanSeen()
}

// queryService queries a single mDNS service type and returns the number of
// new or updated devices found.
func (l *MDNSListener) queryService(ctx context.Context, service string) int {
	entries := make(chan *mdns.ServiceEntry, 16)

	var discovered int
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for entry := range entries {
			if l.processEntry(ctx, entry, service) {
				discovered++
			}
		}
	}()

	params := mdns.DefaultParams(service)
	params.Timeout = 3 * time.Second
	params.Entries = entries
	params.DisableIPv6 = true // Stick to IPv4 for simplicity.

	if err := mdns.Query(params); err != nil {
		l.logger.Debug("mDNS query failed",
			zap.String("service", service),
			zap.Error(err),
		)
	}
	close(entries)
	wg.Wait()

	return discovered
}

// processEntry converts an mDNS service entry into a device and upserts it.
// Returns true if the device was new or updated (not deduplicated).
func (l *MDNSListener) processEntry(ctx context.Context, entry *mdns.ServiceEntry, service string) bool {
	if entry == nil {
		return false
	}

	ip := l.extractIP(entry)
	if ip == "" {
		return false
	}

	// Deduplicate: skip if we've seen this IP within the current interval.
	if l.recentlySeen(ip) {
		return false
	}
	l.markSeen(ip)

	hostname := strings.TrimSuffix(entry.Host, ".")
	if hostname == "" {
		hostname = entry.Name
	}

	device := &models.Device{
		Hostname:        hostname,
		IPAddresses:     []string{ip},
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoverymDNS,
		DeviceType:      inferDeviceTypeFromService(service),
	}

	created, err := l.store.UpsertDevice(ctx, device)
	if err != nil {
		l.logger.Warn("mDNS device upsert failed",
			zap.String("ip", ip),
			zap.String("hostname", hostname),
			zap.Error(err),
		)
		return false
	}

	topic := TopicDeviceUpdated
	if created {
		topic = TopicDeviceDiscovered
	}
	l.publishEvent(ctx, topic, DeviceEvent{
		Device: device,
	})

	l.logger.Info("mDNS device discovered",
		zap.String("ip", ip),
		zap.String("hostname", hostname),
		zap.String("service", service),
		zap.Bool("new", created),
	)

	return true
}

// extractIP returns the best IP address from an mDNS service entry.
func (l *MDNSListener) extractIP(entry *mdns.ServiceEntry) string {
	if entry.AddrV4 != nil && !entry.AddrV4.IsUnspecified() {
		return entry.AddrV4.String()
	}
	// Fallback to deprecated Addr field for older mDNS implementations.
	if entry.Addr != nil && !entry.Addr.IsUnspecified() {
		return entry.Addr.String()
	}
	return ""
}

// recentlySeen returns true if the IP was seen within the current scan interval.
func (l *MDNSListener) recentlySeen(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	lastSeen, ok := l.seen[ip]
	if !ok {
		return false
	}
	return time.Since(lastSeen) < l.interval
}

// markSeen records the IP as recently seen.
func (l *MDNSListener) markSeen(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.seen[ip] = time.Now()
}

// cleanSeen removes entries older than 2x the scan interval.
func (l *MDNSListener) cleanSeen() {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := time.Now().Add(-2 * l.interval)
	for ip, t := range l.seen {
		if t.Before(cutoff) {
			delete(l.seen, ip)
		}
	}
}

// publishEvent publishes an event to the event bus.
func (l *MDNSListener) publishEvent(ctx context.Context, topic string, payload any) {
	if l.bus == nil {
		return
	}
	l.bus.PublishAsync(ctx, plugin.Event{
		Topic:     topic,
		Source:    "recon",
		Timestamp: time.Now(),
		Payload:   payload,
	})
}

// inferDeviceTypeFromService guesses the device type from the mDNS service name.
func inferDeviceTypeFromService(service string) models.DeviceType {
	switch {
	case strings.Contains(service, "printer") || strings.Contains(service, "ipp"):
		return models.DeviceTypePrinter
	case strings.Contains(service, "airplay") || strings.Contains(service, "raop") ||
		strings.Contains(service, "googlecast"):
		return models.DeviceTypeIoT
	case strings.Contains(service, "homekit") || strings.Contains(service, "hap") ||
		strings.Contains(service, "mqtt"):
		return models.DeviceTypeIoT
	default:
		return models.DeviceTypeUnknown
	}
}

