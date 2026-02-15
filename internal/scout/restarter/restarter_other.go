//go:build !windows

package restarter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// detectPlatform returns the best restarter for the current Linux/Unix environment.
func detectPlatform() Restarter {
	// Docker: check for /.dockerenv
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return &dockerRestarter{}
	}
	// systemd: check for /run/systemd/system
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return &systemdRestarter{}
	}
	// OpenRC: check for rc-service binary
	if _, err := exec.LookPath("rc-service"); err == nil {
		return &openrcRestarter{}
	}
	// Fallback: exec self
	return &execRestarter{}
}

// dockerRestarter signals Docker to restart the container by exiting with code 0.
// Requires restart policy (e.g., --restart=unless-stopped) on the container.
type dockerRestarter struct{}

func (r *dockerRestarter) Name() string { return "docker" }
func (r *dockerRestarter) Restart(_ context.Context) error {
	// Exit cleanly; Docker restart policy will restart the container.
	os.Exit(0)
	return nil // unreachable
}

// systemdRestarter uses systemctl to restart the scout service.
type systemdRestarter struct{}

func (r *systemdRestarter) Name() string { return "systemd" }
func (r *systemdRestarter) Restart(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "systemctl", "restart", "subnetree-scout")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("systemctl restart: %w", err)
	}
	// Don't wait -- systemctl will kill and restart us.
	return nil
}

// openrcRestarter uses rc-service to restart the scout service.
type openrcRestarter struct{}

func (r *openrcRestarter) Name() string { return "openrc" }
func (r *openrcRestarter) Restart(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "rc-service", "subnetree-scout", "restart")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("rc-service restart: %w", err)
	}
	return nil
}

// execRestarter re-execs the current binary using syscall.Exec.
// This replaces the current process with a new instance.
type execRestarter struct{}

func (r *execRestarter) Name() string { return "exec" }
func (r *execRestarter) Restart(_ context.Context) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	return syscall.Exec(exe, os.Args, os.Environ())
}
