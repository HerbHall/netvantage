// Package restarter provides init-system-aware restart capabilities for Scout.
package restarter

import "context"

// Restarter abstracts process restart across init systems.
type Restarter interface {
	// Name returns the init system name (e.g., "systemd", "docker", "exec").
	Name() string
	// Restart requests a process restart via the init system.
	Restart(ctx context.Context) error
}

// Detect returns a Restarter appropriate for the current environment.
// Returns nil if auto-restart is not supported.
func Detect() Restarter {
	return detectPlatform()
}
