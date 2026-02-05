---
title: Configuration
weight: 2
---

SubNetree uses a YAML configuration file with sensible defaults. The server runs out of the box with no configuration required.

## Configuration File

By default, SubNetree looks for configuration in:

1. `./subnetree.yaml` (current directory)
2. `$HOME/.config/subnetree/subnetree.yaml`
3. `/etc/subnetree/subnetree.yaml`

Or specify a path explicitly:

```bash
./bin/subnetree -config /path/to/subnetree.yaml
```

## Example Configuration

An example configuration file is included in the repository:

```bash
cp configs/subnetree.example.yaml subnetree.yaml
```

## Environment Variables

All configuration values can be overridden via environment variables using the `SUBNETREE_` prefix with underscore-separated paths.

| YAML Path | Environment Variable | Default |
|-----------|---------------------|---------|
| `server.http.port` | `SUBNETREE_SERVER_HTTP_PORT` | `8080` |
| `server.grpc.port` | `SUBNETREE_SERVER_GRPC_PORT` | `9090` |
| `database.path` | `SUBNETREE_DATABASE_PATH` | `./data/subnetree.db` |
| `log.level` | `SUBNETREE_LOG_LEVEL` | `info` |
| `log.format` | `SUBNETREE_LOG_FORMAT` | `json` |

## API Endpoints

Once running, the following endpoints are available:

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/health` | Aggregated health status from all plugins |
| `GET /api/v1/plugins` | List of registered plugins and their status |

All API responses include the `X-SubNetree-Version` header.

{{< callout type="info" >}}
Configuration options will expand as new modules are implemented. This page reflects the current Phase 1 configuration surface. See the [full requirements](https://github.com/HerbHall/subnetree/tree/main/docs/requirements) for planned configuration options.
{{< /callout >}}
