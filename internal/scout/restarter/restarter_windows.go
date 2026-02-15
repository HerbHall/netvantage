//go:build windows

package restarter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// detectPlatform returns the best restarter for Windows.
func detectPlatform() Restarter {
	// Check if running as a Windows service by querying our service name.
	cmd := exec.Command("sc", "query", "SubNetreeScout")
	if err := cmd.Run(); err == nil {
		return &serviceRestarter{}
	}
	return &execRestarter{}
}

// serviceRestarter uses sc.exe to restart the Windows service.
type serviceRestarter struct{}

func (r *serviceRestarter) Name() string { return "windows-service" }
func (r *serviceRestarter) Restart(ctx context.Context) error {
	// Stop then start; sc doesn't have a "restart" command.
	stop := exec.CommandContext(ctx, "sc", "stop", "SubNetreeScout")
	if err := stop.Run(); err != nil {
		return fmt.Errorf("sc stop: %w", err)
	}
	start := exec.CommandContext(ctx, "sc", "start", "SubNetreeScout")
	if err := start.Start(); err != nil {
		return fmt.Errorf("sc start: %w", err)
	}
	return nil
}

// execRestarter re-execs the current binary on Windows.
type execRestarter struct{}

func (r *execRestarter) Name() string { return "exec" }
func (r *execRestarter) Restart(_ context.Context) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	// Windows doesn't support syscall.Exec the same way.
	// Start a new process and exit the current one.
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	if startErr := cmd.Start(); startErr != nil {
		return fmt.Errorf("start new process: %w", startErr)
	}
	os.Exit(0)
	return nil // unreachable
}
