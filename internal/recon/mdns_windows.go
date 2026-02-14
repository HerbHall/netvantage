//go:build windows

package recon

import (
	"context"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// MDNSListener is a no-op stub on Windows where multicast DNS is not
// reliably supported.
type MDNSListener struct{}

// NewMDNSListener returns a no-op mDNS listener on Windows.
func NewMDNSListener(_ *ReconStore, _ plugin.EventBus, _ *zap.Logger, _ time.Duration) *MDNSListener {
	return &MDNSListener{}
}

// Run is a no-op on Windows. It returns immediately when ctx is cancelled.
func (l *MDNSListener) Run(ctx context.Context) {
	<-ctx.Done()
}
