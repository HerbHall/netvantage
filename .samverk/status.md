---
phase: delivery
updated: 2026-04-07T10:30:00Z
updated_by: claude-code
---

# SubNetree -- Current State

## Phase

v0.6.3 release pending (release-please PR #529 open, auto-managed).
All feature PRs merged. Zero open PRs besides release-please.

## Pending

- PR #529: release-please v0.6.3 (auto-managed, merges when ready)

## Next Actions

- #493-#498: deployment validation phases (human hardware work)
- #487: community engagement launch prep
- #499: content capture for community launch
- #489: Ansible dynamic inventory plugin (Phase 3+)

## Recently Completed

Session 2026-04-07 (massive catchup):

- Fixed govulncheck CI blocker (#552): Go 1.25.8, grpc, go-sdk, sqlite
- Batch dep bumps (#554): grpc v1.80.0, x/crypto, x/net, x/time
- Merged 10 Dependabot + CI PRs, closed 10 superseded
- Device hostname rename (#546 -> merged via #555)
- CODEOWNERS repair (#528 -> merged via #556)
- Dockerfile Go version pin (merged via #557)
- Topology: gateway link inference + hierarchical edges (#559 -> merged via #562)
- Topology: unified toolbar, dashed inferred edges (#558 -> merged via #562)
- Topology: collapse unclassified devices (#561 -> merged via #563)
- Topology: zoom limits for large device counts (#560 -> merged via #562)
- Monitoring: wider trend sparklines with table-fixed layout
- RouteErrorBoundary for chunk load errors
- Docker QC testing passed: all endpoints, 101 MiB memory, persistence OK

## Queued (Roadmap)

- #280: multi-tenancy support (Phase 4+)
- #286: IPv6 scanning (Phase 4+)
- #289: PostgreSQL + TimescaleDB (Phase 4+)

## Start Here (Cold Start Protocol)

1. Read this file
2. Call `samverk get_digest --since 168h` if MCP is configured
3. Read open issues if relevant to the task
4. Proceed -- do not ask the user to explain project state
