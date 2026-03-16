# KUI TODO

**Last updated:** 2026-03-16

---

## Verified Current State

| Check | Status |
|-------|--------|
| Build | N/A — no codebase (greenfield) |
| Test | N/A |
| Vet | N/A |
| Spec integrity | pass |
| Doc integrity | pass |

### Active Specs (10)

| spec_id | progress | blockers |
|---------|----------|----------|
| schema-storage | spec ✓; impl 0% | — |
| spec-libvirt-connector | spec ✓; impl 0% | — |
| spec-application-bootstrap | spec ✓; impl 0% | schema-storage |
| api-auth | spec ✓; impl 0% | schema-storage, spec-libvirt-connector |
| spec-audit-integration | spec ✓; impl 0% | schema-storage, spec-application-bootstrap, api-auth |
| spec-vm-lifecycle-create | spec ✓; impl 0% | spec-libvirt-connector, schema-storage, spec-application-bootstrap, spec-audit-integration |
| spec-frontend-build | spec ✓; impl 0% | spec-application-bootstrap |
| spec-template-management | spec ✓; impl 0% | schema-storage, spec-vm-lifecycle-create, spec-libvirt-connector, spec-audit-integration, spec-application-bootstrap |
| spec-console-realtime | spec ✓; impl 0% | spec-application-bootstrap, spec-libvirt-connector, spec-frontend-build, api-auth |
| spec-ui-deployment | spec ✓; impl 0% | spec-application-bootstrap, api-auth, spec-frontend-build, spec-console-realtime, spec-vm-lifecycle-create |

### Done Specs

None (greenfield).

---

## Remaining Implementation Tasks

### Foundation (Order 1–2)

| Task ID | Spec | Description | Requirements |
|---------|------|-------------|--------------|
| T1 | schema-storage | Scaffold go.mod, internal/config (YAML load, env overrides, validation) | schema-storage spec §2.6; stack.md; no stubs |
| T2 | schema-storage | Implement internal/db (SQLite, schema §2.2) | DDL from spec; apply on init |
| T3 | schema-storage | Implement internal/git (templates + audit layout) | spec §2.4; init dirs |
| T4 | spec-libvirt-connector | Implement Connector interface, domain/network/storage ops | plan §2; test driver for CI |

### Core (Order 3–5)

| Task ID | Spec | Description | Requirements |
|---------|------|-------------|--------------|
| T5 | spec-application-bootstrap | cmd/kui/main.go, config load, middleware, routes, startup/shutdown | bootstrap plan §2–8 |
| T6 | api-auth | Auth service, setup endpoints, JWT middleware | api-auth spec |
| T7 | spec-audit-integration | Audit service, integration points | audit spec |

### Feature (Order 6+)

Deferred until foundation and core complete.

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
