# Spec Completion Order

Produced from explorer runs on each `specs/active/` specification.

## Completion Order Table

| Order | Spec | Phase | Blocked By | Blocks | Notes |
|------:|------|-------|------------|--------|-------|
| 1 | **schema-storage** | Foundation | — | — | DONE. Moved to specs/done/. |
| 2 | **spec-libvirt-connector** | Foundation | — | — | DONE. Moved to specs/done/. |
| 3 | **spec-application-bootstrap** | Core | — | — | DONE. Moved to specs/done/. |
| 4 | **api-auth** | Core | schema-storage, spec-libvirt-connector | spec-audit-integration, spec-ui-deployment | DONE. Moved to specs/done/. |
| 5 | **spec-audit-integration** | Core | schema-storage, spec-application-bootstrap, api-auth | spec-vm-lifecycle-create, spec-template-management | Audit service; wire `wizard_complete` in setup, then VM/template events as those specs land. |
| 6 | **spec-vm-lifecycle-create** | Feature | spec-libvirt-connector, schema-storage, spec-application-bootstrap, spec-audit-integration | spec-template-management, spec-ui-deployment | VM create, clone, lifecycle, orphans, config edit, discovery endpoints. |
| 7 | **spec-frontend-build** | Feature | spec-application-bootstrap (for serving) | spec-console-realtime, spec-ui-deployment | Vite, web/, Winbox, noVNC, xterm. Can scaffold in parallel with backend. |
| 8 | **spec-template-management** | Feature | schema-storage, spec-vm-lifecycle-create, spec-libvirt-connector, spec-audit-integration, spec-application-bootstrap | — | Save VM as template, list templates. Reuses clone/disk copy patterns. |
| 9 | **spec-console-realtime** | Feature | spec-application-bootstrap, spec-libvirt-connector, spec-frontend-build, api-auth | spec-ui-deployment | noVNC, xterm.js, SSE `GET /api/events`. Connector may need VNC/serial helpers. |
| 10 | **spec-ui-deployment** | Integration | spec-application-bootstrap, api-auth, spec-frontend-build, spec-console-realtime, spec-vm-lifecycle-create | — | Winbox canvas, host selector, alerts, first-run checklist, VM list, systemd, TLS. |

## Phase Summary

| Phase | Specs | Purpose |
|-------|-------|---------|
| **Foundation** | schema-storage, spec-libvirt-connector | Storage design and libvirt integration; no upstream dependencies |
| **Core** | spec-application-bootstrap, api-auth, spec-audit-integration | App startup, auth, setup, audit contract |
| **Feature** | spec-vm-lifecycle-create, spec-frontend-build, spec-template-management, spec-console-realtime | VM lifecycle, frontend, templates, console/SSE |
| **Integration** | spec-ui-deployment | Full UI and deployment behavior |

## Parallelization Opportunities

- **spec-libvirt-connector** and **schema-storage** (via bootstrap design) can proceed in parallel.
- **spec-frontend-build** can be scaffolded in parallel with backend work once bootstrap exists.
- **spec-audit-integration** can be implemented incrementally as each feature spec lands (wizard → VM → template).

## Critical Path

```
schema-storage → spec-application-bootstrap → api-auth → spec-audit-integration
       ↓                    ↓
spec-libvirt-connector ─────┴──→ spec-vm-lifecycle-create → spec-template-management
       ↓                    ↓
spec-frontend-build ───────┴──→ spec-console-realtime → spec-ui-deployment
```
