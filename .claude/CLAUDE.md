# NetVantage - Claude Code Project Configuration

## Project Overview

NetVantage is a modular, source-available network monitoring and management platform written in Go. It consists of a server with plugin-based modules and a lightweight agent (Scout) for monitored devices.

**Free for personal/home use forever. Commercial for business use.** Licensed under BSL 1.1 (core) and Apache 2.0 (plugin SDK). Built with acquisition readiness in mind.

## Guiding Principles

These principles govern every development decision. When in doubt, refer here:

1. **Ease of use first.** No tech degree required. Intuitive for non-technical users, powerful for experts. If it needs a manual to understand, simplify the UI.
2. **Sensible defaults, deep customization.** Ship preconfigured for instant deployment. Every aspect is user-configurable. Defaults get you running; customization makes it yours.
3. **Stability and security are non-negotiable.** Every release must be stable enough for production infrastructure and secure enough to trust with credentials. If a feature compromises either, it does not ship.
4. **Plugin-powered architecture.** Every major feature is a plugin. The core is minimal. Users and developers can replace, extend, or supplement any module.
5. **Progressive disclosure.** Simple by default, advanced on demand. Never overwhelm a first-time user.

## Architecture

- **Server** (`cmd/netvantage/`): Central application with HTTP API, plugin registry
- **Scout** (`cmd/scout/`): Lightweight agent installed on monitored devices
- **Dashboard** (`web/`): React + TypeScript SPA served by the server
- **Modules** (`internal/`): Recon (scanning), Pulse (monitoring), Dispatch (agent mgmt), Vault (credentials), Gateway (remote access)
- **Plugin SDK** (`pkg/plugin/`, `pkg/roles/`, `pkg/models/`): Public interfaces, Apache 2.0 licensed
- **Proto** (`api/proto/v1/`): gRPC service definitions, Apache 2.0 licensed
- **Design System** (`web/src/styles/design-tokens.css`, `web/tailwind.config.ts`): Forest green + earth tone palette

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

# Check dependency licenses
make license-check

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
- Database: `modernc.org/sqlite` (pure Go, no CGo)

## Code Style

- Follow standard Go conventions (gofmt, go vet)
- Error handling: return errors, don't panic
- Use context.Context for cancellation/timeouts
- Interfaces in the consumer package
- Table-driven tests
- No ORM -- raw SQL with thin repository layer

## Plugin Architecture

Each module implements the `plugin.Plugin` interface:
- `Info() PluginInfo` -- metadata, dependencies, roles
- `Init(ctx context.Context, deps Dependencies) error`
- `Start(ctx context.Context) error`
- `Stop(ctx context.Context) error`

Optional interfaces detected via type assertions:
- `HTTPProvider` -- REST API routes
- `GRPCProvider` -- gRPC services
- `HealthChecker` -- health reporting
- `EventSubscriber` -- event bus subscriptions
- `Validator` -- config validation
- `Reloadable` -- hot config reload

Plugins are registered at compile time in `cmd/netvantage/main.go`.

## Licensing

- **Core (BSL 1.1):** `LICENSE` at repo root. Change Date: 4 years, converts to Apache 2.0.
- **Plugin SDK (Apache 2.0):** `pkg/plugin/`, `pkg/roles/`, `pkg/models/`, `api/proto/`
- **Block:** GPL, AGPL, LGPL, SSPL dependencies. Use `make license-check` to verify.
- **CLA required** for all contributions (GitHub Actions workflow).

## Git Conventions

- Conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`
- Branch naming: `feature/`, `fix/`, `refactor/`
- Co-author tag: `Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>`
- Branch protection on `main`: PRs required, CLA check must pass
