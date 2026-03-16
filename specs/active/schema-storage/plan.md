# Schema & Storage Spec — Plan

**Purpose:** Define the implementation plan for `spec.md` covering SQLite schema, Git storage layout, and Config YAML structure. Ready for developer to produce the spec document.

**References:** [docs/prd/decision-log.md](../../docs/prd/decision-log.md) §§0–4, [docs/prd.md](../../docs/prd.md) §2, [docs/prd/architecture.md](../../docs/prd/architecture.md), [docs/prd/stack.md](../../docs/prd/stack.md)

---

## 1. Exploration Summary

### Decision-log entries (schema, storage, config)

| Topic | Source | Key content |
|-------|--------|-------------|
| Database | §2 Canonical | SQLite: users, preferences, vm_metadata. Git: templates (full audit chain) + audit (diffs per entity). Path configurable; default /var/lib/kui/ |
| Audit structure | §2, §4 | Single audit table with event_type column. Git for diffs per entity changeset (MVP from day one) |
| VM metadata | §2, §4 | host_id + libvirt_uuid (composite key), claimed, display_name, console preference, last_access; provenance in audit |
| Preferences | §2, §4 | One row per user_id; default host + list view (columns, sort) as JSON blob |
| Users | §2 | Local auth; SQLite users table; MVP single admin only |
| Config | §2, §4 | YAML; hosts required; vm_defaults for pool+path; template_storage optional; Git path configurable; default /etc/kui/config.yaml; env for Docker |
| Clone defaults | §2, §4 | default_host, default_pool, default_name_template; MVP {source} only |
| Session | §2 | JWT stateless; timeout configurable in YAML; long default (24h or until browser close) |
| SSH keys | §2, §4 | Config (YAML) only; env override for Docker |

### Codebase state

- **Greenfield:** No Go code, no schema, no config loader. All design in docs.
- **Specs layout:** `.sdd/config.json` defines specs/active, specs/done, specs/backlog. Required: spec.md, plan.md, tasks.md.
- **Templates:** `.sdd/templates/spec-compact.md` — Problem, Users, Value; Must/Should Have; User Stories; Success Metrics; Dependencies; Out of Scope.

---

## 2. SQLite Schema (exact columns)

### 2.1 `users`

| Column | Type | Constraints |
|--------|------|--------------|
| id | TEXT | PRIMARY KEY (UUID v4) |
| username | TEXT | NOT NULL UNIQUE |
| password_hash | TEXT | NOT NULL |
| role | TEXT | NOT NULL DEFAULT 'admin' |
| created_at | TEXT | NOT NULL (ISO8601) |
| updated_at | TEXT | NULL |

**Notes:** MVP single admin; no RBAC. Password hashing (bcrypt/argon2) deferred to auth spec.

### 2.2 `preferences`

| Column | Type | Constraints |
|--------|------|--------------|
| user_id | TEXT | PRIMARY KEY, REFERENCES users(id) ON DELETE CASCADE |
| default_host_id | TEXT | NULL (references config host id) |
| list_view_options | TEXT | NULL (JSON: columns, sort) |
| updated_at | TEXT | NOT NULL (ISO8601) |

**Notes:** One row per user. JSON blob for list-view options (columns, sort order).

### 2.3 `vm_metadata`

| Column | Type | Constraints |
|--------|------|--------------|
| host_id | TEXT | NOT NULL, part of PK |
| libvirt_uuid | TEXT | NOT NULL, part of PK |
| claimed | INTEGER | NOT NULL (0/1 boolean) |
| display_name | TEXT | NULL |
| console_preference | TEXT | NULL (noVNC, xterm, SPICE; else libvirt fallback) |
| last_access | TEXT | NULL (ISO8601) |
| created_at | TEXT | NOT NULL (ISO8601) |
| updated_at | TEXT | NOT NULL (ISO8601) |

**Constraints:** PRIMARY KEY (host_id, libvirt_uuid). host_id must match config hosts.id.

### 2.4 `audit_events`

| Column | Type | Constraints |
|--------|------|--------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT |
| event_type | TEXT | NOT NULL (wizard_complete, vm_config_change, vm_lifecycle, template_create, auth) |
| entity_type | TEXT | NULL (vm, template, user, etc.) |
| entity_id | TEXT | NULL (composite or single id) |
| user_id | TEXT | NULL, REFERENCES users(id) |
| payload | TEXT | NULL (JSON: event-specific metadata) |
| git_commit | TEXT | NULL (SHA of audit git commit for diffs) |
| created_at | TEXT | NOT NULL (ISO8601) |

**Notes:** Event log in SQLite; full diffs stored in Git (see §3). git_commit links SQLite row to Git audit repo.

---

## 3. Git Directory Layout

**Base path:** Configurable in YAML; default `/var/lib/kui/`.

### 3.1 Templates (full audit chain)

```
<git_base>/templates/
├── <template_id>/
│   ├── meta.yaml          # name, base_image, cpu, ram, network, disk naming
│   └── domain.xml         # libvirt domain XML
```

- One directory per template; template_id = slug or UUID.
- Git history = full audit chain (create, edit, delete).
- No disk images in git; meta.yaml references pool/path.

### 3.2 Audit (diffs per entity changeset)

```
<git_base>/audit/
├── vm/
│   └── <host_id>/
│       └── <libvirt_uuid>/
│           └── <timestamp>.diff    # before/after for config changes
├── template/
│   └── <template_id>/
│       └── <timestamp>.diff
└── wizard/
    └── <timestamp>.diff
```

- One diff file per changeset; git commit per write.
- Diff format: unified diff or JSON before/after (spec to define).
- Links: audit_events.git_commit → git rev-parse HEAD at write time.

---

## 4. Config YAML Structure

**Path:** Default `/etc/kui/config.yaml`; override via `--config` or `KUI_CONFIG`.

### 4.1 Required sections

| Section | Required | Description |
|---------|----------|-------------|
| hosts | **Yes** | List of libvirt hosts. Setup wizard validates before save. |

**hosts** structure (each entry):

```yaml
hosts:
  - id: local          # unique id; referenced by preferences, vm_metadata
    uri: qemu:///system
    keyfile: null      # null for local; path for qemu+ssh
  - id: remote1
    uri: qemu+ssh://user@host.example/system?keyfile=/path/to/key
    keyfile: /path/to/key
```

### 4.2 Optional sections (with defaults)

| Section | Default | Description |
|---------|---------|-------------|
| vm_defaults | (see below) | CPU, RAM, network for pool+path create |
| default_host | first host id | Default host for create/clone/console |
| default_pool | null | Default storage pool for clone |
| default_name_template | "{source}" | MVP: {source} only; v2: {date}, {timestamp} |
| template_storage | null | Pool/path for save-as-template; user picks if missing |
| git | (see below) | Git base path |
| session | (see below) | Session timeout |
| db | (see below) | DB path |

**vm_defaults:**

```yaml
vm_defaults:
  cpu: 2
  ram_mb: 2048
  network: default   # libvirt network name
```

**git:**

```yaml
git:
  path: /var/lib/kui
```

**session:**

```yaml
session:
  timeout: 24h       # or "until_close"; parse as duration
```

**db:**

```yaml
db:
  path: /var/lib/kui/kui.db
```

### 4.3 Env overrides (Docker)

| Env var | Overrides |
|---------|-----------|
| KUI_CONFIG | Config file path |
| KUI_DB_PATH | db.path |
| KUI_GIT_PATH | git.path |
| KUI_*_KEYFILE | Per-host keyfile (e.g. KUI_REMOTE1_KEYFILE) |

---

## 5. Spec.md Structure (for developer)

The `spec.md` file should:

1. **What & Why** — Problem: schema/storage/config are undefined; Users: developers, operators; Value: canonical reference for implementation.
2. **Requirements**
   - Must: SQLite schema (4 tables), Git layout (templates + audit), Config YAML (required vs optional).
   - Should: Example config, env override table, security notes (no secrets in examples).
3. **User Stories** — Developer implements schema; operator configures hosts; setup wizard writes config.
4. **Success Metrics** — Schema applied; Git dirs created; config loads without error.
5. **Dependencies** — PRD decision log, stack.md, architecture.md.
6. **Out of Scope** — Migrations, backfill, backwards compatibility (greenfield).

**Format:** Adapt `.sdd/templates/spec-compact.md`; include full schema DDL, Git layout diagram, and config example.

**Target:** <800 lines or <10 tasks. Greenfield only. No stubs.

---

## 6. Deliverables

| Artifact | Location | Owner |
|----------|----------|-------|
| plan.md | specs/active/schema-storage/plan.md | (this document) |
| spec.md | specs/active/schema-storage/spec.md | developer subagent |
| tasks.md | specs/active/schema-storage/tasks.md | developer (optional; can follow spec) |

---

## 7. Verification

- [ ] spec.md exists and is <800 lines
- [ ] SQLite schema matches plan §2 (tables, columns, constraints)
- [ ] Git layout matches plan §3
- [ ] Config structure matches plan §4 (required vs optional)
- [ ] No migration paths or backwards-compatibility sections
- [ ] References decision-log, prd, architecture, stack
