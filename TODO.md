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
| spec-audit-integration | spec ✓; impl 0% | — |
| spec-vm-lifecycle-create | spec ✓; impl 0% | spec-audit-integration |
| spec-frontend-build | spec ✓; impl 0% | — |
| spec-template-management | spec ✓; impl 0% | spec-vm-lifecycle-create |
| spec-console-realtime | spec ✓; impl 0% | spec-frontend-build |
| spec-ui-deployment | spec ✓; impl 0% | spec-console-realtime, spec-vm-lifecycle-create |

### Done Specs (4)

| spec_id | status |
|---------|--------|
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
| T7 | spec-audit-integration | Audit service, integration points | audit spec |

### Feature (Order 6+)

Deferred until foundation and core complete.

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

---

## Critical Path

```
schema-storage (T1–T3) → spec-application-bootstrap (T5)
spec-libvirt-connector (T4) ─┘
```
