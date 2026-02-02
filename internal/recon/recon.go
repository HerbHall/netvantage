package recon

import (
	"context"
	"strconv"
	"sync"

	"github.com/HerbHall/netvantage/pkg/plugin"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin        = (*Module)(nil)
	_ plugin.HTTPProvider  = (*Module)(nil)
	_ plugin.HealthChecker = (*Module)(nil)
)

// Module implements the Recon network discovery plugin.
type Module struct {
	logger       *zap.Logger
	cfg          ReconConfig
	store        *ReconStore
	bus          plugin.EventBus
	oui          *OUITable
	orchestrator *ScanOrchestrator
	activeScans  sync.Map // scanID -> context.CancelFunc
	wg           sync.WaitGroup
	scanCtx      context.Context
	scanCancel   context.CancelFunc
}

// New creates a new Recon plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "recon",
		Version:     "0.1.0",
		Description: "Network discovery and device scanning",
		Roles:       []string{"discovery"},
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(ctx context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger
	m.bus = deps.Bus

	// Load config with defaults.
	m.cfg = DefaultConfig()
	if deps.Config != nil {
		if d := deps.Config.GetDuration("scan_timeout"); d > 0 {
			m.cfg.ScanTimeout = d
		}
		if d := deps.Config.GetDuration("ping_timeout"); d > 0 {
			m.cfg.PingTimeout = d
		}
		if v := deps.Config.GetInt("ping_count"); v > 0 {
			m.cfg.PingCount = v
		}
		if v := deps.Config.GetInt("concurrency"); v > 0 {
			m.cfg.Concurrency = v
		}
		if deps.Config.IsSet("arp_enabled") {
			m.cfg.ARPEnabled = deps.Config.GetBool("arp_enabled")
		}
		if d := deps.Config.GetDuration("device_lost_after"); d > 0 {
			m.cfg.DeviceLostAfter = d
		}
	}

	// Run database migrations.
	if err := deps.Store.Migrate(ctx, "recon", migrations()); err != nil {
		return err
	}

	// Initialize store and scanners.
	m.store = NewReconStore(deps.Store.DB())
	m.oui = NewOUITable()

	pinger := NewICMPScanner(m.cfg, m.logger.Named("icmp"))
	var arp ARPTableReader
	if m.cfg.ARPEnabled {
		arp = NewARPReader(m.logger.Named("arp"))
	}

	m.orchestrator = NewScanOrchestrator(m.store, m.bus, m.oui, pinger, arp, m.logger)

	m.logger.Info("recon module initialized")
	return nil
}

func (m *Module) Start(_ context.Context) error {
	m.scanCtx, m.scanCancel = context.WithCancel(context.Background())
	m.logger.Info("recon module started")
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	m.logger.Info("recon module stopping, cancelling active scans")
	if m.scanCancel != nil {
		m.scanCancel()
	}
	// Cancel all individual scans.
	m.activeScans.Range(func(_, value any) bool {
		if cancel, ok := value.(context.CancelFunc); ok {
			cancel()
		}
		return true
	})
	m.wg.Wait()
	m.logger.Info("recon module stopped")
	return nil
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "POST", Path: "/scan", Handler: m.handleScan},
		{Method: "GET", Path: "/scans", Handler: m.handleListScans},
		{Method: "GET", Path: "/scans/{id}", Handler: m.handleGetScan},
		{Method: "GET", Path: "/topology", Handler: m.handleTopology},
	}
}

// Health implements plugin.HealthChecker.
func (m *Module) Health(_ context.Context) plugin.HealthStatus {
	var activeCount int
	m.activeScans.Range(func(_, _ any) bool {
		activeCount++
		return true
	})

	details := map[string]string{
		"active_scans": strconv.Itoa(activeCount),
		"arp_enabled":  strconv.FormatBool(m.cfg.ARPEnabled),
	}

	return plugin.HealthStatus{
		Status:  "ok",
		Details: details,
	}
}

// newScanContext creates a child context from the module's scan context.
func (m *Module) newScanContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(m.scanCtx)
}
