# API & Auth Spec

Specification for KUI's REST API, authentication, and setup wizard. Implementable; greenfield only. Decisions: [decision-log.md](../../docs/prd/decision-log.md) §§0–4. Architecture: [architecture.md](../../docs/prd/architecture.md). Stack: [stack.md](../../docs/prd/stack.md).

---

## 1. What & Why

### Problem

KUI needs a secure, stateless API layer with local authentication and a one-time setup flow. Without it, VM management is inaccessible and first-run deployment requires manual config editing.

### Users

- **MVP**: Single admin account (SQLite users table). Decision-log §0 A5, §2 Local auth.
- **Future**: Authentik SSO users (post-MVP; separate spec).

### Value

- **Authenticated access**: Protects VM lifecycle, console, and config from unauthenticated use.
- **First-run onboarding**: Setup wizard collects admin credentials and host config, validates before saving, and writes config once—no manual YAML editing.
- **Stateless sessions**: JWT (decision-log §0 A13, §2 Session) avoids server-side session storage; scales to 1–2 concurrent users.

---

## 2. Endpoint List and Request/Response Shapes

API style: REST/JSON informal; code is the contract (decision-log §2 API style).

### Endpoint Summary

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| POST | /api/auth/login | None | Login (username, password) → JWT |
| POST | /api/auth/logout | JWT | Logout (optional; client discards token) |
| GET | /api/auth/me | JWT | Current user info |
| GET | /api/setup/status | None | Setup required? (config missing, DB missing, no admin) |
| POST | /api/setup/validate-host | None | Validate host URI + keyfile (attempt connection) |
| POST | /api/setup/complete | None | Create admin + write config (one-time) |
| GET | /api/preferences | JWT | User preferences (default_host_id, list_view_options) |
| PUT | /api/preferences | JWT | Update user preferences |
| GET | /api/hosts | JWT | List hosts from config (id, uri; keyfile redacted) |

### Request/Response Shapes

#### POST /api/auth/login

**Request** (JSON):

```json
{
  "username": "admin",
  "password": "secret"
}
```

**Response** (200 OK):

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-03-17T12:00:00Z"
}
```

- `token`: JWT string. Also set in HTTP-only cookie (see §5).
- `expires_at`: ISO 8601 timestamp of token expiration.

**Errors**:

- 400: Missing username or password → `{ "error": "username and password required" }`
- 401: Invalid credentials → `{ "error": "invalid credentials" }`

---

#### POST /api/auth/logout

**Request**: None (JWT from cookie or Authorization header).

**Response** (200 OK): Empty body. Server clears session cookie if present. Client should discard token.

**Errors**:

- 401: No valid JWT → `{ "error": "unauthorized" }`

---

#### GET /api/auth/me

**Request**: JWT from cookie or `Authorization: Bearer <token>`.

**Response** (200 OK):

```json
{
  "id": "uuid",
  "username": "admin",
  "role": "admin"
}
```

**Errors**:

- 401: No valid/expired JWT → `{ "error": "unauthorized" }`

---

#### GET /api/setup/status

**Request**: None.

**Response** (200 OK):

```json
{
  "setup_required": true,
  "reason": "config_missing"
}
```

- `setup_required`: `true` if setup wizard must run; `false` otherwise.
- `reason`: One of `config_missing`, `db_missing`, `no_admin`, or `null` when `setup_required` is false.

**Logic**:

- `config_missing`: Config file does not exist at resolved path.
- `db_missing`: SQLite DB file does not exist or cannot be opened.
- `no_admin`: DB exists but users table has no admin user.

---

#### POST /api/setup/validate-host

**Request** (JSON):

```json
{
  "host_id": "local",
  "uri": "qemu:///system",
  "keyfile": ""
}
```

- `host_id`: Identifier for the host (used in config).
- `uri`: Libvirt URI (e.g. `qemu:///system`, `qemu+ssh://user@host/system`).
- `keyfile`: Path to SSH private key for remote; empty for local.

**Response** (200 OK):

```json
{
  "valid": true
}
```

or

```json
{
  "valid": false,
  "error": "connection refused"
}
```

**Logic**: Attempt libvirt connection using URI and keyfile. Return `valid: true` on success; `valid: false` with `error` message on failure.

**Errors**:

- 400: Missing uri → `{ "error": "uri required" }`

---

#### POST /api/setup/complete

**Request** (JSON):

```json
{
  "admin": {
    "username": "admin",
    "password": "secret"
  },
  "hosts": [
    {
      "id": "local",
      "uri": "qemu:///system",
      "keyfile": ""
    }
  ],
  "default_host": "local"
}
```

- `admin`: Username and password for the single admin user.
- `hosts`: Array of host configs (id, uri, keyfile).
- `default_host`: ID of the default host (must exist in hosts).

**Response** (201 Created): Empty body.

**Logic**:

1. Validate admin username/password (non-empty, reasonable length).
2. Validate each host (uri required; keyfile required for `qemu+ssh` URIs).
3. Validate `default_host` is in hosts.
4. Create users table if missing; insert admin (bcrypt hash).
5. Auto-generate `jwt_secret` (e.g. 32+ cryptographically random bytes, base64-encoded). Setup never accepts user-supplied `jwt_secret`; the backend always generates it for security.
6. Write YAML config to disk (path from `--config` or default `/etc/kui/config.yaml`), including the generated `jwt_secret` and all host/admin/default values.
7. Drop write access to config (read-only at runtime).
8. Audit log: `wizard_complete` event (see `specs/active/spec-audit-integration/spec.md` §5.1).

**Errors**:

- 400: Validation failure → `{ "error": "..." }` (e.g. "admin username required", "default_host must be in hosts").
- 409: Setup already complete → `{ "error": "setup already complete" }`

---

#### GET /api/preferences

**Request**: JWT from cookie or `Authorization` header.

**Response** (200 OK):

```json
{
  "default_host_id": "local",
  "list_view_options": {
    "list_view": {
      "sort": "name",
      "page_size": 25,
      "group_by": "last_access"
    },
    "onboarding_dismissed": false
  }
}
```

- `default_host_id`, `list_view_options` may be `null`.
- `list_view_options.group_by`: `"last_access"` | `"created_at"`; default `"last_access"` when absent (decision-log §4 VM list grouping).
- `list_view_options.onboarding_dismissed`: `true` when user has dismissed the first-run checklist; default `false` when absent.

**Logic**:

- Read row from `preferences` table by `user_id` in JWT.
- Return row values as-is when present.
- If no row exists, return defaults: both `default_host_id` and `list_view_options` as `null`; `onboarding_dismissed` implied `false`.
- Decision-log §4: preferences are `default_host_id` + `list_view_options`, with one row per `user_id`.

**Errors**:

- 401: No valid JWT → `{ "error": "unauthorized" }`

---

#### PUT /api/preferences

**Request** (JSON): all fields optional; partial updates are allowed.

```json
{
  "default_host_id": "local",
  "list_view_options": {
    "list_view": {
      "sort": "name",
      "page_size": 25,
      "group_by": "last_access"
    },
    "onboarding_dismissed": true
  }
}
```

- `default_host_id` accepts `string` or `null`.
- `list_view_options` accepts JSON object or `null`.
- `list_view_options.group_by`: `"last_access"` | `"created_at"`.
- `list_view_options.onboarding_dismissed`: `boolean` — when `true`, persist so the first-run checklist is not shown again.

**Response** (200 OK): Same shape as `GET /api/preferences`.

**Logic**:

- Upsert row for `user_id` from JWT into `preferences` table (`user_id`, `default_host_id`, `list_view_options`, `updated_at`).
- If `default_host_id` is non-null, validate it exists in configured hosts from runtime config.
- Return the updated preferences row; if either field is omitted, preserve prior value.

**Errors**:

- 400: `default_host_id` missing from config hosts → `{ "error": "default_host_id is not configured" }`
- 401: No valid JWT → `{ "error": "unauthorized" }`

---

#### GET /api/hosts

**Request**: JWT from cookie or `Authorization` header.

**Response** (200 OK):

```json
[
  {
    "id": "local",
    "uri": "qemu:///system"
  }
]
```

**Logic**:

- Read configured hosts from runtime config.
- Return only `id` and `uri`.
- Omit or redact `keyfile` (for security).

**Errors**:

- 401: No valid JWT → `{ "error": "unauthorized" }`

---

### Error Response Contract

All API errors return JSON:

```json
{
  "error": "user-facing message"
}
```

Optional detail for debug UI:

```json
{
  "error": "user-facing message",
  "details": "sanitized technical summary"
}
```

- No stack traces or raw internal errors are returned to clients.
- `400`: Bad request (validation, missing required fields)
- `401`: Unauthorized (missing/invalid/expired JWT)
- `404`: Not found (resource does not exist)
- `409`: Conflict (e.g. setup already complete, duplicate)
- `500`: Internal server error (unexpected failure; full details logged server-side)
---

## 3. JWT Flow

### Issuance

On successful `POST /api/auth/login`:

1. Verify username/password against SQLite users table (bcrypt).
2. Build JWT payload: `sub` (user id), `exp`, `iat`, `role` (optional, always "admin" for MVP).
3. Sign with HS256 using secret from config.
4. Return token in response body and set HTTP-only cookie (see §5).

### Signing

- Algorithm: HS256.
- Secret: From config `jwt_secret` (YAML). Env override `KUI_JWT_SECRET` for Docker.
- Secret must be at least 32 bytes. Fail startup if missing or too short.

### Validation

- Middleware on protected routes: extract JWT from cookie or `Authorization: Bearer <token>`.
- Validate signature, `exp`, `iat`.
- Return 401 if invalid or expired.
- Attach user context (id, username, role) to request for downstream handlers.

### Session Timeout

- Config key: `session.timeout` (YAML, nested under `session`). Aligns with `specs/active/schema-storage/spec.md` §2.6.
- Default: `24h` (86400 seconds). Decision-log §2 Session timeout.
- JWT `exp` = `iat` + session timeout value.
- Cookie `Max-Age` = session timeout value (seconds).

---

## 4. Setup Wizard Flow

### Trigger

`GET /api/setup/status` returns `setup_required: true` when:

1. Config file missing at resolved path.
2. DB missing or cannot be opened.
3. DB exists but no admin user in users table.

Decision-log §2 First-run, §4 First-run flow.

### Steps (UI-driven; API supports each)

1. **Validate host**: `POST /api/setup/validate-host` — attempt libvirt connection; return success/failure. User may add multiple hosts.
2. **Complete**: `POST /api/setup/complete` — create admin in SQLite, write YAML config to disk.

### Config Write Behavior

- Write config once to path from `--config` or default `/etc/kui/config.yaml`.
- After write: KUI drops write access to config (chmod or equivalent; read-only at runtime).
- User restarts KUI (or process restarts) to load new config. No hot-reload.
- Decision-log §2 Config, §2 Setup wizard scope.

### Idempotency

If setup already complete (config exists, admin exists), `POST /api/setup/complete` returns 409 Conflict.

### Host Validation

- Setup wizard validates URI + keyfile before saving (decision-log §2 Setup wizard scope).
- Use libvirt connector to attempt connection; report success/failure.

---

## 5. Session Storage Resolution

Per decision-log §2 Session, §4 Auth storage: OAuth 2.1/OIDC best practice for browser is HTTP-only cookies (avoid localStorage/sessionStorage due to XSS).

### MVP (Local Auth)

- **Mechanism**: JWT in HTTP-only cookie.
- **Cookie name**: `kui_session`.
- **Attributes**:
  - `HttpOnly`: true
  - `SameSite`: Lax
  - `Secure`: true when TLS; false for HTTP (dev)
  - `Path`: /
  - `Max-Age`: session.timeout value (seconds)
- **Dual delivery**: Response body includes `token` for API clients (e.g. curl); cookie set for browser clients.
- **Acceptance**: Middleware accepts JWT from cookie or `Authorization: Bearer <token>`.

### Future (Authentik)

Defer to separate spec. Document that Authentik OIDC will follow same cookie pattern for consistency; avoid token in JS-accessible storage.

---

## 6. CORS

### CORS Policy

- Dev: allow Vite dev origin `http://localhost:5173`.
- Prod: allowed origins are configurable and explicit; no wildcard in production.
- Implement using standard `Access-Control-Allow-Origin`, `Access-Control-Allow-Credentials`, and headers for browser requests.
- Configuration keys:
  - `cors.allowed_origins` (YAML): list of origin strings (e.g. `["http://localhost:5173"]`).
  - `KUI_CORS_ORIGINS` (environment): comma-separated list.

Defaults:
- Dev: `["http://localhost:5173"]`
- Prod: empty list or explicit allowed origins.

---

## 7. Out of Scope (This Spec)

- Authentik SSO implementation (decision-log §0 A5).
- RBAC (viewer/operator roles); MVP admin only (decision-log §2 Role model).
- Formal OpenAPI spec (decision-log §2 API style).
- Password reset flow.

---

## 8. Dependencies

### SQLite Schema (Users Table)

Canonical schema from `specs/active/schema-storage/spec.md` §2.2:

```sql
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'admin',
  created_at TEXT NOT NULL,
  updated_at TEXT NULL
);
```

- `id`: UUID v4 generated at user creation.
- `password_hash`: bcrypt hash.
- MVP: single row; role always "admin".

### Config Format (YAML)

Reference decision-log §2 Config, §4 Config path.

```yaml
hosts:
  - id: local
    uri: qemu:///system
    keyfile: ""
  - id: remote1
    uri: qemu+ssh://user@host/system
    keyfile: /path/to/key
default_host: local
session:
  timeout: 24h
jwt_secret: "base64-or-hex-32-bytes-minimum"
cors:
  allowed_origins: ["http://localhost:5173"] # dev default; prod: explicit list
```

- `hosts`: Required; setup wizard validates before write.
- `session.timeout`: Duration string (e.g. `24h`, `1h`).
- `jwt_secret`: Required; min 32 bytes.
- `cors.allowed_origins`: Allowed CORS origins by environment (`KUI_CORS_ORIGINS` can override with comma list).

### Libvirt Connector

- Interface for host validation: `ValidateConnection(uri, keyfile string) error`.
- Implementation uses `libvirt.org/go/libvirt`; test driver `test:///default` for CI.

---

## 9. Tasks

| # | Task | Deliverable |
|---|------|-------------|
| 1 | Define users table schema (SQLite) | Migration or init SQL; id, username, password_hash, role, created_at |
| 2 | Implement auth service | Login (bcrypt verify, JWT issue), validate JWT, user lookup |
| 3 | Implement auth middleware (chi) | Extract JWT from cookie/header; validate; attach user to context; 401 on failure |
| 4 | Implement setup endpoints and config write behavior | GET /api/setup/status, POST /api/setup/validate-host, POST /api/setup/complete; one-time config write, read-only behavior, 409 idempotency |
| 5 | Implement preferences endpoints | GET /api/preferences, PUT /api/preferences; upsert logic and host validation |
| 6 | Implement hosts endpoint | GET /api/hosts; redact keyfile in responses |
| 7 | Implement CORS middleware | Config/env driven allowed origins and dev/prod policy |
| 8 | Enforce Error Response Contract | Consistent 400/401/404/409/500 JSON responses and no stack traces |
| 9 | Tests | Auth, setup, preferences, hosts, and error/CORS coverage |

---

## Verification Checklist

- [ ] Spec <800 lines
- [ ] Tasks <10
- [ ] No stub implementations
- [ ] Greenfield only (no migration)
- [ ] All decision-log §0–4 entries for API/auth/setup addressed
- [ ] Session storage resolved
- [ ] Error Response Contract applied to all API error paths
- [ ] CORS policy and config overrides are implemented and documented
