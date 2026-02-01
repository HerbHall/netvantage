package vault

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/HerbHall/netvantage/internal/plugin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Plugin implements the Vault credential management module.
type Plugin struct {
	logger *zap.Logger
	config *viper.Viper
}

// New creates a new Vault plugin instance.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string    { return "vault" }
func (p *Plugin) Version() string { return "0.1.0" }

func (p *Plugin) Init(config *viper.Viper, logger *zap.Logger) error {
	p.config = config
	p.logger = logger
	p.logger.Info("vault module initialized")
	return nil
}

func (p *Plugin) Start(ctx context.Context) error {
	p.logger.Info("vault module started")
	return nil
}

func (p *Plugin) Stop() error {
	p.logger.Info("vault module stopped")
	return nil
}

func (p *Plugin) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/credentials", Handler: p.handleListCredentials},
		{Method: "POST", Path: "/credentials", Handler: p.handleCreateCredential},
	}
}

func (p *Plugin) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]any{})
}

func (p *Plugin) handleCreateCredential(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "not_implemented",
		"message": "credential storage will be implemented in Phase 3",
	})
}
