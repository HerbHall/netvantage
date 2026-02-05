## Observability

### Structured Logging

Configurable Zap logger factory supporting:
- **Level:** debug, info, warn, error (configurable, default: info)
- **Format:** json (production), console (development with color)
- **Per-plugin named loggers:** `logger.Named("recon")` for filtering

#### Logging Conventions

| Context | Required Fields |
|---------|----------------|
| HTTP requests | request_id, method, path, status, duration, remote_addr |
| Plugin operations | plugin name (via Named logger) |
| Agent communication | agent_id |
| Database operations | query name, duration |
| Credential access | credential_id, action, user_id |

### Prometheus Metrics

Exposed at `GET /metrics` from day one.

#### Metric Naming Convention

`subnetree_{subsystem}_{metric}_{unit}` (e.g., `subnetree_http_request_duration_seconds`)

#### Core Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `subnetree_http_requests_total` | Counter | method, path, status_code | Total HTTP requests |
| `subnetree_http_request_duration_seconds` | Histogram | method, path | Request latency |
| `subnetree_recon_devices_total` | Gauge | status | Discovered devices by status |
| `subnetree_recon_scans_total` | Counter | status | Network scans by outcome |
| `subnetree_recon_scan_duration_seconds` | Histogram | -- | Scan duration |
| `subnetree_dispatch_agents_connected` | Gauge | -- | Connected Scout agents |
| `subnetree_dispatch_agent_checkins_total` | Counter | -- | Agent check-in RPCs |
| `subnetree_vault_access_total` | Counter | action, success | Credential vault accesses |
| `subnetree_db_query_duration_seconds` | Histogram | query | Database query latency |

### Health Endpoints

| Endpoint | Purpose | Checks |
|----------|---------|--------|
| `GET /healthz` | **Liveness** -- Is the process alive? | Always 200 unless deadlocked. Never call DB. |
| `GET /readyz` | **Readiness** -- Can we handle requests? | DB connectivity, plugin health status. 503 if not ready. |

### OpenTelemetry Tracing (Phase 2)

- OTLP exporter for distributed tracing
- Trace scan operations: ICMP sweep -> ARP scan -> SNMP enrichment -> OUI lookup
- Trace agent check-in pipeline
- 10% sampling rate by default
