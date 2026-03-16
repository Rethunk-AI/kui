# KUI PRD

Core product reference. Decisions: [prd/decision-log.md](prd/decision-log.md). Architecture: [prd/architecture.md](prd/architecture.md). Stack: [prd/stack.md](prd/stack.md).

---

## §1 Overview

KUI (KVM User Interface) is a web-based interface for users who prefer a UI over CLI to manage KVM VMs via libvirt. Multi-host, single-tenant, 1–2 users. MVP: full flow — create from template (pool+path, VM templates, clone), lifecycle, console, host selector; single admin; real-time status. See [prd/decision-log.md](prd/decision-log.md) §§1–4 for binding decisions.

---

## §2 Open Assumptions (to be resolved in spec)

- Database schema — SQLite: users, preferences, vm_metadata (exact columns in spec). Git: templates, audit. Templates in git; audit diffs in git.
- Frontend framework — defer to spec; Winbox.js compatibility is a constraint.
- Console proxy design — prefer URL-based (wss://kui/...); proxy implementation may be required; needs exploration in spec.
- Session/token storage — research OAuth 2.1 OIDC/SSO standards for Authentik; follow token storage standards (Credential API, etc.). Defer exact mechanism to spec.
- Audit git storage — MVP from day one; git for diffs per entity changeset; exact structure in spec.
Resolved (decision-log): VM list/console UX (Winbox.js Canvas). Phase 15: config path (/etc/kui/config.yaml), first-run (config/DB missing or write access), setup wizard (writes YAML, user restarts, KUI drops write access), console selection (per-VM preference + libvirt fallback), clone (automatic; stream-only OK for MVP), template from VM (domain XML + source disk ref), pool validation (libvirt API preferred). Maintenance/recovery: KUI maintenance mode (defer to spec), degraded (host-offline covers it), KUI recovery (v3 backup/restore); VM stuck detection + escalating recovery, orphan bulk + conflict resolution, domain XML edit — all v2 (decision-log §0 A15–A22, §2, §4).

---

## §3 Execution Order

1. **Scaffold cleanup** — done.
2. **Research** — done. [research/kui-libvirt-research.md](research/kui-libvirt-research.md).
3. **Research (console)** — findings in decision-log §3; formal research doc optional.
4. **Spec** — full spec: schema, API, flows, deployment, component boundaries. Split into multiple specs (target <800 lines or <10 tasks per spec).
5. **Spike** — PoC (libvirt + test driver).

---

## §4 References

| Doc | Content |
|-----|---------|
| [prd/decision-log.md](prd/decision-log.md) | §1 Formal decisions; §2 Canonical decisions; §3 Console protocol; §4 Inquisition findings |
| [prd/architecture.md](prd/architecture.md) | §1 System overview; §2 Components; §3 Data flow; §4 Deployment |
| [prd/stack.md](prd/stack.md) | §1 Backend; §2 Database; §3 Frontend; §4 Config |
| [prd/backlog.md](prd/backlog.md) | v2/v3 prioritized backlog (shortcuts, a11y, stuck VM, orphan bulk, XML edit, backup) |
| [research/kui-libvirt-research.md](research/kui-libvirt-research.md) | Libvirt API, qemu+ssh, test driver, web UI comparison |
| [research/winbox-ux-research.md](research/winbox-ux-research.md) | Winbox.js — Canvas engine for VM list/console |
| [research/xyflow-canvas-ui-research.md](research/xyflow-canvas-ui-research.md) | xyflow deferred for topology/infrastructure maps (not Canvas) |

---

## §5 Competitive Context

Cockpit (libvirt-dbus, virt-install), WebVirtCloud (Python/Django, noVNC), Kimchi (Wok plugin). KUI differentiator: Go, direct libvirt API, simpler scope, single-tenant, Authentik-ready.
