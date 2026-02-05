# Development Setup Guide

This guide walks through setting up a local development environment for SubNetree.

## Prerequisites

- **Go 1.25+** -- [Download](https://go.dev/dl/)
- **Git** -- [Download](https://git-scm.com/downloads)
- **Make** -- Included on Linux/macOS; on Windows, use MSYS2 or WSL
- **protoc** (optional) -- Only needed if modifying gRPC definitions

Verify your Go installation:

```bash
go version
# Expected: go1.25.x or later
```

## Clone and Build

```bash
git clone https://github.com/HerbHall/subnetree.git
cd subnetree
make build
```

This produces two binaries in `bin/`:
- `subnetree` -- the server
- `scout` -- the lightweight agent

## Project Structure

```
cmd/
  subnetree/    # Server entry point
  scout/         # Agent entry point
internal/
  config/        # Viper-backed Config implementation
  event/         # In-memory EventBus
  registry/      # Plugin lifecycle management
  recon/         # Network discovery module
  pulse/         # Monitoring module
  dispatch/      # Agent management module
  vault/         # Credential storage module
  gateway/       # Remote access module
  scout/         # Agent runtime
  server/        # HTTP server and config loading
  version/       # Build-time version injection
pkg/
  plugin/        # Public plugin SDK (Apache 2.0)
  models/        # Shared data types
api/
  proto/v1/      # gRPC service definitions
docs/
  adr/           # Architecture Decision Records
  guides/        # Developer guides (this file)
  requirements/  # Split requirement specifications
```

## Running

### Server

```bash
# With defaults (port 8080, SQLite in ./data/)
make run-server

# Or directly
./bin/subnetree

# With a config file
./bin/subnetree --config ./configs/subnetree.yaml

# Print version
./bin/subnetree --version
```

### Agent

```bash
# With defaults (connects to localhost:9090)
make run-scout

# Or with options
./bin/scout --server localhost:9090 --interval 30
```

## Configuration

SubNetree uses [Viper](https://github.com/spf13/viper) for configuration. Sources (in priority order):

1. Environment variables with `NV_` prefix (e.g., `NV_SERVER_PORT=9090`)
2. Config file (`subnetree.yaml` in current dir, `./configs/`, or `/etc/subnetree/`)
3. Built-in defaults

Example config file:

```yaml
server:
  host: 0.0.0.0
  port: 8080
  data_dir: ./data

logging:
  level: info
  format: json

plugins:
  recon:
    enabled: true
  pulse:
    enabled: true
  dispatch:
    enabled: true
  vault:
    enabled: true
  gateway:
    enabled: false  # Requires Apache Guacamole
```

## Testing

```bash
# Run all tests
make test

# Run with race detector (requires CGo / Linux)
make test-race

# Run with coverage report
make test-coverage

# Run linter (go vet)
make lint
```

## Plugin Development

All plugins implement the `plugin.Plugin` interface from `pkg/plugin/`:

```go
import "github.com/HerbHall/subnetree/pkg/plugin"

type MyPlugin struct{}

func (p *MyPlugin) Info() plugin.PluginInfo {
    return plugin.PluginInfo{
        Name:       "myplugin",
        Version:    "0.1.0",
        APIVersion: plugin.APIVersionCurrent,
    }
}

func (p *MyPlugin) Init(ctx context.Context, deps plugin.Dependencies) error { ... }
func (p *MyPlugin) Start(ctx context.Context) error { ... }
func (p *MyPlugin) Stop(ctx context.Context) error { ... }
```

Optional interfaces (implement only what you need):
- `plugin.HTTPProvider` -- expose REST routes
- `plugin.HealthChecker` -- report health status
- `plugin.EventSubscriber` -- declare event subscriptions
- `plugin.Validator` -- validate config post-init
- `plugin.Reloadable` -- support hot config reload

Register plugins in `cmd/subnetree/main.go`.

## Code Style

- Run `gofmt` before committing
- Follow standard Go conventions
- Use `context.Context` for cancellation
- Return errors; do not panic
- Use table-driven tests
- Conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`

## Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build server and agent |
| `make build-server` | Build server only |
| `make build-scout` | Build agent only |
| `make test` | Run all tests |
| `make test-race` | Run tests with race detector |
| `make test-coverage` | Run tests with coverage report |
| `make lint` | Run go vet |
| `make run-server` | Build and run server |
| `make run-scout` | Build and run agent |
| `make proto` | Regenerate protobuf code |
| `make clean` | Remove build artifacts |
| `make license-check` | Verify dependency licenses |
