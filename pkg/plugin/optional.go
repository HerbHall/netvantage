package plugin

import "context"

// HTTPProvider is implemented by plugins that expose REST API routes.
type HTTPProvider interface {
	Routes() []Route
}

// HealthChecker is implemented by plugins that report their health status.
type HealthChecker interface {
	Health(ctx context.Context) HealthStatus
}

// EventSubscriber is implemented by plugins that declare event subscriptions at init.
type EventSubscriber interface {
	Subscriptions() []Subscription
}

// Validator is implemented by plugins that validate their config post-init.
type Validator interface {
	ValidateConfig() error
}

// Reloadable is implemented by plugins that support config hot-reload.
type Reloadable interface {
	Reload(ctx context.Context, config Config) error
}
