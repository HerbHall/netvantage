package gateway

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/HerbHall/netvantage/internal/plugin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Plugin implements the Gateway remote access module.
type Plugin struct {
	logger *zap.Logger
	config *viper.Viper
}

// New creates a new Gateway plugin instance.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return "gateway" }
func (p *Plugin) Version() string { return "0.1.0" }

func (p *Plugin) Init(config *viper.Viper, logger *zap.Logger) error {
	p.config = config
	p.logger = logger
	p.logger.Info("gateway module initialized")
	return nil
}

func (p *Plugin) Start(ctx context.Context) error {
	p.logger.Info("gateway module started")
	return nil
}

func (p *Plugin) Stop() error {
	p.logger.Info("gateway module stopped")
	return nil
}

func (p *Plugin) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/sessions", Handler: p.handleListSessions},
	}
}

func (p *Plugin) handleListSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]any{})
}
