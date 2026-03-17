# KUI TODO

**Last updated:** 2026-03-17

---

## Verified Current State

| Check | Status |
|-------|--------|
| Build | pass |
| Test | pass |
| Vet | pass |
| Spec integrity | pass |
| Doc integrity | pass |

### Specs

All 13 specs complete and in `specs/done/`:
schema-storage, spec-libvirt-connector, spec-application-bootstrap, api-auth,
spec-audit-integration, spec-vm-lifecycle-create, spec-frontend-build,
spec-template-management, spec-console-realtime, spec-ui-deployment,
feat-keyboard-shortcuts, feat-a11y, feat-stuck-vm.

### Active Specs

None. All planned specs complete.

---

## Next Steps (Formal)

### Verified Current State (2026-03-17)

| Item | Status |
|------|--------|
| Build | pass |
| Test | pass |
| Vet | pass |
| Active specs | 0 |
| Done specs | 13 |
| Per-spec task status | feat-a11y T1–T9 DONE; feat-stuck-vm T1–T5 DONE |

### Remaining Implementation Tasks


### Deferred (Planning Required)

- feat-orphan-bulk, feat-domain-xml-edit (v2)
- v3: Backup/restore, import/export

### Planner Triggers

Create specs for feat-orphan-bulk, feat-domain-xml-edit (planner triggers).

### Recommended Delegation Order

All v2 specs complete. Next: create specs for feat-orphan-bulk, feat-domain-xml-edit.

---

## Remaining Work

- feat-orphan-bulk, feat-domain-xml-edit (planning required)

---

## Security Audit (2026-03-17) — RESOLVED

All findings addressed: config chmod 0o600, setup idempotency, validate-host
sanitization, secure cookies, login rate limit, jwt_secret required.
