// Package version provides build-time version information for NetVantage components.
// Variables are injected at build time via ldflags.
package version

import (
	"fmt"
	"runtime"
)

// Build-time variables injected via ldflags.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// Info returns a formatted version string suitable for --version output.
func Info() string {
	return fmt.Sprintf("NetVantage %s (commit: %s, built: %s, go: %s)",
		Version, GitCommit, BuildDate, runtime.Version())
}

// Short returns just the version string (e.g., "0.1.0" or "dev").
func Short() string {
	return Version
}

// Map returns version info as a map for JSON serialization.
func Map() map[string]string {
	return map[string]string{
		"version":    Version,
		"git_commit": GitCommit,
		"build_date": BuildDate,
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	}
}
