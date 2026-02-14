## Documentation Requirements

### Three-Tier Documentation Model

SubNetree follows the documentation pattern used by high-adoption open-source projects (Home Assistant, Grafana, Traefik, Uptime Kuma). Each tier has a distinct purpose and audience.

| Tier | Surface | Purpose | Audience |
| ---- | ------- | ------- | -------- |
| 1 | **README.md** | Hook, Quick Start, links out | Everyone (30 seconds) |
| 2 | **Docs site** (MkDocs Material on GitHub Pages) | Guides, tutorials, config reference | Users by skill level |
| 3 | **In-repo `/docs/`** | Requirements, ADRs, internal design | Contributors only |

**Key rule:** README length inversely correlates with project maturity. Keep the README under ~2,000 words. Everything else lives on the docs site.

See `.claude/rules/novice-ux-principles.md` for full content rules, patterns, and litmus tests.

### Tier 1: README (~2,000 words max)

The README is a landing page, not a manual. It answers: What is this? Why should I care? How do I try it?

```text
# SubNetree
[badges: CI | Go version | License | Latest Release | Docker Image]

One-sentence tagline + 1-2 screenshots/GIFs

## Why SubNetree?
Bullet-point value prop + comparison table

## Screenshots
2-3 key screens (dashboard, topology, device detail)

## Quick Start
Single copy-paste docker command (host networking recommended)
Bridge mode in collapsible <details> block

## Features
User-benefit language, not technical specs
Link to docs site for details

## Documentation
Link to docs site with section overview

## Community + Support
Links to Discussions, Issues, Discord

--- For Contributors ---

## Building from Source
Build, test, lint commands

## Contributing
Link to CONTRIBUTING.md

## License
BSL 1.1 clarity
```

### Tier 2: Docs Site (MkDocs Material)

Organized by skill level in the sidebar. Hosted on GitHub Pages at `docs.subnetree.io` or `herbhall.github.io/subnetree`.

#### Getting Started (beginners)

| Page | Content |
| ---- | ------- |
| Installation | Tabbed: Docker / Binary / Source, per-platform notes |
| First Scan | Step-by-step from login to seeing devices |
| Dashboard Tour | What each panel shows, how to navigate |
| FAQ | Common questions from new users |

#### User Guide (intermediate)

| Page | Content |
| ---- | ------- |
| Discovery | Scan types (ARP, ICMP, SNMP), scheduling |
| Monitoring | Checks, alerts, notification channels |
| Topology | Network map, layout, export |
| Credential Vault | Storing and using device credentials |
| Remote Access | SSH-in-browser, HTTP proxy |
| Themes | Built-in themes, customization |
| Configuration Reference | All YAML keys, env vars, defaults |

#### Operations (advanced)

| Page | Content |
| ---- | ------- |
| Backup and Restore | CLI commands, scheduling |
| Upgrading | Version migration, breaking changes |
| Troubleshooting | Expanded scenarios with fixes |
| Platform Notes | UnRAID, Proxmox, Synology, Pi, NAS, macOS/Windows |
| Performance Tuning | Profiles, concurrency, resource limits |
| Scout Agent | Deployment, enrollment, mTLS |

#### Developer Guide (contributors)

| Page | Content |
| ---- | ------- |
| Architecture | System design, plugin system, data flow |
| Plugin Development | Creating modules, role interfaces, SDK |
| API Reference | REST endpoints, auto-generated from OpenAPI |
| gRPC Protocol | Agent-server communication, proto definitions |
| Contributing | Development setup, code style, testing |

### Tier 3: In-Repo Documentation

| Location | Content | Audience |
| -------- | ------- | -------- |
| `docs/requirements/` | Product requirements (28 files) | Contributors, Claude Code |
| `docs/adr/` | Architecture Decision Records | Contributors |
| `docs/guides/` | Deployment guides (Tailscale, etc.) | Transitional (move to docs site) |
| `.claude/` | Claude Code project config, rules | Claude Code sessions |

### Community Health Files (GitHub)

Standard files that GitHub recognizes and surfaces in the repository UI.

| File | Description | Status |
| ---- | ----------- | ------ |
| `CONTRIBUTING.md` | Development setup, PR process, code style, testing, CLA | Exists |
| `SECURITY.md` | Vulnerability disclosure process | Exists |
| `CODE_OF_CONDUCT.md` | Contributor Covenant v2.1 | Exists |
| `.github/pull_request_template.md` | PR checklist | Exists |
| `.github/ISSUE_TEMPLATE/` | Bug report, feature request, plugin idea | Exists |
| `.github/FUNDING.yml` | Sponsor button | Exists |
| `SUPPORTERS.md` | Financial supporter recognition | Exists |
| `LICENSING.md` | Human-readable licensing explanation | Exists |

### MkDocs Material Configuration

```yaml
# mkdocs.yml (target configuration)
site_name: SubNetree Documentation
site_url: https://herbhall.github.io/subnetree
repo_url: https://github.com/HerbHall/subnetree

theme:
  name: material
  features:
    - navigation.tabs         # Top-level skill-level tabs
    - navigation.sections     # Grouped sidebar
    - navigation.expand       # Auto-expand current section
    - search.suggest          # Search autocomplete
    - content.tabs.link       # Linked content tabs (Docker/Binary/Source)
    - content.code.copy       # Copy button on code blocks

markdown_extensions:
  - admonition               # Callout boxes (tip, warning, note)
  - pymdownx.details         # Collapsible sections
  - pymdownx.tabbed:         # Tabbed content (platform/method switching)
      alternate_style: true
  - pymdownx.superfences     # Fenced code blocks with titles

nav:
  - Home: index.md
  - Getting Started:
      - Installation: getting-started/installation.md
      - First Scan: getting-started/first-scan.md
      - Dashboard Tour: getting-started/dashboard-tour.md
      - FAQ: getting-started/faq.md
  - User Guide:
      - Discovery: guide/discovery.md
      - Monitoring: guide/monitoring.md
      - Topology: guide/topology.md
      - Credential Vault: guide/vault.md
      - Remote Access: guide/remote-access.md
      - Configuration: guide/configuration.md
  - Operations:
      - Backup & Restore: ops/backup-restore.md
      - Upgrading: ops/upgrading.md
      - Troubleshooting: ops/troubleshooting.md
      - Platform Notes: ops/platforms.md
      - Scout Agent: ops/scout-agent.md
  - Developer Guide:
      - Architecture: dev/architecture.md
      - Plugin Development: dev/plugins.md
      - API Reference: dev/api-reference.md
      - Contributing: dev/contributing.md
```

### Migration Path

1. **Phase 1:** Set up MkDocs Material scaffolding, deploy to GitHub Pages
2. **Phase 2:** Trim README to ~2,000 words, add docs site links
3. **Phase 3:** Migrate existing `docs/guides/` content to docs site pages
4. **Phase 4:** Expand with tutorials, auto-generated API docs from OpenAPI spec
