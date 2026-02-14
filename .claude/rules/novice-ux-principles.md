# Documentation UX Principles

SubNetree uses a **three-tier documentation model** proven by high-adoption open-source projects (Home Assistant, Grafana, Traefik, Uptime Kuma). Each tier has a distinct purpose, audience, and scope.

## Three-Tier Model

| Tier | Surface | Purpose | Audience | Target Length |
| ---- | ------- | ------- | -------- | ------------- |
| 1 | **README.md** | Hook, Quick Start, links out | Everyone (30 seconds) | ~2,000 words max |
| 2 | **Docs site** (MkDocs Material, GitHub Pages) | Guides, tutorials, config reference, troubleshooting | Users by skill level | Unlimited, searchable |
| 3 | **In-repo `/docs/`** | Requirements, ADRs, internal design | Contributors only | As needed |

### Tier 1: README

The README is a landing page, not a manual. Its job is to answer three questions in 30 seconds: What is this? Why should I care? How do I try it?

- Badges, one-sentence tagline, 1-2 screenshots/GIFs
- "Why SubNetree?" value proposition (bullets, not paragraphs)
- Single copy-paste Quick Start command
- Feature highlights in user-benefit language
- Links to docs site for everything else
- License clarity
- Developer content below a visible separator (`--- For Contributors ---`)

**Rule: If content doesn't help someone decide to install or get running, it belongs on the docs site, not the README.**

### Tier 2: Docs Site

The docs site is organized by skill level in the sidebar navigation:

- **Getting Started** (beginners): Install, first scan, dashboard tour
- **User Guide** (intermediate): Modules, configuration, alerts, vault, themes
- **Operations** (advanced): Backup, upgrade, troubleshooting, platform-specific notes
- **Developer Guide** (contributors): Architecture, plugin SDK, API reference, ADRs

Within each page, use these patterns for mixed-level audiences:

- **Tabbed content** for install method (Docker / Binary / Source) and platform (Linux / macOS / Pi)
- **Collapsible `<details>` blocks** for explanatory "why" content -- advanced users see a clean page, novices expand for context
- **TL;DR blocks** at the top of long guides -- 2-3 lines with the essential command or config
- **Reference tables after prose** -- narrative explanation followed by a compact scannable table

### Tier 3: In-Repo Docs

`/docs/requirements/`, `/docs/adr/`, internal design documents. These are for contributors and Claude Code sessions. They don't need to be beginner-friendly -- technical precision matters here.

## Content Rules

These apply across all three tiers:

1. **User-benefit language over technical specs.** Feature lists describe what the user gains. "Alerts you when device behavior changes" not "EWMA baselines with Z-score anomaly detection." Technical terms get parenthetical explanations or glossary links.

2. **One recommended path first.** When multiple options exist, present the recommended option prominently. Alternatives go in tabs, collapsible blocks, or secondary sections.

3. **Show expected outcomes.** After every setup step, describe what the user should see. "After running this command, open http://your-server-ip:8080 and you should see the setup wizard."

4. **Platform-aware.** Call out when behavior differs on macOS, Windows, Raspberry Pi, NAS, or LXC. Use tabs or callout boxes, not inline paragraphs.

5. **Short paragraphs + bullets.** Scannable, not dense. No paragraph should exceed 3-4 sentences.

## UI/UX Rules

6. **Progressive disclosure.** Simple by default, advanced on demand. Don't show every option at once. Use expandable sections, tabs, or "Advanced" toggles.

7. **Error messages should suggest fixes.** "No devices found" should link to troubleshooting or suggest checking network mode. Never show a bare error without guidance.

8. **First-run experience is critical.** The path from install to seeing your first device scan should be frictionless. Every extra click or decision point is a potential drop-off.

## API/Config Rules

9. **Sensible defaults for everything.** A user should be able to run SubNetree with zero configuration and get a useful result. Config files are for customization, not requirements.

10. **Comment example configs heavily.** Every setting should explain what it does, what the default is, and when you'd change it. Use plain language, not just the field name.

## Litmus Tests

**README:** "Can someone decide whether to install SubNetree within 30 seconds of landing on this page?"

**Docs site (beginner):** "Would a homelab user who found this on Reddit at 11pm understand this without Googling?"

**Docs site (advanced):** "Can an experienced sysadmin find the answer they need within 10 seconds of opening this page?"

**In-repo docs:** "Does this give a contributor enough context to implement correctly?"
