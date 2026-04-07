---
phase: delivery
updated: 2026-04-07T09:00:00Z
updated_by: claude-code
---

# SubNetree -- Current State

## Phase

v0.6.3 release pending (release-please PR #529 open).
All CI blockers resolved. Active UI/topology improvements in review.

## Pending

- PR #529: release-please v0.6.3 (auto-managed)
- PR #555: Device hostname rename (#546)
- PR #556: CODEOWNERS fix (#528)
- PR #557: Dockerfile Go version pin
- PR #562: Gateway link inference + unified toolbar + UI fixes (#559, #560)

## Next Actions

- #561: Collapse unclassified devices in topology (critical UX improvement)
- Merge pending PRs after CI green
- Phase 1A Docker Desktop validation (#493) -- partially complete, QC passed

## Recently Completed

- Fixed govulncheck CI blocker (#552): Go 1.25.8, grpc, go-sdk, sqlite
- Batch dep bumps (#554): grpc v1.80.0, x/crypto, x/net, x/time
- Merged 10 Dependabot + CI PRs, closed 10 superseded
- Device hostname rename feature (#546 -> PR #555)
- CODEOWNERS repair (#528 -> PR #556)
- Topology: gateway link inference, unified toolbar, dashed inferred edges
- Monitoring: wider trend sparklines with table-fixed layout
- RouteErrorBoundary for chunk load errors
- Docker QC testing: all endpoints pass, 101 MiB memory, persistence OK

## Queued (Roadmap)

- #489: Ansible dynamic inventory plugin
- #487: community engagement launch prep
- #499: content capture for community launch
- #493-#498: deployment validation phases
- #280, #286, #289: future roadmap (Phase 4+)

## Start Here (Cold Start Protocol)

1. Read this file
2. Call `samverk get_digest --since 168h` if MCP is configured
3. Read open issues if relevant to the task
4. Proceed -- do not ask the user to explain project state
