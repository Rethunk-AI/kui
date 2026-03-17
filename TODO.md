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

All 15 specs complete and in `specs/done/`:
schema-storage, spec-libvirt-connector, spec-application-bootstrap, api-auth,
spec-audit-integration, spec-vm-lifecycle-create, spec-frontend-build,
spec-template-management, spec-console-realtime, spec-ui-deployment,
feat-keyboard-shortcuts, feat-a11y, feat-stuck-vm, feat-orphan-bulk, feat-domain-xml-edit.

### Active Specs

| spec_id | plan_path | progress |
|---------|-----------|----------|
| coverage-100 | specs/active/coverage-100/plan.md | In progress (routes 62%, sshtunnel 60%, cmd 70%, web 92%) |

---

## Next Steps (Formal)

### Verified Current State (2026-03-17)

| Item | Status |
|------|--------|
| Build | pass |
| Test | pass |
| Vet | pass |
| Active specs | 1 (coverage-100) |
| Done specs | 15 |
| Per-spec task status | feat-a11y T1–T9 DONE; feat-stuck-vm T1–T5 DONE; coverage-100 routes 62%, cmd 70%, web 92% |

### Remaining Implementation Tasks

**coverage-100** (specs/active/coverage-100/plan.md): Optional — routes 62%, sshtunnel 60%, cmd 70.3%, web 92%. Targets met for cmd and web.

**Gap remediation (specs/active/gap-audit/gaps.md):**

| # | Spec | Task | Status |
|---|------|------|--------|
| 7 | gap-template-network | Validate template network on target host before create; return 400 with clear error | DONE |
| 8 | gap-domain-xml-network | Validate network in edited domain XML before apply (same pattern as patchVMConfig) | DONE |
| 6 | gap-401-session-audit | Audit apiFetch/SSE for 401 handling; add global 401 handler if needed; document findings | DONE |

### Remaining Implementation Tasks

None. Gaps #6, #7, #8 complete.

### Deferred (Planning Required)

- v3: Backup/restore, import/export

### Recommended Delegation Order

1. (Optional) Continue coverage-100 tasks (routes, sshtunnel, cmd, web)

---

## Remaining Work

- None. Gaps #6, #7, #8 (session 401, template network, domain XML network) complete.
- coverage-100 (optional; routes 62%, cmd 70%, web 92%)

---

## Security Audit (2026-03-17) — RESOLVED

All findings addressed: config chmod 0o600, setup idempotency, validate-host
sanitization, secure cookies, login rate limit, jwt_secret required.
