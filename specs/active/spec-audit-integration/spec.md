# Audit Integration Specification

## 1. What & Why

### 1.1 Problem

Audit behavior is currently underspecified for each feature area, leaving handlers without a consistent contract for:

- what events are recorded in SQLite,
- which events emit Git diffs,
- where Git diffs are written,
- what commit naming and payload structure should look like.

This spec defines the complete event contract so all handlers can emit audit records consistently from setup, VM operations, template changes, and authentication actions.

### 1.2 Users

- Developers implementing wizard, VM lifecycle, VM config, template save, and auth handlers.
- Operators verifying what changed and when, especially for setup/configuration actions.
- Future maintainers integrating UI or tooling against the audit trail.

### 1.3 Value

- Deterministic behavior by design: each event type has a fixed persistence path and payload shape.
- Queryability: all events remain in SQLite for fast searching and filtering.
- Audit detail: full before/after state is preserved in Git diffs for config-bearing events.
- Alignment with existing decisions: references PRD decision-log §§0–4 (scope, verbosity, retention) and schema-storage authoritative data model.

## 2. SQLite vs Git split

### 2.1 Authoritative rules

All events are written to SQLite `audit_events` for a complete indexed ledger.

- `audit_events.git_commit` is populated only when a diff file is written to Git.
- `audit_events.git_commit` is `NULL` when no Git diff exists (lifecycle-only or auth events).
- Git path and diff generation follow this spec whenever an event represents a before/after config change.

### 2.2 Event matrix

| Event type        | SQLite row | Git diff | Rationale |
|-------------------|------------|----------|-----------|
| `wizard_complete`  | Yes        | Yes      | Setup config writes are full config changes with traceable before/after state |
| `vm_config_change` | Yes        | Yes      | Domain XML and tracked metadata edits require before/after history |
| `vm_lifecycle`     | Yes        | No       | State transitions are sufficiently captured as payload metadata |
| `template_create`  | Yes        | Yes      | Template save creates new template files with traceable before/after state |
| `auth`             | Yes        | No       | Login/logout do not change tracked config objects |

### 2.3 Event-type coverage

This spec follows the schema-storage contract:

- `wizard_complete`
- `vm_config_change`
- `vm_lifecycle`
- `template_create`
- `auth`

No additional event types are introduced to avoid drift from canonical DDL.

### 2.4 Entity mapping for event rows

- `entity_type`:
  - `wizard`
  - `vm`
  - `template`
  - `auth`
- `entity_id`:
  - wizard: `setup-run-id` (opaque stable identifier if available, else `latest`)
  - vm: `<libvirt_uuid>`
  - template: `<template_id>`
  - auth: session-bound actor identifier (`user_id`)

## 3. Git commit behavior

### 3.1 Trigger semantics

- One commit per diff file write.
- If multiple files change in a handler, there is exactly one atomic audit commit for that handler’s audit diff.
- Only events with a diff emit a Git commit; lifecycle/auth events do not.

### 3.2 Commit ordering

For diff-backed events:

1. Write diff to an event-specific audit path file.
2. `git add` the new/changed diff path.
3. Commit with scope-specific message format.
4. Insert `audit_events` row in SQLite with `git_commit = <sha>`.
5. Return success.

For SQLite-only events:

1. Insert `audit_events` row directly with `git_commit = NULL`.
2. Return success.

This order is mandatory to keep SQLite rows traceable to Git commit identifiers.

### 3.3 Commit message convention

Use deterministic, grep-friendly commit messages:

`audit(<scope>): <event_type> <entity_summary>`

Examples:

- `audit(wizard): wizard_complete`
- `audit(vm): vm_config_change host=local uuid=abc-123`
- `audit(template): template_create id=base-ubuntu`

### 3.4 Failure behavior

- If diff write succeeds but commit fails, do not return success and do not persist the corresponding SQLite audit row.
- For commit failures, retry policy is implementation-local to caller handler; failures remain surfaced as handler errors so upstream can report clearly to the operator.
- A failed lifecycle/auth write is a handler error if SQLite insert fails.

## 4. Diff format and paths

## 4.1 Canonical references

Paths and timestamp format are defined in `specs/done/schema-storage/spec.md`:

- `audit/vm/<host_id>/<libvirt_uuid>/<timestamp>.diff`
- `audit/template/<template_id>/<timestamp>.diff`
- `audit/wizard/<timestamp>.diff`
- `<timestamp>` format: `20060102T150405Z` (UTC)

### 4.2 Diff format

- Unified diff text (`*.diff`) for every diff-backed event.
- Diffs are one file per atomic changeset.
- Diff payload contains before/after representation (text form used by the object being tracked).
- `audit_events.payload` remains summary metadata; diff file remains source of full state difference.

### 4.3 Diff content by domain

- `wizard`:
  - Before: empty file content or previous effective config if available.
  - After: complete serialized config written by wizard.
- `vm_config_change`:
  - Before: serialized previous config state (domain XML and/or vm metadata JSON, depending on what changed).
  - After: serialized new state.
- `template`:
  - Before: previous template files (`meta.yaml`, `domain.xml`) state.
  - After: new template files state.

## 5. Integration points

### 5.1 Setup wizard

- Trigger: setup completion action (`POST /api/setup/complete`).
- SQLite event:
  - `event_type`: `wizard_complete`
  - `entity_type`: `wizard`
  - `entity_id`: setup run identifier
  - `user_id`: actor account id if authenticated or system actor id
- Git:
  - Path: `audit/wizard/<timestamp>.diff`
  - Includes full config write before/after diff.
- Notes:
  - Event is emitted after successful config persistence.
  - If this path fails at any stage, setup handler returns failure and does not report audit success.

### 5.2 VM create/clone

- Triggers:
  - VM create: `POST /api/vms`
  - VM clone: `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/clone`
- SQLite event:
  - `event_type`: `vm_lifecycle`
  - `entity_type`: `vm`
  - `entity_id`: new `libvirt_uuid`
  - Payload includes transition summary:
    - `action`: `create` / `clone`
    - `host_id`
    - `source_uuid` (clone only)
- Git:
  - No diff written.
  - `git_commit` remains `NULL`.
- Rationale:
  - New-object operations are lifecycle events, not in-situ config edits.

### 5.3 VM lifecycle

- Triggers:
  - `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/start`
  - `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/stop`
  - `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/pause`
  - `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/resume`
  - `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/destroy`
- SQLite event:
  - `event_type`: `vm_lifecycle`
  - `entity_type`: `vm`
  - `entity_id`: `libvirt_uuid`
  - Payload:
    - `action`: lifecycle verb
    - `from_state`
    - `to_state`
    - `host_id`
- Git:
  - No diff.
  - `git_commit` remains `NULL`.

### 5.4 VM config change

- Triggers:
  - VM edit endpoint: `PATCH /api/hosts/{host_id}/vms/{libvirt_uuid}` (see `specs/active/spec-vm-lifecycle-create/spec.md` §7).
- SQLite event:
  - `event_type`: `vm_config_change`
  - `entity_type`: `vm`
  - `entity_id`: `libvirt_uuid`
  - Payload contains changed field summaries and before/after metadata pointers.
- Git:
  - Path: `audit/vm/<host_id>/<libvirt_uuid>/<timestamp>.diff`
  - Includes full before/after for domain XML and/or `vm_metadata` fields.
- Rationale:
  - Full auditability is required for config edits by PRD decision-log.

### 5.5 Template save

- Trigger:
  - Save VM as template action (`POST /api/templates` or template-save endpoint).
- SQLite event:
  - `event_type`: `template_create`
  - `entity_type`: `template`
  - `entity_id`: `template_id`
  - Payload includes source VM, template metadata, and storage target.
- Git:
  - Path: `audit/template/<template_id>/<timestamp>.diff`
- Notes:
  - Template file changes (`meta.yaml`, `domain.xml`) are represented as one atomic changeset diff.

### 5.6 Auth login/logout

- Triggers:
  - `POST /api/auth/login`
  - `POST /api/auth/logout`
- SQLite event:
  - `event_type`: `auth`
  - `entity_type`: `auth`
  - `entity_id`: actor id or `user_<id>` token
  - Payload:
    - `action`: `login` / `logout`
    - `session_id` if available
    - `result`: `success|failure`
    - `remote_ip`
    - `user_agent`
- Git:
  - No diff.
  - `git_commit` remains `NULL`.

## 6. audit_events payload shape

All payloads are JSON objects encoded as text, and should be query-friendly with explicit fields for filtering and analytics.

### 6.1 Common audit row fields

For all events:

- `event_type`
- `entity_type`
- `entity_id`
- `user_id`
- `payload` (JSON text)
- `git_commit`
- `created_at`

### 6.2 Payload examples

- `wizard_complete`

```json
{
  "action": "wizard_complete",
  "result": "success",
  "admin_username": "admin",
  "host_id": "local",
  "config_path": "/etc/kui/config.yaml",
  "git_path": "/var/lib/kui"
}
```

- `vm_lifecycle`

```json
{
  "action": "start",
  "from_state": "shutoff",
  "to_state": "running",
  "host_id": "local"
}
```

- `vm_config_change` (VM)

```json
{
  "action": "update",
  "changed_fields": ["display_name", "ram_mb", "domain_xml"],
  "host_id": "local",
  "libvirt_uuid": "b4c7...",
  "diff_path": "audit/vm/local/b4c7.../20260102T150405Z.diff",
  "before_ref": "vm_meta.sha256_before",
  "after_ref": "vm_meta.sha256_after"
}
```

- `template_create`

```json
{
  "action": "template_save",
  "template_id": "template-ubuntu-01",
  "source_libvirt_uuid": "b4c7...",
  "template_storage_path": "/var/lib/kui/templates/template-ubuntu-01",
  "diff_path": "audit/template/template-ubuntu-01/20260102T150405Z.diff"
}
```

- `auth`

```json
{
  "action": "login",
  "result": "success",
  "username": "admin",
  "ip": "192.168.1.10",
  "user_agent": "kui-web/0.1",
  "session_id": "sess-001"
}
```

## 7. User stories

- As an operator, I can trace setup wizard writes and review before/after config changes from one deterministic audit path.
- As an operator, I can see lifecycle actions (create/clone/start/stop/pause/resume/destroy) in queryable audit rows.
- As an operator, I can inspect VM config edits with full before/after context in Git, and quickly locate the commit.
- As an operator, I can review template save audit for provenance and source VM linkage.
- As an operator, I can audit auth activity by actor and result without exposing potentially sensitive payload state.
- As a developer, I can implement handlers by looking up one event contract per route and know exactly when to write Git.

## 8. Success metrics

- 100% of sectioned handlers emit an `audit_events` row per successful operation.
- 100% of diff-backed events emit both:
  - one `.diff` file under the correct `audit/...` path, and
  - matching `audit_events.git_commit` SHA.
- 0 diff-backed handler returns success when `audit` persistence is incomplete.
- Every `vm_lifecycle` and `auth` event persists with `git_commit = NULL`.
- Every `template_create` event persists with `git_commit` populated (diff-backed).
- Audit review can recover full before/after for wizard, VM config edits, and template saves via file diffs and Git metadata.
- Retention behavior follows PRD: unlimited by default, optional retention documented as non-normative v2 concern.

## 9. Dependencies

- `specs/done/schema-storage/spec.md` (authoritative SQLite schema, Git layout, diff format, timestamp format).
- `docs/prd/decision-log.md` §§0–4 (scope, verbosity, retention constraints).
- `docs/prd.md` open assumptions for audit git storage.
- `specs/active/api-auth` endpoint contract for login/logout action behavior.
- `specs/active/spec-vm-lifecycle-create/spec.md` for VM operation flow.
- `specs/active/spec-template-management/spec.md` for template file boundaries.

## 10. Out of scope

- Notification/audit panel UI (v2).
- Storage backends other than SQLite + Git for audit in this architecture.
- Migration, backfill, compatibility-mode, or dual-writing paths.
- Retention policy implementation and purging tooling in this spec (recorded in PRD as not required for MVP).
- Storing VM disk/image bytes in Git.
