# Application Bootstrap Spec — Plan

Plan for creating `specs/active/spec-application-bootstrap/spec.md`. A developer subagent shall implement the spec document following this plan.

---

## Exploration Summary

### Codebase State

- **Greenfield**: No Go code yet. No `cmd/`, `internal/`, or `go.mod`.
- **Specs define**: api-auth (routes, JWT, setup), schema-storage (config schema, DB, Git layout).
- **Dependency chain**: config → db → git → libvirt → API.

### Decision-Log References (§§0–4)

| Topic | Decision |
|-------|----------|
| Config | YAML primary; default `/etc/kui/config.yaml`; `--config` or `KUI_CONFIG` env |
| Deployment | Systemd first; Docker post-MVP |
| TLS | KUI can optionally serve TLS; reverse proxy recommended for prod; HTTP for dev |
| First-run | Setup wizard when config missing or DB missing; wizard writes config once |
| Config path | Default /etc/kui/config.yaml; env override for Docker |

### Spec Dependencies

| Spec | Relevance |
|------|-----------|
| api-auth/spec.md | Route structure (/api/auth/*, /api/setup/*), middleware (JWT), setup flow |
| schema-storage/spec.md | Config YAML structure, env overrides, DB path, Git path |

### Architecture (docs/prd/architecture.md)

- API layer: REST/JSON, auth, audit, session.
- Components: Web UI, API, Libvirt Connector, SQLite, Git.
- Data flow: Browser → API → SQLite + Libvirt.

---

## Plan Structure (Target <800 Lines, <10 Tasks)

The spec.md shall use the following structure. Each section must be implementable (no stubs).

### 1. What & Why

- **Problem**: KUI needs a single entrypoint that loads config, initializes persistence and libvirt, mounts routes with correct middleware order, and shuts down gracefully.
- **Users**: Operator deploying KUI; developer extending the server.
- **Value**: Deterministic startup/shutdown; consistent middleware and route layout; no manual wiring.

### 2. Main Entrypoint

- **File**: `cmd/kui/main.go`.
- **Flow**:
  1. Parse flags (`--config`, `--listen`, `--tls-cert`, `--tls-key`).
  2. Resolve config path: flag > `KUI_CONFIG` env > default `/etc/kui/config.yaml`.
  3. Check if config file exists.
  4. **If config exists**: Load config; validate; init DB, Git, Libvirt connector; start server.
  5. **If config missing**: Setup mode — minimal init (DB path from env/default; no hosts; no JWT secret); mount setup routes + static assets; start server.
  6. Block on server; handle SIGINT/SIGTERM for graceful shutdown.
- **CLI**: stdlib `flag`; fail fast on invalid flags.
- **Logging**: `slog` (log/slog) for startup, shutdown, and errors.

### 3. Config Load and Validation

- **Source**: schema-storage §2.6 (config YAML structure).
- **Path resolution**: `--config` or `KUI_CONFIG`; default `/etc/kui/config.yaml`.
- **Env overrides**: `KUI_CONFIG`, `KUI_DB_PATH`, `KUI_GIT_PATH`, `KUI_HOST_<ID>_KEYFILE`, `KUI_DEFAULT_HOST`, `KUI_DEFAULT_POOL`, `KUI_SESSION_TIMEOUT` (schema-storage §2.6 table).
- **Validation**: `hosts` required when config exists; fail startup if required values missing or invalid.
- **Setup mode**: When config file missing, no config load; use `KUI_DB_PATH` or default `/var/lib/kui/kui.db` for DB path.

### 4. Middleware Stack and Order

Order (outermost first):

1. **Request ID** (optional): generate or propagate `X-Request-ID` for request tracing.
2. **Logging**: log method, path, status, duration; structured (slog).
3. **Recovery**: panic recovery; return 500 with `{ "error": "internal server error" }`; log full stack.
4. **Auth (JWT)**: on protected routes only; extract JWT from cookie or `Authorization: Bearer`; validate; attach user to context; 401 on failure.

- **Excluded from auth**: `/api/auth/login`, `/api/setup/*`, `/` (static assets).
- **Protected**: `/api/auth/logout`, `/api/auth/me`, `/api/*` (future: vms, etc.).

### 5. Route Registration

- **Router**: chi (`github.com/go-chi/chi/v5`).
- **Route groups**:
  - `GET /` — static assets (SPA or static files); per spec-ui-deployment.
  - `POST /api/auth/login` — no auth.
  - `GET /api/setup/status`, `POST /api/setup/validate-host`, `POST /api/setup/complete` — no auth.
  - `POST /api/auth/logout`, `GET /api/auth/me` — JWT required.
  - `POST /api/*` (future: vms, etc.) — JWT required.
- **Mount order**: static assets first; then API routes.
- **Reference**: api-auth spec §2 endpoint list.

### 6. Startup Sequence

1. Parse flags.
2. Resolve config path.
3. **If config exists**:
   - Load YAML; apply env overrides; validate.
   - Open DB at `db.path`; apply schema (create tables if missing).
   - Init Git at `git.path` (ensure templates dir, audit dir).
   - Init Libvirt connector (connection strategy: defer to libvirt spec; bootstrap spec: "init connector; connections on first use or per-request").
4. **If config missing**:
   - DB path from `KUI_DB_PATH` or default `/var/lib/kui/kui.db`.
   - Open DB (create if missing); apply schema.
   - No Git init; no Libvirt init; no JWT secret.
5. Build chi router; apply middleware in order.
6. Mount routes.
7. Start HTTP server (or HTTPS if `--tls-cert` and `--tls-key` provided).
8. Log "KUI listening on ...".

### 7. Shutdown Sequence

1. On SIGINT/SIGTERM: call `http.Server.Shutdown(ctx)` with timeout (e.g. 30s).
2. Close Libvirt connections (connector cleanup).
3. Close DB connection.
4. Close Git handles (if any).
5. Log "KUI shutdown complete".

### 8. TLS and Listen

- **HTTP (default)**: `--listen` default `:8080`; document for dev.
- **TLS**: `--tls-cert` and `--tls-key`; when both set, serve HTTPS. Decision-log §2 TLS: reverse proxy recommended for prod; document both.
- **Listen address**: `--listen` or `KUI_LISTEN` env.

### 9. Dependencies

- **internal/config**: Load YAML; env overrides; validation.
- **internal/db**: Open SQLite; schema init; close.
- **internal/git**: Init at path; templates/audit dirs.
- **internal/libvirt**: Connector; init; close connections.
- **internal/middleware**: Logging, recovery, auth.
- **internal/routes**: Auth, setup, static (future: vms).

### 10. Schema Note

- **Canonical schema**: `specs/active/schema-storage/spec.md` §2.2 is authoritative. api-auth and bootstrap use the same schema (users.id UUID, preferences, vm_metadata, audit_events).

### 11. Out of Scope

- Migration workflows, backfill, compatibility mode.
- Hot-reload of config.
- Connection pooling strategy (defer to libvirt spec).

### 12. Tasks (Target <10)

| # | Task | Deliverable |
|---|------|-------------|
| 1 | Define main entrypoint flow | plan.md §2 reflected in spec |
| 2 | Implement config load | internal/config; path, env overrides, validation |
| 3 | Implement middleware stack | internal/middleware; order: logging, recovery, auth |
| 4 | Implement route registration | chi router; mount routes per api-auth |
| 5 | Implement startup sequence | DB init, Git init, Libvirt init |
| 6 | Implement shutdown sequence | Graceful drain; close libvirt, DB, Git |
| 7 | Implement setup mode | Config-missing startup; minimal init |
| 8 | Tests | Startup with/without config; shutdown; middleware order |

---

## Verification Checklist

- [ ] Spec <800 lines
- [ ] Tasks <10
- [ ] No stub implementations
- [ ] Greenfield only
- [ ] All decision-log §0–4 entries for Config, Deployment, TLS addressed
- [ ] Middleware order: logging → recovery → auth
- [ ] Route structure matches api-auth spec
- [ ] Config schema matches schema-storage spec

---

## Developer Instructions

When implementing this spec:

1. Create `specs/active/spec-application-bootstrap/spec.md` using this plan as the blueprint.
2. Ensure each section references the correct source (decision-log, architecture, api-auth, schema-storage).
3. Keep tasks under 10 and spec under 800 lines.
4. Do not add migration or backfill logic.
5. Do not create stub implementations; document functional behavior.
