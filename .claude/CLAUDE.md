# NetVantage - Claude Code Project Configuration

## Project Overview

NetVantage is a modular network monitoring and management platform written in Go. It consists of a server with plugin-based modules and a lightweight agent (Scout) for monitored devices.

## Architecture

- **Server** (`cmd/netvantage/`): Central application with HTTP API, plugin registry
- **Scout** (`cmd/scout/`): Lightweight agent installed on monitored devices
- **Modules** (`internal/`): Recon (scanning), Pulse (monitoring), Dispatch (agent mgmt), Vault (credentials), Gateway (remote access)
- **Shared** (`pkg/models/`): Types shared between server and agent
- **Proto** (`api/proto/v1/`): gRPC service definitions

## Build Commands

```bash
# Build everything
make build

# Build server only
make build-server

# Build agent only
make build-scout

# Run server
make run-server

# Run tests
make test

# Run linter
make lint

# Generate protobuf code
make proto

# Clean build artifacts
make clean
```

## Go Conventions

- Module path: `github.com/HerbHall/netvantage`
- Go 1.25+
- Use `internal/` for private packages, `pkg/` for public
- Standard Go project layout
- Structured logging via `go.uber.org/zap`
- Configuration via `github.com/spf13/viper`
- gRPC for agent-server communication

## Code Style

- Follow standard Go conventions (gofmt, go vet)
- Error handling: return errors, don't panic
- Use context.Context for cancellation/timeouts
- Interfaces in the consumer package
- Table-driven tests

## Plugin Architecture

Each module implements the `plugin.Plugin` interface:
- `Name() string`
- `Version() string`
- `Init(config *viper.Viper, logger *zap.Logger) error`
- `Start(ctx context.Context) error`
- `Stop() error`
- `Routes() []Route`

Plugins are registered at compile time in `cmd/netvantage/main.go`.

## Git Conventions

- Conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`
- Branch naming: `feature/`, `fix/`, `refactor/`
- Co-author tag: `Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>`
