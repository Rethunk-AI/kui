# VM Lifecycle & Create Specification

## 1. Scope & Constraints

This specification defines the MVP VM lifecycle and provisioning contract for KUI.

- In scope:
  - VM creation from pool+path with `vm_defaults`.
  - VM clone for same-host and cross-host flows.
  - Lifecycle operations: list, start, stop, pause, resume, destroy.
  - Discovery endpoints (pools, volumes, networks) for VM create form.
  - Pool/path validation via libvirt.
  - Orphan domain discovery and claim flow.
  - Informal API surface for these features.
- Out-of-scope for this spec is migration, compatibility modes, backward-compatibility defaults, and template-based creation.
- Source constraints:
  - `docs/prd/decision-log.md` §§2,4 (VM create flow, clone scope, lifecycle, orphans, API style, greenfield).
  - `docs/prd/architecture.md` (§§1–3 for API/connector/data flow boundaries).
  - `docs/research/kui-libvirt-research.md` (libvirt API stack, URI formats, test driver).
- Greenfield-only: no migration, no backfill, no dual-path logic.
- Document target target is <800 lines.

## 2. Pool+Path Create Flow

## 2.1 Inputs and defaults

- `hosts` is required in config.
- Required config defaults for create:
  - `vm_defaults.cpu`, `vm_defaults.ram_mb`, `vm_defaults.network` are required baseline values.
  - `default_host` is read from config and can be overridden in user preferences.
  - `default_pool` may be configured and can be overridden at request time.
- User create inputs:
  - `host_id` (required; default from config).
  - `pool` (required unless user supplies an existing full path that implies pool).
  - `disk` mode:
    - existing volume selector, or
    - generated path + size.
  - Optional overrides:
    - `cpu`
    - `ram_mb`
    - `network`
    - `disk.path` and `disk.name` (explicit path override).
- Disk option behavior:
  - Prefer existing pool volume if provided.
  - If not provided, generate a pool-resident path with size.
  - User-specified path/name override is allowed.

## 2.2 Flow

1. Resolve host and validate config inputs.
2. Validate pool and path/volume constraints against libvirt (section 5).
3. Resolve final VM config:
   - start from `vm_defaults`.
   - apply request overrides.
4. Render or load domain XML (using libvirt-go-xml helpers where useful).
5. Create storage object when auto-generated disk was selected.
6. Define domain in libvirt on target host.
7. Insert `vm_metadata` row with:
   - `claimed = true`
   - `display_name` from request or resolved domain name
   - `host_id` from target host
   - `libvirt_uuid` from `Domain.GetUUIDString`
   - `last_access = now`.
8. Return API response with `host_id`, `libvirt_uuid`, final effective config.

## 2.3 Auto-start after create

- Auto-start after create is **out of scope for this MVP spec**.
- Create defines and tracks VM metadata only.
- VM start remains explicit via lifecycle `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/start`.

## 3. Clone Flow

## 3.1 Inputs

- Source identity:
  - `source.host_id`
  - `source.libvirt_uuid`
- Destination:
  - `target_host_id` (required).
  - `target_pool` (required).
  - `target_name` (optional).
- Clone behavior is governed by:
  - same format requirement as source disk format.
  - source-name template rules.

## 3.2 Name template rule

- Template support in MVP:
  - `{source}` only.
- v2-only templates:
  - `{date}`
  - `{timestamp}`
- Resolution precedence:
  - request override > user preference > config default.
- If no custom name is provided, use resolved `{source}` template.

## 3.3 Source VM readiness and state requirement

- Source VM must be in stopped state for clone.
- If running, API returns a user-facing error and rejects the request.
- Reason: prevents inconsistent disk copy semantics in MVP and aligns with stable clone behavior expectations.

## 3.4 Flow

1. Validate source VM exists in source host domain inventory.
2. Validate source state is stopped and target pool exists/active (section 5).
3. Resolve final clone name.
4. Copy/stream source disk to target host and target pool:
   - same disk format as source.
   - cross-host path uses copy/stream transport; implementation may choose transport detail.
   - Connector must guarantee end-to-end result is a usable cloned disk at the resolved target path.
5. Build clone domain XML:
   - new domain name from clone naming step.
   - new disk path in target pool.
   - same format as source.
6. Define domain on target host.
7. Insert `vm_metadata` row for cloned VM with `claimed = true`.

## 3.5 Cross-host implementation contract (non-over-specified)

- For cross-host clones:
  - source host provides exact disk source metadata (source path + format + capacity expectations).
  - target host receives generated/validated destination disk location in pool.
- Allowed implementation approaches are allowed as long as the same result is achieved:
  - host-to-host transfer using SSH-backed path operations.
  - libvirt-assisted copy or stream mechanisms.
  - `qemu-img` conversion/stream path for format-preserving copy.
- The spec intentionally avoids binding to a single transport utility beyond requirement conformance.

## 3.6 Cross-host requirement

- Cross-host cloning is supported.
- Clone from source host and target host may differ.
- All cross-host clones must maintain source format unless pool-target constraints force an explicit format conversion step documented in code-level notes.

## 4. Lifecycle Operations

## 4.1 List

- API list must be host-aware and include:
  - all domains from all configured hosts that are reachable.
  - merged per-VM KUI metadata from local SQLite (`vm_metadata`) for display fields and claimed state.
- Unreachable hosts are represented distinctly with host/offline state.

## 4.2 Start

- Operation:
  - look up domain by `host_id` + `libvirt_uuid`.
  - call libvirt domain start API.
- Binding:
  - use `Domain.Create()` (or API-equivalent libvirt domain create operation).

## 4.3 Stop (graceful then force)

- Graceful stop path:
  - call libvirt graceful shutdown API.
  - default timeout: `30s`.
- Config key:
  - `vm_lifecycle.graceful_stop_timeout`
  - config file path: `/etc/kui/config.yaml` (default).
- Timeout behavior:
  - if domain not stopped within timeout, execute force stop path.
- Force stop:
  - call libvirt domain destroy API.
- This exactly matches: graceful first, force if needed.

## 4.4 Pause / Resume

- Pause:
  - call libvirt suspend API.
- Resume:
  - call libvirt resume API.

## 4.5 Destroy

- Destroy executes immediate hard stop and resource removal of runtime domain definition:
  - call libvirt destroy API.
- Destroy is not a soft or timed operation.

## 4.6 Libvirt API choice

- Use Go bindings methods from `libvirt.org/go/libvirt` directly:
  - `Domain.Create`
  - `Domain.Shutdown` (or equivalent graceful shutdown API)
  - `Domain.Destroy`
  - `Domain.Suspend`
  - `Domain.Resume`
  - `LookupByUUIDString` plus domain list operations.
- These method names are implementation-level bindings of the libvirt C API family.

## 4.7 Transient vs persistent handling

- If domain is transient:
  - lifecycle and state calls still target active domain object and return domain state immediately after each action.
- If domain is persistent:
  - operations execute against definition-backed domain object.
- APIs must return user-facing errors for already-running/already-shut-off operations.

## 4.8 VM Detail

- A dedicated detail endpoint is required for:
  - single-VM fetch with fresh state and metadata (used by detail view and before console open).
  - bumping `last_access` when the user opens the detail view or opens the console.
- List (`GET /api/vms`) returns full per-VM data for list display but does **not** bump `last_access`; list refresh must not update access time.
- Endpoint: `GET /api/hosts/{host_id}/vms/{libvirt_uuid}`.
- Returns: VM info (host_id, libvirt_uuid, display_name, claimed), current domain state from libvirt, and vm_metadata fields (console_preference, last_access, created_at, updated_at).
- On successful response: update `vm_metadata.last_access = now` for the VM.
- Used by: detail view page load; call before opening console (or console-open flow may bump separately; detail endpoint covers both semantics).

## 5. Pool/Path Validation

- A validation step must run before create and clone operations.
- Validation steps:
  - Resolve host via config `host_id`.
  - Validate pool exists through libvirt query.
  - Validate pool is active.
  - If explicit path/name is supplied:
    - resolve it as volume/path inside the selected pool.
    - return a validation error if missing.
- If pool not active/ineligible, reject operation with user-friendly reason.
- For auto path mode:
  - target path must not collide with existing names in that pool.

## 6. Orphan Claim Flow

## 6.1 Discovery

- On demand or periodic reconciliation:
  - list all domains per host through connector.
  - compare (`host_id`, `libvirt_uuid`) against `vm_metadata`.
- Orphans include:
  - domains not present in `vm_metadata`, or
  - rows where `claimed = false`.

## 6.2 UI surface

- Orphan VMs must be shown in separate UI section/tab from claimed VMs.
- UI provides one action: Claim.

## 6.3 Claim operation

- Claim requires selection of host-discovered orphan.
- Claim inserts/updates row:
  - `claimed = true`
  - `host_id` from discovery host
  - `display_name` from domain name unless user-provided input
  - `last_access = now`.
- Claim does not modify libvirt domain XML or storage.

## 7. VM Config Edit

### 7.1 Endpoint

`PATCH /api/hosts/{host_id}/vms/{libvirt_uuid}`

### 7.2 Request body (partial, all optional)

- `display_name` (string | null) — KUI display name
- `console_preference` (string | null) — `novnc` | `xterm` | `spice` (v2)
- `cpu` (int) — vCPU count
- `ram_mb` (int) — memory in MB
- `network` (string) — network name (must exist on host)

### 7.3 Flow

1. Validate VM exists and is claimed.
2. For domain edits (cpu, ram_mb, network): domain must be **stopped** (libvirt requires it for most config changes).
3. Fetch current domain XML via connector.
4. Apply changes: update vm_metadata for display_name/console_preference; merge domain XML for cpu/ram_mb/network.
5. If domain XML changed: `Domain.DefineXML` (redefine).
6. Emit `vm_config_change` audit event (SQLite + Git diff per `specs/active/spec-audit-integration/spec.md` §5.4).

### 7.4 Response

Same shape as detail endpoint (refreshed VM state + metadata).

### 7.5 Errors

- 400: Domain running and domain edit (cpu/ram_mb/network) requested
- 404: VM not found
- 409: Network invalid or does not exist on host

## 8. Data Model (`vm_metadata`)

- Composite key:
  - `host_id` + `libvirt_uuid`.
- Canonical fields:
  - `claimed` (integer/bool, non-null, default false).
  - `display_name` (nullable text).
  - `console_preference` (nullable text).
  - `last_access` (nullable timestamp string).
- Required operational fields:
  - `host_id` and `libvirt_uuid` required for all operations.
  - `created_at`, `updated_at` required for tracking and deterministic UI sort behavior.
- Behavioral note:
  - this table is authoritative for KUI-only metadata.
  - libvirt remains authoritative for runtime domain state and storage inventory.

## 9. Informal API Surface

### 9.1 API style

- REST/JSON is the implementation contract; API is informal at this stage.
- Decisions do not require formal OpenAPI in MVP.
- All VM endpoints are under `/api` prefix.

### 9.2 VM identity in paths

- VM identity is always `(host_id, libvirt_uuid)`.
- Paths use explicit segments: `/api/hosts/{host_id}/vms/{libvirt_uuid}`.
- Lifecycle, clone, claim, and config edit endpoints use this pattern for consistency with the detail endpoint.

### 9.3 Endpoints

#### VM operations

- `GET /api/vms` — list VMs (flat list + orphans).
- `GET /api/hosts/{host_id}/vms/{libvirt_uuid}` — get VM detail (state + metadata; bumps last_access).
- `POST /api/vms` — create VM from pool+path.
- `PATCH /api/hosts/{host_id}/vms/{libvirt_uuid}` — update VM config (metadata + domain XML).
- `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/clone` — clone VM to target host/pool. Request body: `target_host_id`, `target_pool`, optional `target_name`.
- `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/start` — start VM.
- `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/stop` — stop VM (graceful then force).
- `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/pause` — pause VM.
- `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/resume` — resume VM.
- `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/destroy` — destroy VM.
- `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/claim` — claim orphan VM.

#### Discovery (for VM create form)

- `GET /api/hosts/{host_id}/pools` — list storage pools on host. Response: `[{ "name", "uuid", "state" }]`. Used to populate pool dropdown.
- `GET /api/hosts/{host_id}/pools/{pool_name}/volumes` — list volumes in pool. Response: `[{ "name", "path", "capacity" }]`. Used for "pick existing volume" option.
- `GET /api/hosts/{host_id}/networks` — list networks on host. Response: `[{ "name", "uuid", "active" }]`. Used for network override dropdown.

### 9.4 Request/response shape

#### List response (`GET /api/vms`)

```json
{
  "vms": [
    {
      "host_id": "local",
      "libvirt_uuid": "...",
      "display_name": "...",
      "claimed": true,
      "status": "running",
      "console_preference": null,
      "last_access": "...",
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "hosts": {
    "local": "online",
    "remote1": "offline"
  },
  "orphans": [
    { "host_id": "local", "libvirt_uuid": "...", "name": "..." }
  ]
}
```

- `vms`: flat list of claimed VMs with host_id, libvirt_uuid, display_name, claimed, status, metadata.
- `hosts`: map of host_id → `online` | `offline`.
- `orphans`: list of unclaimed domains (host_id, libvirt_uuid, name from libvirt).

#### Other shapes

- Create request includes:
  - `host_id`, `pool`, disk selection/path info, optional overrides for cpu/ram/network, optional disk values.
- Clone request (path provides source; body provides target) includes:
  - `target_host_id`, `target_pool`, optional `target_name`.
- Create/clone response includes:
  - `host_id`, `libvirt_uuid`, `display_name`, `created_at`, `status`.
- Detail response (`GET /api/hosts/{host_id}/vms/{libvirt_uuid}`) includes:
  - `host_id`, `libvirt_uuid`, `display_name`, `claimed`, `status` (domain state), `console_preference`, `last_access`, `created_at`, `updated_at`.
- Lifecycle responses are action-result centered and include refreshed VM state.

## 10. Libvirt Connector Responsibilities

- Owns connection handling per request (per-request connection, no pooling; see `specs/done/spec-libvirt-connector/spec.md` §2.3), with config-sourced URI and keyfile.
- Remote URIs use `qemu+ssh://user@host/system?keyfile=...`.
- Responsibilities include:
  - connection lifecycle and host-scoped error reporting.
  - domain CRUD and power/state operations (list/create/start/stop/pause/resume/destroy/define lookup).
- Domain redefine: config edit flow uses `Domain.DefineXML` to update existing persistent domain XML; domain must be stopped for most config changes (see §7 VM Config Edit).
  - storage operations (pool lookup, storage volume lookup/create, copy/stream flow primitives).
  - reporting host reachability for orchestration/state list output.
- Connector MUST NOT encode KUI provisioning rules (no orchestration policies in this layer).
- Test strategy:
  - use `test:///default` for libvirt test-driver-based CI.
  - no manual mock implementation in unit tests for connector semantics.
- Dependency stack:
  - `libvirt.org/go/libvirt`
  - `libvirt.org/libvirt-go-xml` where helpful for XML composition.

## 11. Out of Scope

- Template-based create flow (`Create VM from template`) is deferred.
- Docker deployment details.
- Storage/network provisioning/management (existing storage/networks only).
- Clone progress/events (`%, ETA, stage`) in v1.
- Migration of existing domains or data migration tools.
- Source-template name macros beyond `{source}`.
- HTTP/template source import flows for clone input.
- Formal OpenAPI generation for this spec.
- Connection pooling strategy decisions beyond connector-owned internal handling.

## References

- `docs/prd/decision-log.md`
- `docs/prd/architecture.md`
- `docs/research/kui-libvirt-research.md`
