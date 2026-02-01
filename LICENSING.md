# NetVantage Licensing

NetVantage uses a split licensing model to balance commercial protection with
ecosystem openness.

## Quick Summary

| What you want to do | License | Cost |
|---------------------|---------|------|
| Use NetVantage at home or in a home lab | BSL 1.1 (permitted) | Free |
| Use NetVantage for personal/educational purposes | BSL 1.1 (permitted) | Free |
| Use NetVantage in production (non-competing) | BSL 1.1 (permitted) | Free |
| Build plugins using the Plugin SDK | Apache 2.0 | Free |
| Build tools that integrate via gRPC/Protobuf | Apache 2.0 | Free |
| Offer NetVantage as a hosted/managed service | Commercial license required | Contact us |
| Embed NetVantage in a competing commercial product | Commercial license required | Contact us |
| Resell or white-label NetVantage | Commercial license required | Contact us |

## License Details

### Core Software (BSL 1.1)

The NetVantage server, Scout agent, built-in modules, dashboard, and all code
not covered by a more specific LICENSE file are licensed under the
[Business Source License 1.1](LICENSE).

**Key terms:**

- You may copy, modify, create derivative works, redistribute, and make
  non-production use of the software.
- Production use is permitted as long as it does not compete with NetVantage
  commercial offerings (e.g., offering it as a hosted service, reselling it,
  or embedding it in a competing product).
- Personal, home-lab, and educational use is always permitted.
- On **2030-02-01** (or 4 years after each version's release, whichever comes
  first), the code automatically converts to the **Apache License 2.0**.

### Plugin SDK (Apache 2.0)

The following directories are licensed under the
[Apache License 2.0](pkg/plugin/LICENSE) to encourage ecosystem growth:

- `pkg/plugin/` -- Plugin interfaces and types
- `pkg/roles/` -- Role interfaces (abstract capabilities)
- `pkg/models/` -- Shared data models
- `api/proto/` -- gRPC/Protobuf service definitions

You may freely use these packages to build plugins, integrations, and tools
that work with NetVantage. There are no restrictions on commercial use of
code you write against these interfaces.

### Community Plugins (Apache 2.0 default)

Plugins in `plugins/community/` use the Apache 2.0 license by default.
Plugin authors may choose a different permissive license for their
contributions.

## Dependency License Elections

NetVantage uses the following dual-licensed dependencies under their
permissive license options:

- **eclipse/paho.mqtt.golang**: Used under EDL-1.0 (BSD-3-Clause), not EPL-2.0
- **hashicorp/go-plugin**: MPL-2.0 (used as unmodified library)

## Contributor License Agreement

All contributions to NetVantage require signing a Contributor License
Agreement (CLA). This is handled automatically via CLA Assistant when you
open your first pull request. See [.github/CLA.md](.github/CLA.md) for
the full CLA text.

## Trademark

"NetVantage" is a trademark of Herb Hall. See [TRADEMARK.md](TRADEMARK.md)
for usage guidelines.

## Commercial Licensing

For commercial licensing inquiries, contact: licensing@netvantage.io

## FAQ

**Q: Can I use NetVantage to monitor my company's network?**
A: Yes. Production use is permitted as long as you are not offering NetVantage
itself as a service to others or competing with NetVantage commercial offerings.

**Q: Can I build and sell a plugin for NetVantage?**
A: Yes. The Plugin SDK is Apache 2.0. You can build and sell plugins with no
restrictions.

**Q: Can I fork NetVantage and offer it as a hosted service?**
A: Not under the BSL 1.1 license. You would need a commercial license. After
the Change Date (4 years), the code converts to Apache 2.0 and this
restriction is removed.

**Q: What happens after the Change Date?**
A: The specific version released 4 years ago becomes Apache 2.0. Newer
versions remain under BSL 1.1 until their own Change Date passes. This means
there is always an open-source version available, just not the latest.
