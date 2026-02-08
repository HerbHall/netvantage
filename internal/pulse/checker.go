package pulse

import (
	"context"
	"fmt"
	"runtime"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// Checker executes a health check against a target and returns the result.
type Checker interface {
	Check(ctx context.Context, target string) (*CheckResult, error)
}

// ICMPChecker pings targets using ICMP via pro-bing.
type ICMPChecker struct {
	timeout time.Duration
	count   int
}

// NewICMPChecker creates a new ICMP checker with the given timeout and ping count.
func NewICMPChecker(timeout time.Duration, count int) *ICMPChecker {
	return &ICMPChecker{
		timeout: timeout,
		count:   count,
	}
}

// Check pings the target and returns the result.
func (c *ICMPChecker) Check(ctx context.Context, target string) (*CheckResult, error) {
	pinger, err := probing.NewPinger(target)
	if err != nil {
		return nil, fmt.Errorf("create pinger: %w", err)
	}

	pinger.Count = c.count
	pinger.Timeout = c.timeout
	pinger.SetPrivileged(runtime.GOOS == "windows")

	// Run pinger in a goroutine for context cancellation.
	done := make(chan error, 1)
	go func() {
		done <- pinger.Run()
	}()

	select {
	case runErr := <-done:
		// Pinger completed (possibly with error).
		stats := pinger.Statistics()
		result := &CheckResult{
			CheckedAt: time.Now().UTC(),
		}

		if runErr != nil {
			result.Success = false
			result.ErrorMessage = runErr.Error()
			result.PacketLoss = 1.0
			return result, nil
		}

		result.LatencyMs = float64(stats.AvgRtt) / float64(time.Millisecond)
		result.PacketLoss = stats.PacketLoss / 100.0 // pro-bing returns 0-100
		result.Success = stats.PacketsRecv > 0

		if !result.Success {
			result.ErrorMessage = "all packets lost"
		}

		return result, nil

	case <-ctx.Done():
		pinger.Stop()
		return &CheckResult{
			Success:      false,
			PacketLoss:   1.0,
			ErrorMessage: "check cancelled",
			CheckedAt:    time.Now().UTC(),
		}, nil
	}
}
