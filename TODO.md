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

### Specs

All 11 specs complete and in `specs/done/`:
schema-storage, spec-libvirt-connector, spec-application-bootstrap, api-auth,
spec-audit-integration, spec-vm-lifecycle-create, spec-frontend-build,
spec-template-management, spec-console-realtime, spec-ui-deployment,
feat-keyboard-shortcuts.

### Active Specs

| spec_id | plan_path | progress |
|---------|-----------|----------|
| feat-a11y | specs/active/feat-a11y/plan.md | TODO |
| feat-stuck-vm | specs/active/feat-stuck-vm/plan.md | TODO |

---

## Next Steps (Formal)

### Verified Current State (2026-03-16)

| Item | Status |
|------|--------|
| Build | pass |
| Test | pass |
| Vet | pass |
| Active specs | 2 |
| Done specs | 11 |
| Per-spec task status | feat-keyboard-shortcuts T1–T6 DONE (spec complete) |

### Remaining Implementation Tasks

**feat-a11y** (specs/active/feat-a11y/plan.md): TODO

**feat-stuck-vm** (specs/active/feat-stuck-vm/plan.md): TODO (after feat-a11y)

### Deferred (Planning Required)

- feat-orphan-bulk, feat-domain-xml-edit (v2)
- v3: Backup/restore, import/export

### Planner Triggers

When feat-keyboard-shortcuts, feat-a11y, feat-stuck-vm are done: create specs for feat-orphan-bulk, feat-domain-xml-edit.

### Recommended Delegation Order

1. feat-a11y (next)
2. feat-stuck-vm (after feat-a11y)

---

## Remaining Work

- feat-a11y, feat-stuck-vm (v2, next)

---

## Security Audit (2026-03-16) — RESOLVED

All findings addressed: config chmod 0o600, setup idempotency, validate-host
sanitization, secure cookies, login rate limit, jwt_secret required.
