# Spec: Audit Integration — Plan for spec.md

## Overview

Create `spec.md` that specifies how audit events are written to SQLite and Git across all integration points (wizard, VM lifecycle, VM config, template save, auth). The spec resolves the "when SQLite vs Git" split, Git commit triggers, diff format, commit message convention, and integration points. Target: <800 lines or <10 tasks; greenfield only; no stubs.

**References:**
- `docs/prd/decision-log.md` §§0–4 (Audit scope, verbosity, retention)
- `docs/prd.md` §2 Open Assumptions (audit git storage)
- `specs/active/schema-storage/spec.md` (Git layout, audit_events DDL, diff format)

---

## Exploration Findings

### 1. Schema-storage spec (authoritative)

- **SQLite `audit_events`:** id, event_type, entity_type, entity_id, user_id, payload, git_commit, created_at
- **event_type values:** wizard_complete, vm_config_change, vm_lifecycle, template_create, auth
- **Git audit layout:** `<git_base>/audit/` with subdirs:
  - `vm/<host_id>/<libvirt_uuid>/<timestamp>.diff`
  - `template/<template_id>/<timestamp>.diff`
  - `wizard/<timestamp>.diff`
- **No `auth/` subdir** — auth events (login/logout) have no config diff; SQLite-only
- **Diff format:** unified diff text
- **Timestamp format:** `20060102T150405Z` (UTC)
- **audit_events.git_commit:** links SQLite row to the Git commit produced by the diff write; NULL when no Git diff

### 2. SQLite vs Git split

| Event type        | SQLite | Git diff | Rationale                                      |
|-------------------|--------|----------|------------------------------------------------|
| wizard_complete   | Yes    | Yes      | Config written; before/after in audit/wizard/  |
| vm_config_change  | Yes    | Yes      | Domain XML or vm_metadata change; before/after |
| vm_lifecycle      | Yes    | No       | State transition; payload sufficient           |
| template_create   | Yes    | Yes      | Template files; diff in audit/template/       |
| auth              | Yes    | No       | Login/logout; no config diff                   |

- **SQLite:** Always write for every event. Queryable index for audit log.
- **Git:** Only when there is a before/after diff. `git_commit` NULL for lifecycle and auth.

### 3. Integration points (from specs)

| Feature        | Spec / endpoint                    | Audit event_type   | Git path (if diff)                    |
|----------------|------------------------------------|--------------------|---------------------------------------|
| Setup wizard   | api-auth: POST /api/setup/complete  | wizard_complete    | audit/wizard/<timestamp>.diff          |
| VM create      | spec-vm-lifecycle-create: POST /api/vms       | vm_lifecycle       | No (create = lifecycle, not config)    |
| VM clone       | spec-vm-lifecycle-create: POST /api/hosts/{id}/vms/{uuid}/clone | vm_lifecycle    | No                                    |
| VM lifecycle   | spec-vm-lifecycle-create: start/stop/pause/resume/destroy | vm_lifecycle | No                                    |
| VM config edit | (future: domain XML edit)          | vm_config_change   | audit/vm/<host>/<uuid>/<ts>.diff       |
| Template save  | (template spec; save VM as template) | template_create         | audit/template/<id>/<ts>.diff   |
| Auth login     | api-auth: POST /api/auth/login     | auth               | No                                    |
| Auth logout    | api-auth: POST /api/auth/logout    | auth               | No                                    |

**Clarification:** VM create/clone insert vm_metadata and define domain. Decision-log says "full before/after for config changes." Create/clone are lifecycle (new entity), not config edits. Treat as vm_lifecycle, SQLite-only. VM config change = edits to existing domain XML or vm_metadata (display_name, console_preference).

**Template save:** Creates template in `<git_base>/templates/<id>/`. The template dir has full git history. Per schema-storage, audit diffs go to `audit/template/<id>/<ts>.diff`. One commit can include both templates/ and audit/ changes.

### 4. Git commit triggers

- **One commit per atomic changeset.** Each diff file write is one commit.
- **When:** Immediately after writing the diff file, before returning from the handler.
- **Transaction order:** 1) Write diff file to audit path; 2) Git add + commit; 3) Insert audit_events row with git_commit = HEAD; 4) Return. For events without diff: insert audit_events only.

### 5. Commit message convention

Proposed format: `audit(<scope>): <event_type> <entity_summary>`

Examples:
- `audit(wizard): wizard_complete`
- `audit(vm): vm_config_change host=local uuid=abc-123`
- `audit(template): template_create id=my-template`

Keep messages short, deterministic, grep-friendly.

### 6. Diff content rules

- **wizard:** Diff of config YAML. Before = empty (first run) or previous config if ever supported. After = written config. Unified diff format.
- **vm_config_change:** Diff of domain XML or vm_metadata JSON. Before = previous state; after = new state.
- **template:** Diff of template files (meta.yaml, domain.xml). Before = empty (create) or previous; after = new content.

---

## Spec Structure (for developer)

The `spec.md` shall have these sections:

### 1. What & Why
- Problem: Audit integration is unspecified; each feature needs to know when and how to write SQLite + Git.
- Users: Developer implementing wizard, VM, template, auth handlers; operator reviewing audit trail.
- Value: Deterministic audit trail; queryable SQLite; full before/after in Git.

### 2. SQLite vs Git split
- Table: event_type → SQLite always, Git when diff exists.
- Rules: git_commit NULL for lifecycle and auth; populated for wizard, vm_config_change, template_create.

### 3. Git commit behavior
- Trigger: one commit per diff file write.
- Order: write diff → git add → commit → insert audit_events with git_commit.
- Message format: `audit(<scope>): <event_type> <entity_summary>`.

### 4. Diff format and paths
- Reference schema-storage §2.4, §2.5.
- Path rules per entity type.
- Unified diff content rules (before/after).

### 5. Integration points (per feature)
- **5.1 Setup wizard:** POST /api/setup/complete → wizard_complete, audit/wizard/<ts>.diff.
- **5.2 VM create/clone:** vm_lifecycle, SQLite only.
- **5.3 VM lifecycle (start/stop/etc):** vm_lifecycle, SQLite only; payload = {action, from_state, to_state}.
- **5.4 VM config change:** vm_config_change, audit/vm/<host>/<uuid>/<ts>.diff (when spec exists).
- **5.5 Template save:** audit/template/<id>/<ts>.diff; event_type template_create.
- **5.6 Auth login/logout:** auth, SQLite only; payload = {action: "login"|"logout"}.

### 6. audit_events payload shape
- JSON payload examples per event_type for query-friendly metadata.

### 7. User stories
- As operator: audit trail for wizard, VM, template, auth.
- As developer: clear integration contract per handler.

### 8. Success metrics
- All integration points write audit_events.
- Git diffs produced for wizard, vm_config_change, template_create.
- git_commit linked correctly.

### 9. Dependencies
- schema-storage, api-auth, spec-vm-lifecycle-create.

### 10. Out of scope
- Retention policy, notification panel audit log (v2).

---

## Tasks (for spec.md authoring)

1. Write §1 What & Why.
2. Write §2 SQLite vs Git split (table + rules).
3. Write §3 Git commit behavior.
4. Write §4 Diff format and paths (reference schema-storage).
5. Write §5 Integration points (all six subsections).
6. Write §6 audit_events payload shape.
7. Write §7 User stories.
8. Write §8 Success metrics.
9. Write §9 Dependencies, §10 Out of scope.
10. Review for <800 lines, <10 tasks; greenfield only.

---

## Deliverable

Create `specs/active/spec-audit-integration/spec.md` following the structure above. No plan.md edits needed after spec is written.
