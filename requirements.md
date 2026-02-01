# NetVantage Requirements

## Product Vision

NetVantage is a modular, open-source network monitoring and management platform that provides unified device discovery, monitoring, remote access, credential management, and IoT awareness in a single self-hosted application.

**Target Users:** Home lab enthusiasts, prosumers, small business IT administrators, managed service providers (MSPs).

**Core Value Proposition:** No existing open-source tool combines device discovery, monitoring, remote access, credential management, and IoT awareness in a single product with a commercially friendly license.

---

## Architecture Overview

### Components

| Component | Name | Description |
|-----------|------|-------------|
| Server | **NetVantage** | Central application: HTTP API, plugin registry, data storage, web dashboard |
| Agent | **Scout** | Lightweight Go agent installed on monitored devices |
| Dashboard | *web/* | React + TypeScript SPA served by the server |

### Server Modules (Plugins)

| Module | Name | Purpose |
|--------|------|---------|
| Discovery | **Recon** | Network scanning, device discovery (ICMP, ARP, SNMP, mDNS, UPnP, SSDP) |
| Monitoring | **Pulse** | Health checks, uptime monitoring, metrics collection, alerting |
| Agent Management | **Dispatch** | Scout agent enrollment, check-in, command dispatch, status tracking |
| Credentials | **Vault** | Encrypted credential storage, per-device credential assignment |
| Remote Access | **Gateway** | Browser-based SSH, RDP (via Guacamole), HTTP/HTTPS reverse proxy, VNC |

### Communication

- **Server <-> Dashboard:** REST API + WebSocket (real-time updates)
- **Server <-> Scout:** gRPC with mTLS (bidirectional streaming)
- **Server <-> Network Devices:** ICMP, ARP, SNMP v2c/v3, mDNS, UPnP/SSDP, MQTT

---

## Technology Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| Server | Go (1.25+) | Performance, single binary deployment, strong networking stdlib |
| Agent | Go | Same language as server, cross-compiles to all targets |
| Dashboard | React + TypeScript (Vite) | Largest ecosystem, rich component libraries |
| Agent Communication | gRPC + Protobuf | Efficient binary protocol, bidirectional streaming, code generation |
| Real-time UI | WebSocket / Server-Sent Events | Push updates to dashboard without polling |
| Configuration | Viper (YAML) | Standard Go config library, env var support |
| Logging | Zap | High-performance structured logging |
| Database (Phase 1) | SQLite via CGo | Zero-config, embedded, sufficient for single-server |
| Database (Phase 2+) | PostgreSQL + TimescaleDB | Time-series metrics at scale |
| HTTP Routing | net/http (stdlib) | No unnecessary dependencies for Phase 1 |
| Remote Desktop | Apache Guacamole (Docker) | Apache 2.0 licensed, proven RDP/VNC gateway |
| SSH Terminal | xterm.js + Go SSH library | Browser-based SSH terminal |
| HTTP Proxy | Go reverse proxy (stdlib) | Access device web interfaces through server |
| SNMP | gosnmp | Pure Go SNMP library |
| MQTT | Eclipse Paho Go | MQTT client for IoT device communication |

---

## Plugin Architecture

### Interface

Every module implements the `plugin.Plugin` interface:

```go
type Plugin interface {
    Name() string
    Version() string
    Init(config *viper.Viper, logger *zap.Logger) error
    Start(ctx context.Context) error
    Stop() error
    Routes() []Route
}
```

### Composition

- **Phase 1:** Compile-time composition. Plugins registered in `main.go`.
- **Future:** Potential runtime loading via Go plugin system or hashicorp/go-plugin.

### Plugin Lifecycle

1. `Init()` -- Load config, validate, prepare resources
2. `Start()` -- Begin background work (goroutines, listeners)
3. `Stop()` -- Graceful shutdown, flush buffers, close connections

### Route Registration

Each plugin exposes HTTP routes via `Routes()`. The server mounts them under `/api/v1/{plugin-name}/`.

---

## Scout Agent Specification

### Purpose

Lightweight agent installed on monitored devices to report system metrics, accept commands, and facilitate remote access.

### Capabilities

- System metrics: CPU, memory, disk, network usage
- Process listing
- Service status monitoring
- Log forwarding (opt-in)
- Command execution (authorized commands only)
- Auto-update (pull new versions from server)

### Communication

- gRPC with mTLS to server
- Periodic check-in (configurable interval, default 30s)
- Bidirectional streaming for real-time commands

### Platforms

| Platform | Priority | Method |
|----------|----------|--------|
| Windows x64 | Phase 1b | Native Go binary, Windows service |
| Linux x64 | Phase 2 | Native Go binary, systemd unit |
| Linux ARM64 | Phase 2 | Cross-compiled Go binary |
| macOS | Phase 3 | Native Go binary, launchd plist |
| Android | Deferred | Passive monitoring only (ping, ARP, mDNS) |
| IoT/Embedded | Phase 4 | Lightweight Go binary or MQTT-based |

### Security

- Agent authenticates to server via enrollment token + mTLS certificate
- Server issues per-agent certificates during enrollment
- Commands require server-side authorization
- Agent binary is open-source for user trust

---

## Data Model (Core Entities)

### Device

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| Hostname | string | Device hostname |
| IPAddresses | []string | All known IP addresses |
| MACAddress | string | Primary MAC address |
| Manufacturer | string | Derived from OUI database |
| DeviceType | enum | server, desktop, laptop, mobile, router, switch, printer, iot, unknown |
| OS | string | Operating system (if known) |
| Status | enum | online, offline, degraded, unknown |
| DiscoveryMethod | enum | agent, icmp, arp, snmp, mdns, upnp, mqtt, manual |
| AgentID | UUID? | Linked Scout agent (if any) |
| LastSeen | timestamp | Last successful contact |
| FirstSeen | timestamp | Initial discovery |
| Notes | string | User-provided notes |
| Tags | []string | User-defined tags |
| CustomFields | map | User-defined key-value pairs |

### Agent (Scout)

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| DeviceID | UUID | Linked device |
| Version | string | Agent software version |
| Status | enum | connected, disconnected, stale |
| LastCheckIn | timestamp | Last successful check-in |
| EnrolledAt | timestamp | Enrollment timestamp |
| Certificate | blob | mTLS certificate |
| Platform | string | OS/architecture |

### Credential (Vault)

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| Name | string | Display name |
| Type | enum | ssh_password, ssh_key, rdp, http_basic, snmp_community, snmp_v3, api_key |
| Data | encrypted blob | Encrypted credential data |
| DeviceIDs | []UUID | Associated devices |
| CreatedAt | timestamp | Creation timestamp |
| UpdatedAt | timestamp | Last modification |

---

## API Design

### REST API

Base path: `/api/v1/`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Server health check |
| `/devices` | GET | List all devices |
| `/devices/{id}` | GET | Get device details |
| `/devices` | POST | Create device manually |
| `/devices/{id}` | PUT | Update device |
| `/devices/{id}` | DELETE | Remove device |
| `/recon/scan` | POST | Trigger network scan |
| `/recon/scans` | GET | List scan history |
| `/pulse/status` | GET | Overall monitoring status |
| `/dispatch/agents` | GET | List connected agents |
| `/dispatch/agents/{id}` | GET | Agent details |
| `/vault/credentials` | GET | List credentials (metadata only) |
| `/vault/credentials` | POST | Store new credential |
| `/gateway/sessions` | GET | List active remote sessions |
| `/gateway/ssh/{device_id}` | WebSocket | SSH terminal session |
| `/gateway/proxy/{device_id}` | ANY | HTTP reverse proxy to device |

### gRPC Services (Agent Communication)

```protobuf
service ScoutService {
  rpc CheckIn(CheckInRequest) returns (CheckInResponse);
  rpc ReportMetrics(stream MetricsReport) returns (Ack);
  rpc CommandStream(stream CommandResponse) returns (stream Command);
}
```

---

## Phased Roadmap

### Phase 1: Foundation (Server + Dashboard + Agentless Scanning)

**Goal:** Functional web-based network scanner. Validate architecture.

- [ ] Go server with plugin architecture
- [ ] HTTP API with health endpoint
- [ ] Plugin registry and lifecycle management
- [ ] Recon module: ICMP ping sweep, ARP scanning
- [ ] SQLite database with device storage
- [ ] Basic React dashboard: device list, scan trigger
- [ ] WebSocket for real-time scan updates
- [ ] Configuration via YAML file
- [ ] Structured logging

### Phase 1b: Windows Scout Agent

**Goal:** First agent reporting metrics to server.

- [ ] Scout agent binary for Windows
- [ ] gRPC communication with mTLS
- [ ] Agent enrollment flow
- [ ] System metrics: CPU, memory, disk, network
- [ ] Dispatch module: agent list, status, check-in tracking
- [ ] Dashboard: agent status view

### Phase 2: Core Monitoring

**Goal:** Comprehensive network awareness.

- [ ] Recon: SNMP v2c/v3 discovery
- [ ] Recon: mDNS/Bonjour discovery
- [ ] Recon: UPnP/SSDP discovery
- [ ] Recon: OUI manufacturer lookup
- [ ] Pulse: Uptime monitoring (ICMP, TCP, HTTP)
- [ ] Pulse: Alerting (email, webhook)
- [ ] Pulse: Metrics history and graphs
- [ ] Scout: Linux agent (x64, ARM64)
- [ ] Dashboard: monitoring views, graphs, alerts

### Phase 3: Remote Access

**Goal:** Browser-based remote access to any device.

- [ ] Gateway: SSH-in-browser via xterm.js
- [ ] Gateway: HTTP/HTTPS reverse proxy via Go stdlib
- [ ] Gateway: RDP/VNC via Apache Guacamole (Docker)
- [ ] Vault: Encrypted credential storage
- [ ] Vault: Per-device credential assignment
- [ ] Vault: Auto-fill credentials for remote sessions
- [ ] Dashboard: remote access launcher, session management

### Phase 4: Extended Platform

**Goal:** IoT awareness, cross-platform, API integrations.

- [ ] MQTT broker integration (MQTTnet/Eclipse Paho)
- [ ] Home Assistant API integration
- [ ] Scout: macOS agent
- [ ] Scout: Lightweight IoT agent
- [ ] API: Public REST API with authentication
- [ ] Multi-site support
- [ ] User roles and permissions (RBAC)
- [ ] Audit logging

---

## Competitive Positioning

### Market Gap

No existing open-source tool combines all five capabilities with a commercially friendly license:

1. Device discovery (network scanning, SNMP, mDNS)
2. Monitoring (uptime, metrics, alerting)
3. Remote access (RDP, SSH, HTTP)
4. Credential management (secure vault, per-device)
5. IoT/home automation awareness (MQTT, smart devices)

### Closest Competitors

| Tool | Gap vs NetVantage |
|------|-------------------|
| Zabbix/Checkmk | No remote access, no credentials, GPL license |
| MeshCentral | Weak discovery, no monitoring |
| Guacamole | Remote access only |
| Domotz | Proprietary, cloud-dependent, $21/site/month |
| Uptime Kuma | Monitoring only, no agents |

---

## Commercialization (Future)

### Licensing

- **Core:** Apache 2.0 (always free, always open-source)
- **Premium modules:** Proprietary license for enterprise features

### Freemium Tiers

| Tier | Devices | Features |
|------|---------|----------|
| Community (Free) | Up to 10 | Discovery, monitoring, basic alerts |
| Personal | Up to 50 | + Remote access, credentials |
| Professional | Up to 250 | + Multi-site, API, advanced alerts |
| Business | Unlimited | + Multi-user, RBAC, SSO, audit logging |

---

## Non-Functional Requirements

### Performance

- Server handles 1,000+ devices with < 100ms API response times
- Agent CPU usage < 1% idle, < 5% during metric collection
- Agent memory usage < 20MB
- Dashboard loads in < 2 seconds

### Security

- All agent communication encrypted (mTLS)
- Credentials encrypted at rest (AES-256-GCM)
- No default credentials
- API authentication required (JWT tokens)
- CORS properly configured
- Input validation at all API boundaries

### Deployment

- Single binary server (Go)
- Single binary agent (Go, cross-compiled)
- Docker Compose for full stack (server + Guacamole + database)
- Configuration via YAML file + environment variables

### Reliability

- Graceful shutdown on SIGTERM/SIGINT
- Automatic agent reconnection with exponential backoff
- Database migrations via embedded SQL
- Health check endpoint for orchestrators
