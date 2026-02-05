---
title: Installation
weight: 1
---

SubNetree can be installed from source, as a standalone binary, or via Docker.

## From Source

Requires Go 1.25+ and Make.

```bash
git clone https://github.com/HerbHall/subnetree.git
cd subnetree
make build
```

Binaries are output to the `bin/` directory.

## Standalone Binary

Download the latest release from [GitHub Releases](https://github.com/HerbHall/subnetree/releases).

{{< callout type="info" >}}
Pre-built binaries will be available starting with the first tagged release. SubNetree is currently in Phase 1 development -- building from source is the recommended method.
{{< /callout >}}

## Docker

```bash
docker run -d \
  --name subnetree \
  -p 8080:8080 \
  -p 9090:9090 \
  -v subnetree-data:/data \
  ghcr.io/herbhall/subnetree:latest
```

## Docker Compose

```yaml
version: '3.8'
services:
  subnetree:
    image: ghcr.io/herbhall/subnetree:latest
    ports:
      - "8080:8080"   # HTTP API + Dashboard
      - "9090:9090"   # gRPC (Scout agents)
    volumes:
      - subnetree-data:/data
    restart: unless-stopped

volumes:
  subnetree-data:
```

{{< callout type="info" >}}
Docker images will be published starting with the first tagged release. The above examples show the intended deployment method.
{{< /callout >}}

## System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 1 core | 2+ cores |
| RAM | 256 MB | 512 MB+ |
| Disk | 100 MB + data | 1 GB+ |
| OS | Linux, Windows, macOS | Linux (production) |

## Next Steps

- [Configuration reference](../configuration) -- customize ports, database, logging
- [Architecture overview](/docs/architecture) -- understand the system design
