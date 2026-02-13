package metrics

import (
	"context"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

// Collector gathers system metrics from the host.
type Collector interface {
	Collect(ctx context.Context) (*scoutpb.SystemMetrics, error)
}

// NewCollector returns a platform-appropriate metrics collector.
func NewCollector(logger *zap.Logger) Collector {
	return newPlatformCollector(logger)
}
