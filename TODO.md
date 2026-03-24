# KUI TODO

**Last updated:** 2026-03-23

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

All 25 specs complete and in `specs/done/`:
api-auth, coverage-100, feat-a11y, feat-domain-xml-edit, feat-host-provisioning,
feat-keyboard-shortcuts, feat-orphan-bulk, feat-setup-wizard-ui, feat-shadcn-ui,
feat-stuck-vm, gap-401-session-audit, gap-audit, gap-domain-xml-network,
gap-remediation, gap-template-network, schema-storage, setup-host-validation,
spec-application-bootstrap, spec-audit-integration, spec-console-realtime,
spec-frontend-build, spec-libvirt-connector, spec-template-management,
spec-ui-deployment, spec-vm-lifecycle-create

Canonical list: `make specs-list` (same as `ls -1 specs/done | sort`).

### Active Specs

None.

---

## Next Steps (Formal)

### Verified Current State (2026-03-23)

| Item | Status |
|------|--------|
| Build | pass |
| Test | pass |
| Vet | pass |
| Active specs | 0 |
| Done specs | 25 |
| Per-spec task status | feat-a11y T1–T9 DONE; feat-stuck-vm T1–T5 DONE; gap-audit T1–T5 DONE; coverage-100 DONE (targets met within exclusions); feat-shadcn-ui T1–T12 DONE |

### Remaining Implementation Tasks

None.

**Gap closure (gap-audit theme):** All 14 gaps resolved. The work is reflected across five gap-theme specs: [gap-audit](specs/done/gap-audit/plan.md) (umbrella; incl. gaps 9–14 such as duplicate host IDs, qemu+ssh keyfile, default host stale, password confirmation, validate-host keyfile, empty host ID), plus focused specs [gap-remediation](specs/done/gap-remediation/plan.md), [gap-401-session-audit](specs/done/gap-401-session-audit/plan.md), [gap-domain-xml-network](specs/done/gap-domain-xml-network/plan.md), and [gap-template-network](specs/done/gap-template-network/plan.md). Gaps 1–8 (remediation, template, domain XML, 401) were addressed in those focused specs and related plans; gaps 9–14 are covered in the gap-audit plan.

**coverage-100 (specs/done/coverage-100/):** Targets met within documented exclusions (routes 62%, sshtunnel 60%, cmd 70.3%, web 92%).

### Deferred (Planning Required)

- v3: Backup/restore, import/export

---

## Remaining Work

- None. All specs complete. Deferred only: v3 backup/restore, import/export (planning required).

---

## Security Audit (2026-03-17) — RESOLVED

All findings addressed: config chmod 0o600, setup idempotency, validate-host
sanitization, secure cookies, login rate limit, jwt_secret required.
