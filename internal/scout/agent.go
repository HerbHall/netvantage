package scout

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"go.uber.org/zap"
)

// Agent is the Scout monitoring agent.
type Agent struct {
	config *Config
	logger *zap.Logger
	cancel context.CancelFunc
}

// NewAgent creates a new Scout agent instance.
func NewAgent(config *Config, logger *zap.Logger) *Agent {
	return &Agent{
		config: config,
		logger: logger,
	}
}

// Run starts the agent and blocks until the context is cancelled.
func (a *Agent) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	defer cancel()

	a.logger.Info("scout agent starting",
		zap.String("server", a.config.ServerAddr),
		zap.String("platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)),
		zap.Int("check_interval_seconds", a.config.CheckInterval),
	)

	// TODO: Establish gRPC connection to server
	// TODO: Perform enrollment if no agent ID

	ticker := time.NewTicker(time.Duration(a.config.CheckInterval) * time.Second)
	defer ticker.Stop()

	a.logger.Info("scout agent running, waiting for check-in interval")

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("scout agent shutting down")
			return nil
		case <-ticker.C:
			a.checkIn()
		}
	}
}

// Stop signals the agent to shut down.
func (a *Agent) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
}

// checkIn performs a periodic check-in with the server.
func (a *Agent) checkIn() {
	a.logger.Debug("performing check-in",
		zap.String("server", a.config.ServerAddr),
	)
	// TODO: Send gRPC CheckIn request with system metrics
}
