package pulse

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/HerbHall/netvantage/internal/plugin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Plugin implements the Pulse monitoring module.
type Plugin struct {
	logger *zap.Logger
	config *viper.Viper
}

// New creates a new Pulse plugin instance.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return "pulse" }
func (p *Plugin) Version() string { return "0.1.0" }

func (p *Plugin) Init(config *viper.Viper, logger *zap.Logger) error {
	p.config = config
	p.logger = logger
	p.logger.Info("pulse module initialized")
	return nil
}

func (p *Plugin) Start(ctx context.Context) error {
	p.logger.Info("pulse module started")
	return nil
}

func (p *Plugin) Stop() error {
	p.logger.Info("pulse module stopped")
	return nil
}

func (p *Plugin) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/status", Handler: p.handleStatus},
	}
}

func (p *Plugin) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "not_implemented",
		"message": "monitoring will be implemented in Phase 2",
	})
}
