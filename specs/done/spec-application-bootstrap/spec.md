# Application Bootstrap Specification

## §1 What & Why

KUI needs a deterministic and testable startup path in `cmd/kui/main.go` that:

- Resolves and validates runtime configuration.
- Initializes required persistence and git/audit layout for normal startup.
- Provides a one-time setup entrypoint when first-run prerequisites are missing.
- Registers routes and middleware in a predictable order.
- Runs and shuts down with controlled resource cleanup.

This spec exists to prevent implicit startup behavior and hard-to-debug partial-initialization states by explicitly describing the full bootstrap contract.

The design is greenfield and non-migratory: only current-correct behavior is defined; no migration, compatibility mode, or backfill branches are introduced.

Decision alignment:

- `docs/prd/decision-log.md` §0 and §2: local Linux targets, single admin-first auth, systemd-first deployment, optional TLS, first-run setup wizard.
- `docs/prd/architecture.md`: API-anchored architecture with UI and persistence (SQLite, Libvirt Connector).
- `specs/done/api-auth/spec.md`: endpoint contract for auth and setup routes.
- `specs/done/schema-storage/spec.md` §§2.2, 2.6: canonical schema and config contract.

## §2 Main Entrypoint (`cmd/kui/main.go` flow)

Bootstrap flow:

1. Parse CLI flags with stdlib `flag`:
   - `--config` path override.
   - `--listen` bind address.
   - `--tls-cert`, `--tls-key` TLS assets.
2. Resolve config path with precedence:
   - `--config`
   - `KUI_CONFIG`
   - `/etc/kui/config.yaml`
3. Check file existence of resolved config path.
4. Build `bootstrap.Mode`:
   - `configured` if config exists and passes schema validation.
   - `setup` if config missing or config invalid for startup.
5. Start dependency initialization according to mode.
6. Build shared `app.Context` containing logger, DB handle, git handle, connector façade, and middleware helpers.
7. Build router (`chi`) and register middleware + routes.
8. Create and start HTTP server.
9. Block on signal waitgroup:
   - Listen for `SIGINT`/`SIGTERM`.
   - Trigger graceful shutdown sequence.

Logging policy:

- Use `log/slog` for startup/stop lifecycle, dependency init milestones, and fatal validation errors.
- Startup logs must include resolved config source (`--config`/env/default) and selected mode.

## §3 Config Load and Validation

### 3.1 Resolution rules

- Path order: `--config` > `KUI_CONFIG` > `/etc/kui/config.yaml`.
- Parse config only when setup does not own the flow.
- In setup mode, skip YAML parse and proceed with a minimal runtime baseline.

### 3.2 Canonical config schema

Config schema and defaults follow `specs/done/schema-storage/spec.md` §2.6:

- Required: `hosts` list
  - `id`, `uri`, optional `keyfile`
- Optional with defaults:
  - `vm_defaults`, `default_host`, `default_pool`, `default_name_template`
  - `template_storage`
  - `git.path` (default `/var/lib/kui`)
  - `session.timeout` (default `24h`)
  - `db.path` (default `/var/lib/kui/kui.db`)

### 3.3 Env overrides

Environment variables override file values when set:

- `KUI_CONFIG`
- `KUI_DB_PATH` -> `db.path`
- `KUI_GIT_PATH` -> `git.path`
- `KUI_HOST_<ID>_KEYFILE` -> host keyfile by host id.
  - `<ID>` is uppercased and non-alphanumeric collapsed to `_`.
- `KUI_DEFAULT_HOST` -> `default_host`
- `KUI_DEFAULT_POOL` -> `default_pool`
- `KUI_SESSION_TIMEOUT` -> `session.timeout`
- `KUI_LISTEN` -> server bind address (listen default resolution)
- `KUI_JWT_SECRET` -> JWT secret used by auth token signing

### 3.4 Validation and failure policy

Validation must be strict and fail-fast:

- Reject startup when `hosts` is missing/empty (decision-log first-run and config requirements).
- Validate unique `host.id` values and unique `host.uri` where practical.
- Reject non-parseable `session.timeout`.
- Reject JWT secret shorter than 32 bytes in configured mode.
- Reject `default_host` if not present in host list.
- Enforce remote host keyfile policy:
  - For `qemu+ssh` URIs, `keyfile` must be present or env-overridden.
- Emit JSON-formatted startup errors to stderr and terminate with non-zero status.

### 3.5 Setup-mode config behavior

- Setup mode must not require an existing valid JWT secret.
- In setup mode:
  - DB path is from env override `KUI_DB_PATH` or default `/var/lib/kui/kui.db`.
  - DB can be initialized and migrated for required schema.
  - Git and libvirt connector initialization are deferred.
  - Setup endpoints are still routable to complete bootstrap data capture.

## §4 Middleware Stack and Order

Outer to inner middleware order:

1. Optional request ID (`X-Request-ID`) generation/passthrough.
2. Structured access logging (`slog`): method, route pattern, status, duration, request id.
3. CORS middleware:
   - Resolve allowed origins from:
     - development default: `http://localhost:5173`
     - production config value `cors.allowed_origins`
     - env override `KUI_CORS_ORIGINS`
   - Apply permissive headers for allowed origins and methods/headers needed by browser workflows.
4. Panic recovery middleware:
   - recover to prevent process crash,
   - return `500` with body `{ "error": "internal server error" }`,
   - log stack/error detail.
5. Auth middleware (JWT):
   - Accept token from cookie `kui_session` or `Authorization: Bearer`.
   - Decode/verify with HS256 + config/JWT secret.
   - Attach user context (`id`, `username`, `role`) for downstream handlers.
   - Return `401` with `{ "error": "unauthorized" }` on failures.

Notes:

- `api-auth` does not own CORS; bootstrap owns it.
- CORS must run early (before auth) so `OPTIONS` preflight can succeed anonymously.

Route-level auth policy:

- Exempt endpoints:
  - `GET /`
  - `POST /api/auth/login`
  - `GET /api/setup/status`
  - `POST /api/setup/validate-host`
  - `POST /api/setup/complete`
- Protected endpoints:
  - `POST /api/auth/logout`
  - `GET /api/auth/me`
  - `/api/*` management and future VM endpoints.

## §5 Route Registration

Router implementation uses `github.com/go-chi/chi/v5`.

### 5.1 Static and SPA fallback

- Static UI assets (SPA bundle) are served from the embedded `dist/` or configured static directory.
- `GET /` serves the SPA entry (e.g. `index.html`).
- **SPA catch-all:** Any path under `/` that does not match an API route (`/api/*`), WebSocket/SSE route, or existing static file must serve `index.html`. This enables client-side routing (e.g. `/vms`, `/hosts/1/vms/uuid`). Per [specs/active/spec-frontend-build/spec.md](../../active/spec-frontend-build/spec.md) §5.
- Route order: API and WebSocket/SSE handlers are registered before the SPA catch-all.
- Static/SPA routes are evaluated so unauthenticated UI entry remains reachable.

### 5.2 API auth + setup routes

Route contracts map to `specs/done/api-auth/spec.md`:

- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- `GET /api/setup/status`
- `POST /api/setup/validate-host`
- `POST /api/setup/complete`

Implementation notes:

- `POST /api/auth/logout` and `GET /api/auth/me` require valid auth context.
- `GET /api/auth/me` must return user payload fields described by api-auth spec (with bootstrap-defined id serialization).
- `POST /api/setup/complete` writes the resolved config path once and sets setup completion state.

### 5.3 Future API mount pattern

- Mount `/api` group under a single router.
- For unimplemented future handlers, mount placeholders only when explicit feature specs exist; no fallback or fake endpoints.
- Route-to-spec reference (MVP):
  - VM lifecycle (list, create, clone, start, stop, pause, resume, destroy, claim, config edit): `specs/active/spec-vm-lifecycle-create/spec.md`
  - Console (VNC, serial WebSocket): `specs/active/spec-console-realtime/spec.md`
  - Real-time (SSE status stream): `specs/active/spec-console-realtime/spec.md`
  - Templates (list, save VM as template): `specs/active/spec-template-management/spec.md`
  - Preferences, hosts: `specs/done/api-auth/spec.md`

## §6 Startup Sequence

### 6.1 Normal startup

1. Resolve configuration and validate schema.
2. Initialize database (see §6.1.1).
3. Initialize git workspace (see §6.1.2).
4. Initialize libvirt connector façade and host metadata from config.
5. Build chi router and middleware chain.
6. Register routes.
7. Start listener at effective bind address:
   - HTTP by default or HTTPS when both TLS flags provided.
8. Log `KUI listening on <addr>`.

### 6.1.1 Database initialization

1. Create the parent directory of `db.path` if it does not exist.
   - Example for default path: `/var/lib/kui/kui.db` requires `/var/lib/kui`.
2. Open SQLite at the effective `db.path`.
3. Apply the canonical DDL from `specs/done/schema-storage/spec.md` §2.2 for:
   - `users`,
   - `preferences`,
   - `vm_metadata`,
   - `audit_events`.
4. If schema application fails, fail fast: log the error and exit non-zero.

### 6.1.2 Git workspace initialization

1. Create git base directory (`git.path`) when missing.
2. Run `git init` when the base path is not already a git repository.
3. Create all required subdirectories:
   - `templates/`
   - `audit/`
   - `audit/vm/`
   - `audit/template/`
   - `audit/wizard/`
4. If workspace initialization fails, fail fast: log the error and exit non-zero.

Subdirectory requirements derive from `specs/done/schema-storage/spec.md` §2.4 and `specs/active/spec-audit-integration/spec.md` §4.1.

### 6.2 Setup startup

1. Resolve config path and fail to normal mode only if user disables setup route exposure.
2. Create/open DB at effective DB path and apply canonical schema.
3. Do not initialize git repositories nor attempt host connections unless setup endpoint explicitly validates host.
4. Register setup routes with auth bypass.
5. Start listener with same transport rules as normal startup.

### 6.3 Health and readiness assumptions

- No separate readiness route is introduced in this bootstrap spec.
- Operational health is inferred from server startup success and signal-aware lifecycle logs.

## §7 Shutdown Sequence

On `SIGINT` / `SIGTERM`:

1. Create shutdown context with bounded timeout (example: 30s).
2. Call `http.Server.Shutdown(ctx)`.
3. Close libvirt host handles (including any open pool/domain contexts).
4. Close DB connection.
5. Close git handles/resources if initialized.
6. Log `KUI shutdown complete`.

Any shutdown errors are logged and reflected as non-fatal after attempting all close steps.

## §8 TLS and Listen

- `--listen` flag sets the bind address.
- `KUI_LISTEN` env var provides default override only when flag is absent.
- Default listen address remains `:8080`.
- TLS is optional:
  - if both `--tls-cert` and `--tls-key` present, serve HTTPS.
  - if only one is present, fail fast before listening.
  - if neither is present, serve HTTP.
- Decision-log alignment:
  - optional HTTPS is supported,
  - production guidance remains reverse proxy + optional cert termination,
  - HTTP allowed for local development.

## §9 Dependencies

Canonical package responsibilities:

- `cmd/kui/main.go`
  - process startup/shutdown orchestration.
- `internal/config`
  - parse YAML and env overrides,
  - apply schema validation.
- `internal/db`
  - open/close SQLite,
  - initialize canonical schema from `specs/done/schema-storage/spec.md` §2.2.
- `internal/git`
  - initialize base path,
  - prepare `templates/` and `audit/` directories.
- `internal/libvirt`
  - connection validation helper,
  - host connector lifecycle.
- `internal/middleware`
  - request ID, logging, recovery, auth middleware.
- `internal/routes` or route modules
  - auth routes and setup routes from api-auth spec.
- external dependencies:
  - `github.com/go-chi/chi/v5` for routing,
  - `libvirt.org/go/libvirt` for connector integration,
  - standard `log/slog`, `os/signal`, `net/http`.

## §10 Tasks

| # | Task | Deliverable |
|---|------|-------------|
| 1 | Document and implement main entrypoint contract | `cmd/kui/main.go` bootstrap flow and mode switch |
| 2 | Implement config load + validation | `internal/config` uses YAML + env precedence and strict errors |
| 3 | Implement middleware stack | request-id, logging, recovery, JWT auth with route exclusions |
| 4 | Register bootstrap routes | chi route tree with static + auth/setup endpoints |
| 5 | Implement normal startup init | DB schema init, git bootstrap, libvirt connector init |
| 6 | Implement setup startup path | DB-only bootstrap + setup endpoints before config exists |
| 7 | Implement shutdown sequencing | deterministic resource close order with timeout |
| 8 | Implement TLS/listen decision logic | dual-mode HTTP/HTTPS startup with fail-fast config |
| 9 | Add bootstrap verification artifacts | spec checks for task compliance and contract alignment |

## Verification Checklist

- [x] Spec file exists under `specs/active/spec-application-bootstrap/spec.md`.
- [x] Total lines under 800.
- [x] Tasks table has fewer than 10 items.
- [x] No stub or placeholder runtime behavior is described for core bootstrap.
- [x] Greenfield constraints preserved (no migration/backfill/compatibility branches).
- [x] decision-log §§0–4 requirements reflected:
  - config path defaults and overrides,
  - systemd-first context and optional Docker/env override model,
  - TLS optional + reverse-proxy-recommended guidance,
  - first-run setup behavior when config/DB/admin missing.
- [x] Middleware order explicitly defined as request-id, logging, recovery, auth.
- [x] Route set is aligned with `specs/done/api-auth/spec.md`.
- [x] Config and env precedence are aligned with `specs/done/schema-storage/spec.md`.
