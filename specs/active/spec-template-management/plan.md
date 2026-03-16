# Spec: Template Management — Plan

**Purpose:** Define the implementation plan for `spec.md` covering save-VM-as-template, list templates, template structure/validation, and API surface. Ready for developer to produce the spec document.

**References:** [docs/prd/decision-log.md](../../docs/prd/decision-log.md) §§0–4, [docs/prd/architecture.md](../../docs/prd/architecture.md), [specs/done/schema-storage/spec.md](../../done/schema-storage/spec.md)

---

## 1. Exploration Summary

### 1.1 Decision-log entries for templates

| Topic | Source | Key content |
|-------|--------|-------------|
| VM templates storage | §2 Canonical | Stored in Git (full audit chain); sharable when RBAC added |
| Template creation | §2, §4 | MVP: creation exists (de-emphasized); create VM from template in v2 |
| Template from VM | §4 Inquisition | Domain XML + copy of source disk. Copy destination: config template_storage first; else same pool as source |
| Template sources | §2, §4 | Pre-existing pools/paths; template_storage optional; user picks pool at save time if missing; else same pool as source |
| Template structure | §2, §4 | Name + base image required; CPU/RAM/network have defaults |
| Disk naming | §2 | MVP: {vm_name} only |
| Base image validation | §2, §4 | Validate pool exists and active via libvirt; validate path/volume in pool if user-specified |
| Empty state | §2, §4 | First-run checklist: "Create Template" in v2; template creation exists in MVP but de-emphasized |

### 1.2 Schema-storage Git layout (authoritative)

From `specs/done/schema-storage/spec.md` §2.4:

```
<git_base>/templates/
└── <template_id>/
    ├── meta.yaml
    └── domain.xml

<git_base>/audit/
└── template/
    └── <template_id>/
        └── <timestamp>.diff
```

- `template_id` = stable identifier (slug or UUID).
- Each template directory has full git history (create/edit/delete).
- No disk images in git; meta.yaml references pool+path for base image.
- `audit_events.git_commit` links to commit SHA at write time.

### 1.3 Config (from schema-storage)

- `template_storage` (optional, default null): pool name or pool+path for save-as-template.
- If null: user picks pool at save time; else same pool as source VM.
- `vm_defaults` (cpu, ram_mb, network) provide defaults for template meta.

### 1.4 Codebase state

- **Greenfield:** No Go implementation yet. All design in specs and docs.
- **Clone flow** (spec-vm-lifecycle-create): disk copy/stream, pool validation, domain XML build — reusable patterns for save-as-template.
- **Libvirt Connector** (spec-vm-lifecycle-create §9): pool lookup, volume lookup/create, copy/stream primitives — same primitives for template disk copy.

---

## 2. Save-VM-as-Template Flow

### 2.1 Inputs

- Source VM: `host_id`, `libvirt_uuid` (required).
- Template name (required; becomes `template_id` slug).
- Target pool (optional): user override; used when `template_storage` is null.

### 2.2 Pool resolution order

1. Request `target_pool` if user provided.
2. Config `template_storage` (pool name or pool+path).
3. Same pool as source VM disk.

### 2.3 Preconditions

- Source VM must exist and be **stopped** (same as clone).
- Target pool must exist and be active (validate via libvirt).

### 2.4 Flow steps

1. Validate source VM exists and is stopped.
2. Resolve target pool (request → config → source pool).
3. Validate target pool exists and is active (libvirt).
4. Copy source disk to target pool:
   - Same format as source.
   - Disk naming: MVP `{vm_name}` only (source VM display name or domain name).
5. Build domain XML for template:
   - Strip VM-specific identifiers (uuid, name) or use placeholder.
   - Update disk path to new copied volume path.
   - Preserve CPU/RAM/network from source or apply vm_defaults.
6. Build `meta.yaml`:
   - `name`, `base_image` (pool+path or pool+volume) required.
   - `cpu`, `ram_mb`, `network` with defaults.
7. Write to Git:
   - Create `<git_base>/templates/<template_id>/meta.yaml` and `domain.xml`.
   - Commit.
8. Write audit diff:
   - Create `<git_base>/audit/template/<template_id>/<timestamp>.diff`.
   - Commit; capture SHA.
9. Insert `audit_events` row:
   - `event_type`: `template_create`.
   - `entity_type`: `template`, `entity_id`: `template_id`.
   - `git_commit`: SHA from step 8.

### 2.5 Domain XML sanitization

- Remove or replace `uuid` (template is a blueprint).
- Replace `name` with placeholder or omit.
- Disk source must reference the **copied** volume path in target pool.

---

## 3. List Templates Flow

### 3.1 Source

- Read from Git: list directories under `<git_base>/templates/`.
- No SQLite table for templates; Git is source of truth.

### 3.2 Per-template data

- Parse `meta.yaml` for each template directory.
- Optionally validate `base_image` (pool+path) exists via libvirt — spec to define if validation is sync or lazy.
- Return: `template_id`, `name`, `base_image`, `cpu`, `ram_mb`, `network`, `created_at` (from git log if needed).

### 3.3 Error handling

- Malformed `meta.yaml` or missing `domain.xml`: skip or return error per template; do not fail entire list.

---

## 4. Template Structure

### 4.1 meta.yaml schema

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| name | Yes | — | Human-readable template name |
| base_image | Yes | — | `pool` + `path` or `pool` + `volume` (pool name and path or volume name) |
| cpu | No | vm_defaults.cpu | vCPU count |
| ram_mb | No | vm_defaults.ram_mb | RAM in MB |
| network | No | vm_defaults.network | Libvirt network name |

**base_image** representation: spec to define exact format (e.g. `pool:default;path:/path/to/disk.qcow2` or `pool:default;volume:disk.qcow2`).

### 4.2 domain.xml

- Libvirt domain XML.
- Disk element references `base_image` path.
- Stripped of VM-specific identifiers for template blueprint use.

### 4.3 Base image validation

- Validate pool exists and is active via libvirt (`virStoragePoolLookupByName`, `virStoragePoolGetInfo`).
- Validate path or volume exists in pool (`virStorageVolLookupByPath` or `virStorageVolLookupByName`).
- Same pattern as spec-vm-lifecycle-create §5 (Pool/Path Validation).

---

## 5. API Surface

### 5.1 Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| POST | /api/templates | Save VM as template |
| GET | /api/templates | List templates |

### 5.2 POST /templates (save VM as template)

**Request body:**
- `source_host_id`, `source_libvirt_uuid` (required).
- `name` (required; used for template_id slug).
- `target_pool` (optional; overrides config/source pool).

**Response:**
- `template_id`, `name`, `base_image`, `created_at` (or git commit timestamp).

### 5.3 GET /templates

**Response:**
- Array of `{ template_id, name, base_image, cpu, ram_mb, network }`.
- Optional: `base_image_valid` (boolean) if validation is performed.

### 5.4 Out of scope for MVP

- `GET /api/templates/{id}` — optional; list may be sufficient.
- `DELETE /api/templates/{id}` — optional for MVP.
- `PUT /api/templates/{id}` — edit template; optional.
- Create VM from template — v2.

---

## 6. Spec.md Structure (for developer)

The `spec.md` file should include:

1. **What & Why** — Problem: template management undefined; Users: operators, developers; Value: canonical reference for save-as-template and list flows.
2. **Requirements**
   - Must: save-VM-as-template flow (disk copy, domain XML, Git commit, audit); list templates from Git; template structure (meta.yaml, domain.xml); base image validation; API surface (POST/GET /templates).
   - Should: Error handling for malformed templates; base_image format specification.
3. **User Stories** — Operator saves VM as template; operator lists templates; developer implements against spec.
4. **Success Metrics** — Save produces valid template in Git with audit; list returns templates; base image validation works.
5. **Dependencies** — decision-log §§0–4, architecture.md, schema-storage spec.
6. **Out of Scope** — Create VM from template (v2); migration; stub implementations.

**Format:** Follow schema-storage spec style; include flow diagrams, meta.yaml schema, API request/response shapes.

**Constraints:**
- Target <800 lines or <10 tasks.
- Greenfield only.
- No stub implementations.

---

## 7. Deliverables

| Artifact | Location | Owner |
|----------|----------|-------|
| plan.md | specs/active/spec-template-management/plan.md | (this document) |
| spec.md | specs/active/spec-template-management/spec.md | developer subagent |

---

## 8. Verification

- [ ] spec.md exists and is <800 lines
- [ ] Save-VM-as-template flow matches plan §2
- [ ] List templates flow matches plan §3
- [ ] Template structure (meta.yaml, domain.xml) matches plan §4
- [ ] API surface matches plan §5
- [ ] No migration paths or backwards-compatibility sections
- [ ] References decision-log, architecture, schema-storage spec
