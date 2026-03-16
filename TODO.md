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

---

## Remaining Work

- None. MVP implementation complete.

---

## Security Audit (2026-03-16) — RESOLVED

All findings addressed: config chmod 0o600, setup idempotency, validate-host
sanitization, secure cookies, login rate limit, jwt_secret required.
