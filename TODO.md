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

All 10 specs complete and in `specs/done/`:
schema-storage, spec-libvirt-connector, spec-application-bootstrap, api-auth,
spec-audit-integration, spec-vm-lifecycle-create, spec-frontend-build,
spec-template-management, spec-console-realtime, spec-ui-deployment.

### Active Specs

| spec_id | plan_path | progress |
|---------|-----------|----------|
| feat-keyboard-shortcuts | specs/active/feat-keyboard-shortcuts/plan.md | TODO |
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
| Active specs | 3 |
| Done specs | 10 |
| Per-spec task status | feat-keyboard-shortcuts T1–T2 DONE, T3–T6 TODO |

### Remaining Implementation Tasks

**feat-keyboard-shortcuts** (specs/active/feat-keyboard-shortcuts/plan.md):

| Task | Description | Location | Status |
|------|-------------|----------|--------|
| T1 | Create shortcuts module | web/src/lib/shortcuts.ts | DONE |
| T2 | VM list row focus and selection | web/src/components/VMList.ts | DONE |
| T3 | Modal Escape handling | CreateVMModal.ts, CloneVMModal.ts | TODO |
| T4 | WinBox Escape handling | web/src/lib/winbox-adapter.ts | TODO |
| T5 | Integrate shortcuts in main.ts | web/src/main.ts | TODO |
| T6 | Shortcut help overlay | web/src/components/ShortcutHelp.ts | TODO |

**feat-a11y**, **feat-stuck-vm**: deferred until feat-keyboard-shortcuts complete.

### Deferred (Planning Required)

- feat-orphan-bulk, feat-domain-xml-edit (v2)
- v3: Backup/restore, import/export

### Planner Triggers

When feat-keyboard-shortcuts, feat-a11y, feat-stuck-vm are done: create specs for feat-orphan-bulk, feat-domain-xml-edit.

### Recommended Delegation Order

1. feat-keyboard-shortcuts T1 → T2 → T3 → T4 → T5 → T6

---

## Remaining Work

- feat-keyboard-shortcuts T2–T6 (v2)
- feat-a11y, feat-stuck-vm (v2, after shortcuts)

---

## Security Audit (2026-03-16) — RESOLVED

All findings addressed: config chmod 0o600, setup idempotency, validate-host
sanitization, secure cookies, login rate limit, jwt_secret required.
