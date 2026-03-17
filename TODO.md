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

All 12 specs complete and in `specs/done/`:
schema-storage, spec-libvirt-connector, spec-application-bootstrap, api-auth,
spec-audit-integration, spec-vm-lifecycle-create, spec-frontend-build,
spec-template-management, spec-console-realtime, spec-ui-deployment,
feat-keyboard-shortcuts, feat-a11y.

### Active Specs

| spec_id | plan_path | progress |
|---------|-----------|----------|
| feat-stuck-vm | specs/active/feat-stuck-vm/plan.md | TODO |

---

## Next Steps (Formal)

### Verified Current State (2026-03-17)

| Item | Status |
|------|--------|
| Build | pass |
| Test | pass |
| Vet | pass |
| Active specs | 1 |
| Done specs | 12 |
| Per-spec task status | feat-a11y T1–T9 DONE (spec complete); feat-stuck-vm TODO |

### Remaining Implementation Tasks

**feat-stuck-vm** (specs/active/feat-stuck-vm/plan.md): TODO (next)

### Deferred (Planning Required)

- feat-orphan-bulk, feat-domain-xml-edit (v2)
- v3: Backup/restore, import/export

### Planner Triggers

When feat-stuck-vm is done: create specs for feat-orphan-bulk, feat-domain-xml-edit.

### Recommended Delegation Order

1. feat-stuck-vm (next)

---

## Remaining Work

- feat-stuck-vm (v2, next)

---

## Security Audit (2026-03-17) — RESOLVED

All findings addressed: config chmod 0o600, setup idempotency, validate-host
sanitization, secure cookies, login rate limit, jwt_secret required.
