package recon

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/HerbHall/netvantage/internal/plugin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Plugin implements the Recon network discovery module.
type Plugin struct {
	logger *zap.Logger
	config *viper.Viper
}

// New creates a new Recon plugin instance.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return "recon" }
func (p *Plugin) Version() string { return "0.1.0" }

func (p *Plugin) Init(config *viper.Viper, logger *zap.Logger) error {
	p.config = config
	p.logger = logger
	p.logger.Info("recon module initialized")
	return nil
}

func (p *Plugin) Start(ctx context.Context) error {
	p.logger.Info("recon module started")
	return nil
}

func (p *Plugin) Stop() error {
	p.logger.Info("recon module stopped")
	return nil
}

func (p *Plugin) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "POST", Path: "/scan", Handler: p.handleScan},
		{Method: "GET", Path: "/scans", Handler: p.handleListScans},
	}
}

func (p *Plugin) handleScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "not_implemented",
		"message": "network scanning will be implemented in Phase 1",
	})
}

func (p *Plugin) handleListScans(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]any{})
}
