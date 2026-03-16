# API & Auth Spec — Plan

Plan for creating `specs/active/api-auth/spec.md`. A developer subagent shall implement the spec document following this plan.

---

## Exploration Summary

### Decision-Log Entries (API, Auth, Setup Wizard)

| § | Topic | Decision |
|---|-------|----------|
| §0 A5 | Authentik | Post-MVP; local SQLite auth MVP; Authentik if trivial (<1 day) |
| §0 A13 | Session | JWT (stateless) |
| §2 | API style | REST/JSON informal; code is the contract |
| §2 | Auth | Local auth first; single admin MVP |
| §2 | Config | YAML; setup wizard validates hosts; writes once; read-only at runtime |
| §2 | First-run | Wizard when config missing or DB missing |
| §2 | Local auth | SQLite users table; single admin only; no RBAC |
| §2 | Session | JWT stateless; storage: research OAuth 2.1 OIDC for Authentik; defer to spec |
| §2 | Session timeout | Configurable in YAML; long default (24h or until browser close) |
| §2 | Setup wizard scope | Admin + host only; validate URI/keyfile before saving; skip or create template later |
| §4 | Auth/Session storage | Research OAuth 2.1 OIDC; follow token storage standards; defer to spec |

### Architecture (from docs/prd/architecture.md)

- API: REST/JSON, auth, audit, session
- Stack: go-chi/chi, slog

### Codebase

- Greenfield: no Go HTTP handlers or auth code yet.

---

## Spec Structure (Target <800 Lines, <10 Tasks)

The spec.md shall use the following structure. Each section must be implementable (no stubs).

### 1. What & Why

- **Problem**: KUI needs a secure, stateless API layer with local auth and a one-time setup flow.
- **Users**: Single admin (MVP); future: Authentik SSO users.
- **Value**: Enables authenticated access to VM management; first-run onboarding without manual config editing.

### 2. Endpoint List and Request/Response Shapes

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| POST | /api/auth/login | None | Login (username, password) → JWT |
| POST | /api/auth/logout | JWT | Logout (optional; client discards token) |
| GET | /api/auth/me | JWT | Current user info |
| GET | /api/setup/status | None | Setup required? (config missing, DB missing, no admin) |
| POST | /api/setup/validate-host | None | Validate host URI + keyfile (attempt connection) |
| POST | /api/setup/complete | None | Create admin + write config (one-time) |

**Request/Response shapes** (informal; code is contract):

- `POST /api/auth/login`: `{ "username": "...", "password": "..." }` → `{ "token": "jwt...", "expires_at": "ISO8601" }` or 401
- `GET /api/auth/me`: → `{ "id": "...", "username": "...", "role": "admin" }` or 401
- `GET /api/setup/status`: → `{ "setup_required": true|false, "reason": "config_missing"|"db_missing"|"no_admin"|null }`
- `POST /api/setup/validate-host`: `{ "host_id": "...", "uri": "...", "keyfile": "..." }` → `{ "valid": true|false, "error": "..." }`
- `POST /api/setup/complete`: `{ "admin": { "username": "...", "password": "..." }, "hosts": [{ "id": "...", "uri": "...", "keyfile": "..." }], "default_host": "..." }` → 201 or 4xx

**Error convention**: `{ "error": "..." }` with appropriate HTTP status codes.

### 3. JWT Flow

- **Issuance**: On successful login, backend issues JWT; payload: `sub` (user id), `exp`, `iat`, optional `role`.
- **Signing**: HS256 (configurable secret in YAML); env override for Docker.
- **Storage**: Per decision log, research OAuth 2.1 OIDC token storage. For MVP local auth: **HTTP-only cookie** preferred (XSS mitigation); Bearer header acceptable for API clients. Spec must specify: cookie name, path, SameSite, Secure (when TLS), max-age from session.timeout config.
- **Validation**: Middleware validates JWT on protected routes; return 401 if invalid/expired.
- **Session timeout**: From config `session.timeout`; default 24h or until browser close (specify exact default).

### 4. Setup Wizard Flow

1. **Trigger**: `GET /api/setup/status` returns `setup_required: true` when: config file missing, DB missing, or no admin user in DB.
2. **Steps** (UI-driven; API supports each):
   - Validate host: `POST /api/setup/validate-host` — attempt libvirt connection; return success/failure.
   - Complete: `POST /api/setup/complete` — create admin in SQLite, write YAML config to disk.
3. **Config write behavior**:
   - Write config once; path from `--config` or default `/etc/kui/config.yaml`.
   - After write: KUI drops write access to config (read-only at runtime).
   - User restarts KUI (or process restarts) to load new config.
4. **Idempotency**: If setup already complete, `POST /api/setup/complete` returns 409 (Conflict).

### 5. Session Storage Approach

Per decision log §2 Session, §4 Auth storage: research OAuth 2.1 OIDC for Authentik; follow token storage standards.

**Findings** (from web research): OAuth 2.1/OIDC best practice for browser: HTTP-only cookies (avoid localStorage/sessionStorage due to XSS). For SPAs: backend can store tokens server-side and return session cookie.

**Spec resolution**:
- **MVP (local auth)**: JWT in HTTP-only cookie (`kui_session` or similar). Cookie: HttpOnly, SameSite=Lax, Secure when TLS, Path=/, Max-Age from session.timeout.
- **Future (Authentik)**: Defer to separate spec; document that Authentik OIDC will follow same cookie pattern for consistency; avoid token in JS-accessible storage.

### 6. Out of Scope (for this spec)

- Authentik SSO implementation
- RBAC (viewer/operator roles)
- Formal OpenAPI spec
- Password reset flow

### 7. Dependencies

- SQLite schema (users table) — reference or inline minimal schema
- Config format (YAML hosts, default_host, session.timeout) — reference decision-log
- Libvirt connector for host validation — reference or stub interface

### 8. Tasks (for plan.md implementation phase)

Tasks should be <10 and map to concrete deliverables:

1. Define users table schema (SQLite)
2. Implement auth service (login, JWT issue, validate)
3. Implement auth middleware (chi)
4. Implement setup status + validate-host + complete endpoints
5. Implement config write (one-time, then read-only)
6. Document session storage (cookie spec)
7. Error handling (401, 409, 4xx)
8. Tests (auth, setup flow)

---

## Verification Checklist

- [ ] Spec <800 lines
- [ ] Tasks <10
- [ ] No stub implementations
- [ ] Greenfield only (no migration)
- [ ] All decision-log §0–4 entries for API/auth/setup addressed
- [ ] Session storage resolved

---

## Next Action

Invoke `developer` subagent with:

> Create `specs/active/api-auth/spec.md` following the structure and content defined in `specs/active/api-auth/plan.md`. The spec must be implementable; include all endpoint shapes, JWT flow, setup wizard flow, and session storage resolution. Target <800 lines or <10 tasks.
