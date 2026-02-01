package plugin

import (
	"context"
	"net/http"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Route represents an HTTP route exposed by a plugin.
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// Plugin defines the interface that all NetVantage modules must implement.
type Plugin interface {
	// Name returns the plugin's unique identifier (e.g., "recon", "pulse").
	Name() string

	// Version returns the plugin's semantic version.
	Version() string

	// Init initializes the plugin with configuration and logger.
	Init(config *viper.Viper, logger *zap.Logger) error

	// Start begins the plugin's background operations.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the plugin.
	Stop() error

	// Routes returns the HTTP routes this plugin exposes.
	Routes() []Route
}
