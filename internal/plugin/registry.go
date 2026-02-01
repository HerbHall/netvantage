package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Registry manages the lifecycle of all registered plugins.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	order   []string
	logger  *zap.Logger
}

// NewRegistry creates a new plugin registry.
func NewRegistry(logger *zap.Logger) *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
		logger:  logger,
	}
}

// Register adds a plugin to the registry.
func (r *Registry) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q already registered", name)
	}

	r.plugins[name] = p
	r.order = append(r.order, name)
	r.logger.Info("plugin registered", zap.String("name", name), zap.String("version", p.Version()))
	return nil
}

// InitAll initializes all registered plugins with their configuration.
func (r *Registry) InitAll(config *viper.Viper) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.order {
		p := r.plugins[name]

		pluginConfig := config.Sub("plugins." + name)
		if pluginConfig == nil {
			pluginConfig = viper.New()
		}

		if enabled := config.GetBool("plugins." + name + ".enabled"); !enabled {
			r.logger.Info("plugin disabled, skipping", zap.String("name", name))
			continue
		}

		r.logger.Info("initializing plugin", zap.String("name", name))
		if err := p.Init(pluginConfig, r.logger.Named(name)); err != nil {
			return fmt.Errorf("failed to initialize plugin %q: %w", name, err)
		}
	}
	return nil
}

// StartAll starts all initialized plugins.
func (r *Registry) StartAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.order {
		p := r.plugins[name]
		r.logger.Info("starting plugin", zap.String("name", name))
		if err := p.Start(ctx); err != nil {
			return fmt.Errorf("failed to start plugin %q: %w", name, err)
		}
	}
	return nil
}

// StopAll stops all plugins in reverse order.
func (r *Registry) StopAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := len(r.order) - 1; i >= 0; i-- {
		name := r.order[i]
		p := r.plugins[name]
		r.logger.Info("stopping plugin", zap.String("name", name))
		if err := p.Stop(); err != nil {
			r.logger.Error("failed to stop plugin", zap.String("name", name), zap.Error(err))
		}
	}
}

// Get returns a plugin by name.
func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[name]
	return p, ok
}

// All returns all registered plugins in registration order.
func (r *Registry) All() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Plugin, 0, len(r.order))
	for _, name := range r.order {
		result = append(result, r.plugins[name])
	}
	return result
}

// AllRoutes returns all routes from all registered plugins.
func (r *Registry) AllRoutes() map[string][]Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make(map[string][]Route)
	for _, name := range r.order {
		p := r.plugins[name]
		if pr := p.Routes(); len(pr) > 0 {
			routes[name] = pr
		}
	}
	return routes
}
