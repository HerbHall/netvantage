---
title: FAQ
weight: 4
---

Frequently asked questions about SubNetree.

## General

### What is SubNetree?

SubNetree is a modular, self-hosted network monitoring and management platform. It combines device discovery, monitoring, remote access, credential management, and IoT awareness in a single application.

### Who is SubNetree for?

- Home lab enthusiasts
- Prosumers managing home networks
- Small business IT administrators
- Managed service providers (MSPs)

### What's the current status?

SubNetree is in **Phase 1: Foundation** development. The core server, HTTP API, plugin registry, and SQLite database are implemented. See the [Roadmap](/#roadmap) for details on upcoming phases.

### Is there a hosted/cloud version?

No. SubNetree is self-hosted by design. Your data stays on your hardware. There are no plans for a SaaS offering.

## Licensing

### Is SubNetree free?

Yes, for personal, home-lab, educational, and non-competing production use. The core is licensed under [Business Source License 1.1](https://github.com/HerbHall/subnetree/blob/main/LICENSE) with an Additional Use Grant covering these use cases.

### What does the BSL 1.1 license mean?

- **Free** for personal, home-lab, educational, and non-competing production use
- **Commercial license required** if you're competing with SubNetree as a product
- **Automatically converts to Apache 2.0** four years after each release
- This is the same model used by HashiCorp, MariaDB, CockroachDB, and Sentry

### Can I build plugins?

Yes. The Plugin SDK (`pkg/plugin/`, `pkg/roles/`, `pkg/models/`, `api/proto/`) is licensed under **Apache 2.0** with no restrictions. You can build and distribute commercial or open-source plugins freely.

### Can I use SubNetree at my company?

Yes, as long as you're not building a competing network monitoring product. Using SubNetree to monitor your company's infrastructure is an explicitly permitted use case.

## Technical

### What languages/frameworks does SubNetree use?

- **Server**: Go 1.25+
- **Dashboard**: React + TypeScript
- **Agent communication**: gRPC
- **Database**: SQLite (with optional PostgreSQL upgrade path)
- **API**: RESTful JSON with RFC 7807 error responses

### What protocols does SubNetree support?

Currently planned: SNMP (v1/v2c/v3), ICMP, ARP, mDNS, UPnP, and MQTT. Protocol support is being added progressively through the phased roadmap.

### Can I run it on Windows?

Yes. SubNetree compiles and runs on Windows, Linux, and macOS. CI tests run across all three platforms.

### How does the plugin system work?

Every major feature is a plugin that implements role interfaces defined in the SDK. Plugins are registered at startup and managed through a lifecycle system with dependency resolution. See the [Architecture](/docs/architecture) page for details.

## Contributing

### How do I contribute?

See the [Contributing](/docs/contributing) section for guides on reporting bugs, requesting features, and setting up a development environment.

### Do I need to sign a CLA?

Yes. All contributors must sign the Contributor License Agreement before their first PR can be merged. This is automated through a GitHub bot. The CLA ensures a clean IP chain for the project's split licensing model.

### What dependencies are blocked?

GPL, AGPL, LGPL, and SSPL licensed dependencies are not permitted. The CI pipeline includes a license check that enforces this. See `make license-check`.
