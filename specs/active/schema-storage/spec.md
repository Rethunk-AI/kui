# Schema & Storage Specification

## 1. What & Why

### 1.1 Problem
The schema, Git-backed template/audit storage, and runtime config contract are still unset in implementation terms. KUI needs one authoritative storage design that enables:

- Stable VM metadata tracking across libvirt domains and hosts.
- Deterministic auditability for setup, lifecycle, auth, and configuration edits.
- Reproducible template lifecycle and change history in git.
- A single validated config shape used by all startup paths.

### 1.2 Users

- Primary operator: single admin using KUI to manage 1–2 concurrent VM users.
- Developer/operator implementing MVP backend modules for persistence, config loading, and audit/history.

### 1.3 Value

- Prevents drift between SQLite, git history, and config interpretation.
- Creates predictable paths for query logic (`vm_metadata`, `preferences`, `audit_events`) and operational troubleshooting.
- Enables safe implementation of setup wizard and import/claim flows with auditable records.

## 2. Requirements

### 2.1 Must Have

1. **Canonical SQLite schema** with `users`, `preferences`, `vm_metadata`, `audit_events`.
2. **Git storage layout** with:
   - full template audit chain under `<git_base>/templates/...`
   - diff-per-changeset audit records under `<git_base>/audit/...`
3. **Config schema** where:
   - `hosts` is required
   - `vm_defaults`, `default_host`, `default_pool`, `default_name_template`, `template_storage`, `git`, `session`, `db` are optional with defaults
4. **Config loading contract** for env-based overrides of critical values.
5. **Greenfield implementation** only; no migration mode or backwards-compatibility branches.

### 2.2 Full SQLite DDL (authoritative)

```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'admin',
    created_at TEXT NOT NULL,
    updated_at TEXT NULL
);

CREATE TABLE preferences (
    user_id TEXT PRIMARY KEY,
    default_host_id TEXT NULL,
    list_view_options TEXT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT fk_preferences_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE vm_metadata (
    host_id TEXT NOT NULL,
    libvirt_uuid TEXT NOT NULL,
    claimed INTEGER NOT NULL DEFAULT 0,
    display_name TEXT NULL,
    console_preference TEXT NULL,
    last_access TEXT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (host_id, libvirt_uuid)
);

CREATE TABLE audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    entity_type TEXT NULL,
    entity_id TEXT NULL,
    user_id TEXT NULL,
    payload TEXT NULL,
    git_commit TEXT NULL,
    created_at TEXT NOT NULL,
    CONSTRAINT fk_audit_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);
```

### 2.3 SQLite constraints and behaviors

- `users.id` is a UUID v4 (TEXT) generated at user creation.
- `vm_metadata.claimed` is a boolean flag encoded as integer `0/1`.
- `vm_metadata.host_id` must reference a configured host `id` from runtime config.
- `audit_events.event_type` supports at minimum:
  - `wizard_complete`
  - `vm_config_change`
  - `vm_lifecycle`
  - `template_create`
  - `auth`
- `audit_events.git_commit` stores the repository commit SHA associated with the persisted audit diff.

#### preferences.list_view_options (JSON)

The `list_view_options` column stores a JSON object. Structure matches `GET /api/preferences` response (see `specs/active/api-auth/spec.md` §2). Supported keys:

- `group_by` (optional): `"last_access"` | `"created_at"` — VM list grouping key; default `"last_access"`. May be nested under `list_view`.
- `onboarding_dismissed` (optional): `true` when the user has dismissed the first-run checklist; default `false` when absent.

### 2.4 Git directory layout (templates + audit)

Base path default: `/var/lib/kui/`.

```text
<git_base>/templates/
└── <template_id>/
    ├── meta.yaml
    └── domain.xml

<git_base>/audit/
├── vm/
│   └── <host_id>/
│       └── <libvirt_uuid>/
│           └── <timestamp>.diff
├── template/
│   └── <template_id>/
│       └── <timestamp>.diff
└── wizard/
    └── <timestamp>.diff
```

- `template_id` is a stable identifier chosen by creator (slug or UUID).
- Each template directory in `templates/` has full git history (create/edit/delete), storing source files only (no disk images).
- `audit` entries are one file per atomic changeset commit.
- `<timestamp>` format: `20060102T150405Z` (UTC, stable sorting).
- `audit_events.git_commit` points to the commit produced by the corresponding `*.diff` write.

### 2.5 Audit diff format

- Diff file contents are **unified diff** text.
- The diff payload describes changes to the tracked object JSON/text form at the time of the event.
- The corresponding `audit_events.payload` stores event metadata summary to keep SQLite query-friendly.
- The full before/after detail is recovered from the git diff archive.

### 2.6 Config YAML structure

#### Required sections

- `hosts` is required and is a list of libvirt connection targets.
- `jwt_secret` is required for normal mode (setup mode exempt): string, min 32 bytes; env override `KUI_JWT_SECRET`.

```yaml
hosts:
  - id: local
    uri: qemu:///system
    keyfile: null
```

#### Optional sections with defaults

- `vm_defaults` (default shown below)
  - `cpu: 2`
  - `ram_mb: 2048`
  - `network: default`
- `default_host` (default: first entry in `hosts`)
- `default_pool` (default: null)
- `default_name_template` (default: `"{source}"`)
- `template_storage` (default: null)
- `git`
  - `path` default: `/var/lib/kui`
- `session`
  - `timeout` default: `24h`
- `vm_lifecycle`
  - `graceful_stop_timeout` default: `30s` — timeout before force stop on graceful shutdown
- `cors`
  - `allowed_origins` (default: `["http://localhost:5173"]` for dev; prod: explicit list). Env override `KUI_CORS_ORIGINS`.
- `db`
  - `path` default: `/var/lib/kui/kui.db`

```yaml
vm_defaults:
  cpu: 2
  ram_mb: 2048
  network: default
  # optional: disk_mb: 10000

default_host: local
default_pool: null
default_name_template: "{source}"
template_storage: null

git:
  path: /var/lib/kui

session:
  timeout: 24h

vm_lifecycle:
  graceful_stop_timeout: 30s

cors:
  allowed_origins: ["http://localhost:5173"]

db:
  path: /var/lib/kui/kui.db
```

#### Environment variable overrides

| Env var | Config override | Notes |
|---------|-----------------|-------|
| `KUI_CONFIG` | config path | default `/etc/kui/config.yaml` |
| `KUI_DB_PATH` | `db.path` | full sqlite file path |
| `KUI_GIT_PATH` | `git.path` | defaults to `/var/lib/kui` if omitted |
| `KUI_HOST_<ID>_KEYFILE` | `hosts[].keyfile` | per-host keyfile override (ID uppercase, non-alnum collapsed to `_`) |
| `KUI_DEFAULT_HOST` | `default_host` | host ID string |
| `KUI_DEFAULT_POOL` | `default_pool` | storage pool name |
| `KUI_SESSION_TIMEOUT` | `session.timeout` | duration string |
| `KUI_JWT_SECRET` | `jwt_secret` | JWT signing secret, min 32 bytes |
| `KUI_CORS_ORIGINS` | `cors.allowed_origins` | comma-separated list of allowed origins |

### 2.7 Minimal valid `config.yaml`

Normal mode requires `jwt_secret`; setup mode does not.

```yaml
hosts:
  - id: local
    uri: qemu:///system
    keyfile: null

default_host: local
default_name_template: "{source}"
vm_defaults:
  cpu: 2
  ram_mb: 2048
  network: default

jwt_secret: "<base64-or-hex-32-bytes-minimum>"

db:
  path: /var/lib/kui/kui.db

git:
  path: /var/lib/kui

session:
  timeout: 24h

vm_lifecycle:
  graceful_stop_timeout: 30s
```

### 2.8 Required security posture in config/schema docs

- Password hashes belong only in `users.password_hash` (no clear text fields).
- `host` credentials are represented as `keyfile` paths, never inline secrets.
- No secrets or tokens in example files.

## 3. User Stories

- **As an operator,** I can create a VM-backed tracking record with host + uuid so VM metadata is claimable and auditable.
  - AC: a row in `vm_metadata` is inserted/updated with non-null `host_id` and `libvirt_uuid`.
  - AC: `audit_events` captures each tracked lifecycle change with a matching `git_commit` link.

- **As a developer,** I can initialize persistence from this spec so schema setup is deterministic.
  - AC: DB startup applies all four table definitions as listed in §2.2.
  - AC: all non-null and key constraints are enforced by SQLite.

- **As an operator,** I can run setup wizard with required hosts and optional defaults configured.
  - AC: missing `hosts` blocks startup and prevents write/commit.
  - AC: wizard writes config with a concrete `hosts` list and optional defaults.

- **As an operator,** I can inspect config overrides and apply env adjustments in containerized or privileged contexts.
  - AC: documented env vars override matching YAML fields.
  - AC: unresolved or malformed values fail fast at startup.

## 4. Success Metrics

- Schema creation includes all required tables and keys with no missing constraints.
- `users`, `preferences`, `vm_metadata`, and `audit_events` can be inserted and queried in one transaction cycle.
- Templates and audit repositories initialize at `<git.path>/templates` and `<git.path>/audit`.
- At least one template history commit and one audit diff commit include matching `audit_events.git_commit`.
- Config parser accepts the minimal config example and rejects missing `hosts`.
- Runtime startup succeeds with env override precedence over file values for listed variables.

## 5. Dependencies

- `docs/prd/decision-log.md`
- `docs/prd.md`
- `docs/prd/architecture.md`
- `docs/prd/stack.md`

## 6. Out of Scope

- Migration workflows, compatibility modes, and backfill scripts.
- PostgreSQL or other non-SQLite databases.
- Storage of VM disk images inside git.
- Long-term policy features deferred to later versions (roles, alerts panel audit retention policy, advanced import/export tools, etc.).

