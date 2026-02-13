//go:build !windows

package metrics

import (
	"context"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

// stubCollector returns zero metrics on non-Windows platforms.
type stubCollector struct {
	logger *zap.Logger
}

// Compile-time guard.
var _ Collector = (*stubCollector)(nil)

func newPlatformCollector(logger *zap.Logger) Collector {
	logger.Warn("metrics collection is only supported on Windows; returning zero metrics")
	return &stubCollector{logger: logger}
}

func (c *stubCollector) Collect(_ context.Context) (*scoutpb.SystemMetrics, error) {
	return &scoutpb.SystemMetrics{}, nil
}
