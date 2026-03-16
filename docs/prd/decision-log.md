# KUI Decision Log

Binding technical decisions. Each topic appears once; later phases supersede earlier when conflicting. Core PRD: [prd.md](../prd.md).

---

## §0 Assumptions (resolved via Inquisition)

*Explicit assumptions. Label each; resolved through questioning.*

| ID | Assumption | Status |
|----|------------|--------|
| A1 | KUI targets Linux hosts with libvirt; no Windows/macOS support | Verified |
| A2 | Users have basic sysadmin skills (understand VM, storage, network concepts) | Verified |
| A3 | 1–2 concurrent users is sufficient for single-tenant use case | Verified |
| A4 | Template-first VM creation is the right UX (vs manual XML); MVP: pool+path + clone; template-based create in v2 | Modified |
| A5 | Authentik SSO is post-MVP; local SQLite auth is MVP | Modified — MVP if trivial (simple OIDC &lt;1 day OR reuse existing Authentik pattern); otherwise post-MVP |
| A6 | HTTP/URL template sources are post-MVP; local pools/paths only for MVP | Verified |
| A7 | Systemd is primary deployment; Docker only after MVP stable | Verified |
| A8 | SQLite is sufficient for 1–2 users (no PostgreSQL needed) | Verified |
| A9 | SPICE in browser (spice-html5 / spice-web-client) will meet console needs | Verified |
| A10 | No migration path needed — greenfield only | Verified (project rule) |
| A11 | VM templates are user-defined in SQLite; sharable when RBAC added (future) | Verified |
| A12 | KUI proxies all console traffic (browser → KUI → libvirt host) | Verified |
| A13 | Session: JWT (stateless) | Verified |
| A14 | SSH keys in config (YAML) only; env override for Docker | Verified |
| A15 | KUI maintenance mode (upgrade, DB migration) — defer to spec when designing deployment/upgrade flows | Verified |
| A16 | KUI degraded state — host-offline behavior (show reachable VMs only) is sufficient; no separate degraded mode | Verified |
| A17 | KUI recovery (DB/config corruption) — v3 backup/restore covers it; no additional MVP recovery flow | Verified |
| A18 | VM maintenance mode or quarantine — defer to v2; libvirt pause is sufficient for MVP | Verified |
| A19 | Stuck VM detection — libvirt state only (e.g., domain running but qemu process gone) | Verified |
| A20 | Stuck VM recovery actions — escalating: force stop → force destroy → undefine | Verified |
| A21 | Orphan conflict resolution — display name collision, UUID on multiple hosts, claimed-but-host_id-mismatch | Verified |
| A22 | Domain XML edit — defer to spec (full vs guided) | Verified |

---

## §1 Formal Decisions (from Research)

### Remote Libvirt Credentials

- Transport: qemu+ssh — all libvirt API calls tunneled over SSH
- Auth: SSH public key only. No password storage. Use keyfile URI param: qemu+ssh://user@host/system?keyfile=/path/to/private/key
- Setup: Client generates key pair; add public key to remote ~/.ssh/authorized_keys. Use ssh-agent or keyfile to avoid prompts
- Prerequisites: Remote host must have libvirtd running and nc (netcat) installed. No libvirt daemon config changes — auth delegated to sshd
- Source: libvirt.org/remote.html; docs/research/kui-libvirt-research.md

### Test Driver for CI

- URI: test:///default (built-in) or test:///path/to/config.xml (custom)
- Behavior: Per-process fake hypervisor; all state in memory; no real KVM or hardware
- Use: Unit tests for libvirt integration in CI without KVM. Both libvirt.org/go/libvirt and digitalocean/go-libvirt support it
- Source: libvirt.org/drvtest.html; docs/research/kui-libvirt-research.md

### Go Libvirt Bindings

- Package: libvirt.org/go/libvirt (CGo) — official bindings, comprehensive API
- XML: libvirt.org/libvirt-go-xml for domain/network/storage pool structs
- Rationale: Systemd-first deployment; build on host with libvirt-dev

---

## §2 Canonical Decisions (by topic)

| Topic | Decision |
|-------|----------|
| API style | REST/JSON internal first; formal API later. MVP: informal — code is the contract. Formalize when needed; no fixed timeline |
| Audit retention | Unlimited by default; configurable retention optional. Git-based audit from MVP. Notification panel audit log in v2 |
| Audit scope | Wizard completion, VM config changes, VM lifecycle, auth events |
| Audit verbosity | Full before/after for config changes. Git for diffs per entity changeset (MVP from day one; simpler than SQLite for diff storage) |
| Auth | Local auth first (SQLite users); Authentik SSO later. MVP: single admin only; Authentik if trivial |
| Backend stack | Go — libvirt.org/go/libvirt |
| Config | YAML file primary; hosts required; vm_defaults (CPU, RAM, network) for pool+path create; others have defaults. Setup wizard validates hosts. Git path configurable; default /var/lib/kui/. Default path: /etc/kui/config.yaml; env for Docker override |
| VM templates | Stored in git; sharable when RBAC added. MVP: creation exists (de-emphasized); create VM from template v2 |
| Console protocol | MVP: noVNC + xterm.js; SPICE in v2. Selection: per-VM preference in vm_metadata first; else libvirt domain XML → noVNC → xterm.js |
| Console proxy | KUI proxies all console traffic (browser → KUI → libvirt host). For remote: KUI opens SSH tunnel and forwards console stream. KUI can run anywhere. Prefer URL-based; needs exploration in spec |
| Concurrent users | 1–2 users |
| Database | SQLite: users, preferences, vm_metadata. Git: templates (full audit chain) + audit (diffs per entity). Git path configurable in YAML; default e.g. /var/lib/kui/ |
| Audit structure | Single audit table with event_type column |
| Default host | Config defines default; user override in preferences. Host selector: global default + contextual override (create, clone, console) |
| Deployment | Systemd first; Docker only after MVP stable. Target: both local and remote libvirt hosts |
| Docs scope | SDD-driven; specs/plans are the docs; README for quick start. Specs: split into multiple specs; target <800 lines or <10 tasks per spec |
| Empty state | First-run checklist: MVP "Create VM" only (pool+path or clone); "Create Template" in v2. Template creation exists in MVP but de-emphasized. Visible when VM list empty; completed when done OR dismissed |
| Error handling | Toast with user-friendly message. Optional "Show details" expands to sanitized technical summary (no stack trace). Full stack/raw only in backend log |
| First-run | Setup wizard when: config missing, DB missing. Prompts for admin + host; skip or create template later. Onboarding checklist (separate) guides to Create Template, Create VM. Wizard writes config once; config read-only at runtime |
| Frontend | Recommend based on noVNC/SPICE + real-time updates (WebSocket/SSE). Framework defer to spec; Winbox.js compatibility is a constraint |
| i18n | English only |
| KVM interaction | Full: lifecycle (create, start, stop, destroy), provisioning (full flow from base image to running VM), monitoring |
| Libvirt connection | Multi-host from day one. Host selector: global default + contextual override. Per-host URI + keyfile in config. Connection strategy: simplest approach (defer to spec) |
| License | MIT |
| Local auth | SQLite users table. MVP: single admin account only; no RBAC |
| Logging | slog (log/slog) — structured, Go stdlib |
| MVP lifecycle | Full: list, create, start, stop (graceful then force), pause, resume, destroy |
| MVP scope | Create (pool+path, clone), lifecycle, console, host selector; single admin; real-time status. Template-based create in v2 |
| Multi-tenancy | Single user / single org. No multi-tenancy |
| Notifications | Toast for transient feedback. MVP: alerts panel (host offline, clone/create failed, connection errors, VM state changes). Alerts in-memory only; disappear on refresh. Audit log in v2 |
| Open source | Yes — may be shared or reused |
| Persistence scope | Full — auth, audit log, preferences (default host + list view: columns, sort), VM templates/metadata |
| Read-only mode | Post-MVP. MVP: admin-only |
| Real-time updates | v1: VM state + host online/offline. v2: clone/create progress (%, ETA, stage). Prefer single WebSocket; else WebSocket for console, SSE for status |
| Role model | Three: viewer (read-only), operator (lifecycle only), admin (full + config). MVP: admin only |
| Session | JWT (stateless). Storage: research OAuth 2.1 OIDC/SSO standards for Authentik; follow token storage standards (e.g. Credential API if applicable). Defer exact mechanism to spec |
| Session timeout | Configurable in YAML; long default (24h or until browser close) |
| SSH keys | Config (YAML) only; env override for Docker |
| Slab relationship | KUI is standalone. Works on any system with libvirt API |
| Host offline | Show VMs from reachable hosts only; unreachable hosts as "offline" with error state. Real-time updates restore when available |
| Orphan domains | Show all libvirt domains; offer to import/claim. Claim = add to tracked list; host_id from host where orphan was discovered |
| Storage/networks MVP | Use existing only — no storage/network management in MVP |
| Pool/path validation | Validate pool exists and active via libvirt; validate path/volume in pool if user-specified |
| Clone implementation | User selects source VM + target host + target pool; name optional. YAML defaults: default_host, default_pool, default_name_template. MVP: {source} only; {date}, {timestamp} in v2. User preferences override. Automatic storage/copy/stream; cross-host same format |
| Target distro | Any Linux with libvirt |
| Team | Solo developer |
| Template sources | Pre-existing storage pools or whitelisted paths (local) in config YAML; HTTP templates deferred to post-MVP. template_storage optional; user picks pool at save time if missing; else same pool as source |
| Terminology | "VM" or "Virtual Machine" |
| TLS | KUI can optionally serve TLS; reverse proxy recommended for prod. HTTP for dev. Document both |
| Test strategy | Libvirt test driver (test:///default); no mocks |
| Timeline | Weeks — want something usable soon |
| UI type | Web UI (browser-based) |
| UI complexity | Functional only (forms, tables, minimal styling); desktop-first |
| Setup wizard scope | Admin + host only; skip or create template later. Validate host URI/keyfile by attempting connection before saving config. Onboarding checklist guides step actions. Wizard writes YAML once; config read-only at runtime |
| User skill | Basic sysadmin — understands concepts but prefers UI |
| VM create flow | MVP: pool+path + clone only; template-based create in v2. Pool+path: config vm_defaults (CPU, RAM, network); user overrides at create. Disk: prefer pick existing volume or auto-gen path + size; user can override with explicit path/name. Clone cross-host. VM stop: graceful first; force if needed |
| VM list | Flat; libvirt for domain list + state; vm_metadata for KUI fields. Unclaimed in separate section/tab. Group by month/year. MVP: simple relative-time badge (e.g. "started 2m ago") |
| VM list UI | Winbox.js — standardized for Canvas (draggable VM windows). xyflow deferred for topology/infrastructure maps |
| VM metadata | host_id + libvirt_uuid (composite key), claimed, display_name, optional console preference, last_access; provenance in audit |
| VM rename | KUI display name stored in SQLite |
| VM scale (MVP) | 5–20 VMs — design for this range |
| VM templates | Stored in git (full audit chain). Name + base image required; CPU/RAM/network have defaults. MVP: template creation exists (de-emphasized); create VM from template in v2. Save VM as template: template_storage or user picks; else same pool. Disk naming: MVP {vm_name} only |
| Versioning | v1 = core; v2 = enhancements (shortcuts, a11y); v3 = ops (backup, import/export) |
| Search/filter | Superseded by Canvas — N/A for MVP |
| KUI maintenance mode | Defer to spec — decide when designing deployment/upgrade flows |
| KUI degraded state | Host-offline behavior covers it — show reachable VMs only; no separate degraded mode |
| KUI recovery | v3 backup/restore; first-run wizard + manual restore sufficient until then |
| VM maintenance mode | Defer to v2; libvirt pause/resume sufficient for MVP |
| Stuck VM detection | Libvirt state only — e.g., domain in running but qemu process gone |
| Stuck VM recovery | Escalating actions: force stop → force destroy → undefine. v2 |
| Orphan bulk claim | Bulk claim + conflict resolution. v2 |
| Orphan conflict types | Display name collision; UUID exists on multiple hosts; claimed in KUI but host_id mismatch |
| VM repair (broken config/disk) | Edit domain XML in UI. v2. Full vs guided — defer to spec |

---

## §3 Console Protocol Order of Preference

**MVP:** noVNC + xterm.js. SPICE (Open365/spice-web-client) in v2.

1. **noVNC** — proxied through KUI (browser → KUI → libvirt host). Primary for MVP.
2. **xterm.js** — for bare serial or console access. Fallback when VNC not available.
3. **SPICE** (v2) — Open365/spice-web-client for QXL, audio, clipboard.

---

## §4 Inquisition Findings

*Derived from Inquisition Workflow (15 phases).*

| Topic | Finding |
|-------|---------|
| Accessibility | WCAG 2.1 AAA, v2 only |
| API formalization | Informal — code is the contract |
| Audit retention | Unlimited OK for MVP |
| Audit scope | Wizard completion, VM config changes, VM lifecycle, auth events |
| Audit verbosity | Full before/after for config changes. Git for diffs per entity changeset (MVP from day one; simpler than SQLite for diff storage) |
| Audit structure | Single table with event_type column |
| Auth storage | Research OAuth 2.1 OIDC/SSO for Authentik; follow token storage standards. Defer to spec |
| Backup/restore | Defer to v3 — much after core works |
| Clone implementation | User selects source VM + target host + target pool; name optional. MVP: default_name_template {source} only; {date}, {timestamp} v2. User preferences override |
| Clone scope | Cross-host supported (Host A → Host B) |
| Config templates | VM templates in git; sharable when RBAC added |
| Connection pooling | Defer to spec — Libvirt Connector owns connection management; pooling strategy TBD in spec |
| Config secrets | YAML with env override for Docker |
| Console MVP | (1) Open365 SPICE; (2) noVNC via KUI; (3) xterm.js for serial/console |
| Console proxy | KUI proxies all console traffic (browser → KUI → libvirt host). For remote: KUI opens SSH tunnel and forwards console stream. KUI can run anywhere. Prefer URL-based; needs exploration in spec |
| Default host | Config defines default; user override in preferences. Host selector: global default + contextual override (create, clone, console) |
| Deployment target | Both local and remote from day one |
| Empty state | First-run checklist: MVP "Create VM" only; "Create Template" v2. Visible when VM list empty; completed when done OR dismissed |
| Error handling | Toast + optional "Show details" (sanitized); stack/raw in backend log only |
| Docker priority | Post-MVP — systemd first; Docker only after MVP stable |
| Existing infra | Hosts, pools, networks already in use |
| First-run flow | Force onboarding when: config missing, DB missing. Wizard writes config once; config read-only at runtime |
| Frontend build | Defer to spec; Winbox.js compatibility is a constraint |
| Host config | Named hosts with explicit id, uri, keyfile in YAML; env override for Docker |
| Host credentials | Per-host in config (URI + keypath each) |
| Host offline | Show VMs from reachable hosts only; unreachable hosts as "offline" with error state. Real-time updates restore when available |
| Import/claim flow | Claim = add to tracked list; host_id from host where orphan discovered |
| Mobile support | No — desktop-first |
| Keyboard shortcuts | v2 — see backlog.md |
| Notifications | Toast for transient. MVP: alerts (host offline, clone/create failed, connection errors, VM state changes); in-memory only. Audit log v2 |
| Next artifact | Full spec: schema, API, flows, deployment, component boundaries. Split into multiple specs (<800 lines or <10 tasks each) |
| Orphan handling | Show all libvirt domains; offer to import/claim. host_id from discovery host |
| Pool/path validation | Validate pool exists and active via libvirt; validate path/volume in pool if user-specified |
| Template structure | Templates in git. MVP: creation exists (de-emphasized); create VM from template v2. Save as template: template_storage or user picks; else same pool |
| Template sharing | Future: when RBAC is added (viewer/operator/admin) |
| Preferences scope | Default host + list view options (columns, sort). Single row per user_id; JSON blob for list-view options |
| Search/filter | Superseded by Canvas — no longer relevant |
| Versioning | v1 = core; v2 = enhancements; v3 = ops |
| Real-time mechanism | v1: VM state + host online/offline. v2: clone progress (%, ETA, stage). Prefer single WebSocket; else WebSocket for console, SSE for status |
| Session storage | Research OAuth 2.1 OIDC/SSO token storage for Authentik; follow standards. Defer to spec |
| SSH key location | Config (YAML) only |
| Setup wizard scope | Admin + host only; skip or create template later. Onboarding checklist guides step actions. Config read-only at runtime |
| Spike scope | Folded into spec implementation |
| Template sources | Mixed — storage pools, local paths, existing domains |
| Timeline | Aspirational — no hard deadline |
| TLS/HTTPS | KUI can optionally serve TLS; reverse proxy recommended for prod. HTTP for dev. Document both |
| UI complexity | Functional only (forms, tables, minimal styling) |
| User persona | Single admin managing a few VMs; then homelab |
| VM list grouping | Flat — all VMs in one view; do not separate by host. Group by month/year; user-configurable date (default: last-access, fallback: creation); within group: display name |
| VM list UI | Winbox.js standardized for Canvas; xyflow deferred for topology/infrastructure maps |
| Winbox.js | Canvas engine — draggable VM console windows; Golden Layout discarded for this use-case |
| Winbox timing | MVP — implement in v1 |
| xyflow | Deferred — use later for network topology, hardware/infrastructure maps (not Canvas) |
| Config path | Default /etc/kui/config.yaml. hosts required; vm_defaults for pool+path create; setup wizard validates. template_storage optional. Git path configurable |
| Console selection | Per-VM preference in vm_metadata first; else libvirt domain XML → SPICE → noVNC → xterm.js |
| Template from VM | Domain XML + copy of source disk. Copy destination: config template storage pool/path first; else same pool as source |
| VM metadata scope | host_id + libvirt_uuid (composite key), claimed, display_name, console preference, last_access; provenance in audit |
| VM create disk | Pool+path: pick existing volume or auto-gen + size; user can override with explicit path. Template-based create v2 |
| VM rename | KUI display name stored in SQLite |
| VM scale (MVP) | 5–20 VMs — design for this range |
| KUI maintenance mode | Defer to spec |
| KUI degraded | Host offline covers it |
| KUI recovery | v3 backup/restore |
| VM maintenance mode | v2 |
| Stuck VM | Explicit detection (libvirt state) + escalating recovery (force stop → destroy → undefine). v2 |
| Orphan bulk | Bulk claim + conflict resolution (name, UUID, host_id). v2 |
| VM repair | Domain XML edit. v2. Full vs guided defer to spec |
