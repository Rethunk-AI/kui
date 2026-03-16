# KUI TODO

**Last updated:** 2026-03-16

---

## Verified Current State

| Check | Status |
|-------|--------|
| Build | pass |
| Test | pass |
| Vet | pass |
| Spec integrity | pass |
| Doc integrity | pass |

### Active Specs (6)

| spec_id | progress | blockers |
|---------|----------|----------|
| spec-audit-integration | spec ✓; impl MVP (wizard, auth) | — |
| spec-vm-lifecycle-create | spec ✓; impl DONE | — |
| spec-frontend-build | spec ✓; impl DONE | — |
| spec-template-management | spec ✓; impl DONE | — |
| spec-console-realtime | spec ✓; impl DONE | — |
| spec-ui-deployment | spec ✓; impl 0% | — |

### Done Specs (6)

| spec_id | status |
|---------|--------|
| spec-console-realtime | DONE — SSE /api/events, noVNC /vnc, xterm serial /serial |
| spec-frontend-build | DONE — Vite, web/, embed, SPA fallback |
| schema-storage | DONE — moved to specs/done/ |
| spec-libvirt-connector | DONE — moved to specs/done/ |
| spec-application-bootstrap | DONE — moved to specs/done/ |
| api-auth | DONE — moved to specs/done/ |

---

## Remaining Implementation Tasks

### Foundation (Order 1–2)

| Task ID | Spec | Description | Status |
|---------|------|-------------|--------|
| T1 | schema-storage | Scaffold go.mod, internal/config (YAML load, env overrides, validation) | DONE |
| T2 | schema-storage | Implement internal/db (SQLite, schema §2.2) | DONE |
| T3 | schema-storage | Implement internal/git (templates + audit layout) | DONE |
| T4 | spec-libvirt-connector | Implement Connector interface, domain/network/storage ops | DONE |

### Core (Order 3–5)

| Task ID | Spec | Description | Status |
|---------|------|-------------|--------|
| T5 | spec-application-bootstrap | cmd/kui/main.go, config load, middleware, routes, startup/shutdown | DONE |
| T6 | api-auth | Auth service, setup endpoints, JWT middleware, preferences, hosts | DONE |
| T7 | spec-audit-integration | Audit service, integration points (wizard_complete, auth) | DONE |

### Feature (Order 6+)

| Task ID | Spec | Description | Status |
|---------|------|-------------|--------|
| T8a | spec-vm-lifecycle-create | Discovery: GET /api/hosts/{id}/pools, volumes, networks | DONE |
| T8b | spec-vm-lifecycle-create | GET /api/vms list (flat + orphans) | DONE |
| T8c | spec-vm-lifecycle-create | VM detail, lifecycle (start/stop/pause/resume/destroy) | DONE |
| T8d | spec-vm-lifecycle-create | POST /api/vms create, clone, claim | DONE |
| T8e | spec-vm-lifecycle-create | PATCH config edit, vm_config_change audit | DONE |

---

## Security Audit Findings (2026-03-16) — RESOLVED

All findings addressed:
- **High:** Config chmod 0o600; setup/complete only when config missing + setupCompleted flag
- **Medium:** validate-host setup-only + sanitized errors; secure cookies config; login rate limit
- **Low:** jwt_secret required in normal mode

**Additional fixes (router subagent):**
- validate-host: removed `err` from Debug logs (prevents URI/keyfile leakage)
- login: removed username from failed-login Warn logs (prevents enumeration)
- setup idempotency: added `os.Stat(configPath)` check before write (prevents race)

---

## Deferred (Planning Required)

- None. All specs have plans.

---

## Planner Triggers

When T1–T7 are done: create specs for feature specs if needed (plans exist; no new planning required).

---

## Recommended Delegation Order

1. **T1** — schema-storage: go.mod + internal/config
2. **T2** — schema-storage: internal/db
3. **T3** — schema-storage: internal/git
4. **T4** — spec-libvirt-connector: Connector + ops
5. **T5** — spec-application-bootstrap: main, middleware, routes
6. **T6** — api-auth
7. **T7** — spec-audit-integration
8. **T8a** — spec-vm-lifecycle-create: discovery endpoints ✓
9. **T8b** — spec-vm-lifecycle-create: GET /api/vms
10. **T8c** — spec-vm-lifecycle-create: VM detail + lifecycle

---

## Critical Path

```
schema-storage (T1–T3) → spec-application-bootstrap (T5)
spec-libvirt-connector (T4) ─┘
```

## Latest Work

**2026-03-16:** spec-audit-integration T7 complete. Added internal/audit package (RecordEvent, RecordEventWithDiff), wired wizard_complete in setupComplete, wired auth events in login/logout. VM/template audit deferred until those specs land.

**2026-03-16:** spec-vm-lifecycle-create T8a started. Added discovery endpoints: GET /api/hosts/{host_id}/pools, GET /api/hosts/{host_id}/pools/{pool_name}/volumes, GET /api/hosts/{host_id}/networks. Per spec §9.3.

**2026-03-16:** spec-vm-lifecycle-create T8b–T8e complete. GET /api/vms (flat+orphans), VM detail+lifecycle (start/stop/pause/resume/destroy), POST create/clone/claim, PATCH config edit with vm_config_change audit.

**2026-03-16:** spec-template-management complete. internal/template package, GET/POST /api/templates (list, save VM as template).

**2026-03-16:** spec-frontend-build complete. Vite scaffold (web/), npm deps (noVNC, xterm, winbox), lib structure (api, console, winbox-adapter). Backend embed (web/embed.go), SPA fallback, KUI_WEB_DIR support. Makefile: `make all` builds web then Go.

**2026-03-16:** spec-console-realtime complete. SSE GET /api/events (broadcaster, vm.state_changed, host.online/offline). noVNC WebSocket proxy GET /api/hosts/{id}/vms/{uuid}/vnc (local only). xterm.js serial proxy GET /api/hosts/{id}/vms/{uuid}/serial (Connector.OpenSerialConsole, local only).
