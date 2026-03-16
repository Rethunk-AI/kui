# Spec: VM Lifecycle & Create — Plan for spec.md

## Overview

Create `spec.md` that formally specifies VM create (pool+path, clone), lifecycle operations, pool/path validation, and orphan claim flow for KUI MVP. The spec consolidates decision-log entries (§§0–4), architecture, and libvirt research into an implementable requirements document. Target: <800 lines or <10 tasks; greenfield only; MVP scope: pool+path + clone only; template-based create in v2.

**References to cite:**
- `docs/prd/decision-log.md` §§0–4
- `docs/prd/architecture.md` (data flow, libvirt connector)
- `docs/research/kui-libvirt-research.md` (libvirt API, test driver)

---

## Spec Structure

The spec.md shall have the following sections. Each maps to decision-log topics and scope items.

### 1. Scope & Constraints

- **Content:** Spec scope summary (pool+path create, clone, lifecycle, pool/path validation, orphan claim). Constraints: greenfield, <800 lines, MVP = pool+path + clone only; template-based create in v2. Libvirt Connector owns connection and domain/storage queries; API layer owns provisioning logic.
- **Sources:** User-provided scope; decision-log §2 (Docs scope, MVP scope, VM create flow, KVM interaction); architecture §2 component boundaries.

### 2. Pool+Path Create Flow

- **Content:**
  - **Config:** `vm_defaults` (CPU, RAM, network) required in YAML for pool+path create. `hosts` required; per-host URI + keyfile. Default host from config; user override in preferences.
  - **User inputs:** Host, pool, disk (pick existing volume OR auto-gen path + size). User can override CPU, RAM, network, disk path/name at create time.
  - **Validation:** Pool exists and active via libvirt; path/volume in pool if user-specified. Reject if pool inactive or path invalid.
  - **Flow:** 1) Validate pool/path; 2) Build domain XML from vm_defaults + overrides; 3) Create disk if auto-gen; 4) DefineDomain; 5) CreateDomain (optional auto-start — defer to spec); 6) Insert vm_metadata (host_id, libvirt_uuid, claimed=true, display_name).
  - **Disk options:** Prefer pick existing volume or auto-gen path + size; user override with explicit path/name allowed.
- **Sources:** decision-log §2 (VM create flow, Config, Pool/path validation, VM metadata), §4 (VM create disk, Pool/path validation).

### 3. Clone Flow

- **Content:**
  - **Inputs:** Source VM (host_id + libvirt_uuid), target host, target pool. Name optional.
  - **Name template:** MVP: `{source}` only; `{date}`, `{timestamp}` in v2. YAML defaults: default_host, default_pool, default_name_template. User preferences override.
  - **Cross-host:** Supported. Copy/stream disk to target host; same format as source. Libvirt has no native cross-host clone — KUI implements copy/stream (e.g., qemu-img convert over SSH, or libvirt stream APIs).
  - **Same-host:** Use virStorageVolClone or virDomainClone where applicable.
  - **Disk format:** Same format as source (no conversion in MVP unless required for target pool type).
  - **Flow:** 1) Validate source VM exists and is stopped (or define policy); 2) Validate target pool; 3) Copy/stream disk to target; 4) Build domain XML with new disk paths, same format; 5) DefineDomain on target; 6) Insert vm_metadata.
- **Sources:** decision-log §2 (Clone implementation), §4 (Clone implementation, Clone scope).

### 4. Lifecycle Operations

- **Content:**
  - **List:** libvirt domain list + state; vm_metadata for KUI fields. Flat list across hosts.
  - **Start:** virDomainCreate (or equivalent).
  - **Stop:** Graceful first (virDomainShutdown / virDomainShutdownFlags); configurable timeout (e.g., 30s default); if timeout, force (virDomainDestroy). Spec must define timeout value and config location.
  - **Pause:** virDomainSuspend.
  - **Resume:** virDomainResume.
  - **Destroy:** virDomainDestroy (no grace; immediate).
  - **State handling:** Handle transient vs persistent domains; document behavior for each.
- **Sources:** decision-log §2 (MVP lifecycle, KVM interaction), §4 (VM create flow: "graceful first; force if needed").

### 5. Pool/Path Validation

- **Content:**
  - Validate pool exists via libvirt (e.g., virStoragePoolLookupByName, virStoragePoolGetInfo).
  - Validate pool is active; if inactive, reject or document activation behavior.
  - If user specifies path/volume: validate path or volume exists in pool (virStorageVolLookupByPath or virStorageVolLookupByName).
  - Validation happens before create/clone operations; return user-friendly error on failure.
- **Sources:** decision-log §2 (Pool/path validation), §4 (Pool/path validation).

### 6. Orphan Claim Flow

- **Content:**
  - **Discovery:** List all libvirt domains per host; compare with vm_metadata (host_id + libvirt_uuid). Domains not in vm_metadata (or claimed=false) are orphans.
  - **UI:** Orphans in separate section/tab (see spec-ui-deployment §6.2).
  - **Claim:** Add to vm_metadata with host_id from host where orphan was discovered; claimed=true; display_name from libvirt name or user input.
  - **Flow:** User selects orphan → Claim → API inserts vm_metadata row; no domain XML change.
- **Sources:** decision-log §2 (Orphan domains), §4 (Orphan handling, Import/claim flow).

### 7. Data Model (vm_metadata)

- **Content:**
  - Composite key: host_id + libvirt_uuid.
  - Columns: claimed, display_name, optional console_preference, last_access.
  - Provenance in audit (decision-log).
  - Reference schema for create, clone, claim flows.
- **Sources:** decision-log §2 (VM metadata, Database), §4 (VM metadata scope).

### 8. API Surface (Informal)

- **Content:**
  - REST/JSON internal first; code is the contract (decision-log §2 API style).
  - Endpoints implied by flows: POST /vms (create), POST /vms/clone (clone), lifecycle actions (start/stop/pause/resume/destroy), POST /vms/:id/claim (orphan claim).
  - Request/response shapes for create and clone (host_id, pool, disk options; source VM, target host, target pool).
  - No formal OpenAPI in MVP; document in spec for implementation.
- **Sources:** decision-log §2 (API style); architecture §3 data flow.

### 9. Libvirt Connector Responsibilities

- **Content:**
  - Connection management: per-host URI + keyfile; qemu+ssh for remote.
  - Domain ops: list, define, create, shutdown, destroy, suspend, resume.
  - Storage: pool lookup, volume lookup, volume create, volume clone/copy.
  - Test driver: test:///default for CI; no mocks.
  - Out of scope for Connector: VM provisioning logic (API layer).
- **Sources:** docs/prd/architecture.md §2; docs/research/kui-libvirt-research.md; decision-log §1 (Go bindings, test driver).

### 10. Out of Scope

- Template-based create flow (v2).
- Storage/network management (use existing only).
- Migration, backfill, backwards compatibility.
- Docker deployment (post-MVP).
- Clone progress (%, ETA, stage) — v2 real-time.
- default_name_template {date}, {timestamp} — v2.
- HTTP/URL template sources — post-MVP.
- Connection pooling strategy — defer to implementation; Connector owns.
- Formal OpenAPI spec — MVP informal.

---

## Implementation Instructions for Developer

1. Create `specs/active/spec-vm-lifecycle-create/spec.md`.
2. Write each section (1–10) per the structure above. Use declarative requirements; cite decision-log entries where binding.
3. Keep total length <800 lines. Use concise bullets and tables.
4. Do not invent new requirements; derive only from decision-log, architecture, and libvirt research.
5. For lifecycle §4: define graceful-stop timeout (recommend 30s default), config path (e.g., `vm_lifecycle.graceful_stop_timeout` in YAML), and libvirt API choice (virDomainShutdown or virDomainShutdownFlags). Research libvirt C API if needed.
6. For clone §3: document cross-host implementation approach (copy/stream) without over-specifying; leave room for implementation.
7. For create §2: clarify whether auto-start after create is in or out of scope; if in, document.
8. For clone §3: define source VM state requirement (e.g., must be stopped).
9. Format: Markdown with clear headings (##, ###). Include a brief "References" section linking to docs/prd/decision-log.md, architecture.md, and kui-libvirt-research.md.

---

## Verification

- [ ] spec.md exists at specs/active/spec-vm-lifecycle-create/spec.md
- [ ] All 10 sections present and populated
- [ ] No migration/backfill/backwards-compatibility language (greenfield)
- [ ] Line count <800
- [ ] Decision-log citations accurate
- [ ] Graceful-stop timeout and config defined
- [ ] Libvirt API choices documented for lifecycle ops
