# Libvirt Connector Spec

## 2.1 Scope & Constraints

- **Scope:** Specify connector behavior for libvirt interaction across hosts, including:
  - Connection management and URI resolution.
  - Domain, network, and storage query and validation operations.
  - Per-request connection lifecycle for every API operation.
  - Test-driver-based integration testing in CI.
- **Non-goals:** Provisioning orchestration, migration planning, VM orchestration sequencing, storage pool creation, and network creation. Storage volume create and copy are in scope for VM create and clone.
- **Constraints:**
  - Greenfield implementation (no migration/backfill or compatibility behavior).
  - No mocks for connector tests; use Libvirt test driver.
  - No stub implementations.
- **Binding references:**
  - Decision-log §2 ("Libvirt connection", "Storage/networks MVP", "Test strategy", "Setup wizard scope" + host config assumptions).
  - Architecture §2 Component Boundaries ("Libvirt Connector owns connection management, URI resolution, domain/network/storage queries").
  - Libvirt research: `qemu+ssh`, `qemu:///system`, `test:///default`, Go binding entrypoint.

## 2.2 Package Structure & Interface

- Connector package: `internal/libvirtconn` (canonical package path; shared package path allowed only if usage requires).
  - Source binding: Architecture §2 identifies this component boundary.
- Host configuration input:
  - `uri` (required).
  - `keyfile` (optional unless qemu+ssh URI requires key-based auth).
  - Host identity and selector keys are provided by API layer config (decision-log §2: named hosts with URI/keyfile in YAML).
- Public API shape (required):

| Item | Requirement |
|---|---|
| Factory | Provide a constructor that accepts host config and returns an initialized connector or connection context. |
| Context | Accept `context.Context` for cancellation/timeouts on all operations. |
| Domain API | Expose methods for list/lookup/create/start/stop/destroy/pause/resume/state retrieval. |
| Network API | Expose methods for network listing and read-only state lookup. |
| Storage API | Expose methods for pool/volume listing, pool/path validation, volume create, and volume copy/stream. |

- The Libvirt Connector owns only libvirt-side orchestration; domain orchestration, policy, and UI-facing workflows remain outside this package.
  - Bound to Architecture §2: Libvirt Connector out-of-scope includes provisioning logic.

## 2.3 Connection Lifecycle

- **MVP connection strategy (explicit): per-request connection, no pooling.**
  - Each request that needs libvirt creates a connection, performs work, and closes it before returning.
  - Decision binding: decision-log entries referencing “simplest approach” for connection strategy and explicit defer-to-spec language.
- URI resolution:
  - Local host defaults to `qemu:///system`.
  - Remote host uses `qemu+ssh://user@host/system?keyfile=/path/to/private/key`.
  - Remote auth is SSH public-key based; no password credential storage in connector config.
  - Decision binding: decision-log §1 Remote Libvirt Credentials; architecture deployment topology.
- Connect semantics:
  - Use `libvirt.NewConnect(uri)` for each operation.
  - Always call `defer conn.Close()` in request scope.
- Failure semantics:
  - Return wrapped errors containing host/uri context and operation intent.
  - Caller/UI receives sanitized error summary; raw error details stay in backend logs.
- No pool reuse and no session cache are permitted in this spec.
  - Decision binding: decision-log decision to defer pooling decision and choose per-request MVP path.

## 2.4 Domain Operations

- Required domain operations are read/list and lifecycle control only:
  - list all domains (active + inactive),
  - lookup by UUID,
  - define from XML,
  - create/start,
  - graceful shutdown,
  - forced destroy,
  - pause,
  - resume,
  - read state.
- API method bindings:

| Requirement | C API equivalent | Go binding |
|---|---|---|
| List all domains | `virConnectListAllDomains` | `Connect.ListAllDomains` |
| Lookup domain by UUID | `virDomainLookupByUUIDString` | `Connect.LookupDomainByUUIDString` |
| Define XML | `virDomainDefineXML` | `Connect.DomainDefineXML` |
| Create/start domain | `virDomainCreate` | `Domain.Create` |
| Graceful stop | `virDomainShutdown` | `Domain.Shutdown` |
| Force stop | `virDomainDestroy` | `Domain.Destroy` |
| Pause | `virDomainSuspend` | `Domain.Suspend` |
| Resume | `virDomainResume` | `Domain.Resume` |
| State read | `virDomainGetState` | `Domain.GetState` |

- Return model for list/get-state:
  - `name`, `uuid`, `state`, and where applicable: optional config summary (`maxMemory`, `vcpus` if required by caller).
  - State mapping is normalized to KUI lifecycle terms (`running`, `paused`, `shutoff`, etc.).
- Domain operation errors are propagated to caller with operation+host context, without leaking connector internals.
- **Domain redefine:** `Domain.DefineXML` (or `Connect.DomainDefineXML`) supports updating existing persistent domain XML. Domain must be stopped for most config changes (e.g. CPU, RAM, network). Used by VM config edit flow in `specs/active/spec-vm-lifecycle-create/spec.md` §7.
- Binding references:
  - Plan requirement for Go binding method names (`Connect.ListAllDomains`, `Domain.Create`, etc.) + decision-log lifecycle scope (MVP lifecycle includes list/create/start/stop/pause/resume/destroy) + libvirt research Go API examples.

## 2.5 Network Operations

- Required network scope:
  - List all configured networks.
  - Return `name`, `uuid`, `active`.
  - Provide read-model only; no network creation/update/delete operations.
- API method bindings:

| Requirement | C API equivalent | Go binding |
|---|---|---|
| List all networks | `virConnectListAllNetworks` | `Connect.ListAllNetworks` |
| Network name | `virNetworkGetName` | `Network.GetName` |
| Network UUID | `virNetworkGetUUIDString` | `Network.GetUUIDString` |
| Active state | `virNetworkIsActive` | `Network.IsActive` |

- Use case:
  - Connector supplies network metadata required for VM create flow defaults.
- Binding references:
  - Plan §2.5 scope; architecture §2 component boundary includes storage/network queries.

## 2.6 Storage Operations

- Required storage scope:
  - Validate pool and storage readiness before domain create/clone actions.
  - List pools and list pool volumes.
  - Validate configured pool exists and is active.
  - Validate user-specified path/volume exists in target pool.
  - Create storage volume (for pool+path VM create when auto-generating disk).
  - Copy or stream storage volume (for VM clone, same-host and cross-host).
- API method bindings:

| Requirement | C API equivalent | Go binding |
|---|---|---|
| List storage pools | `virConnectListAllStoragePools` | `Connect.ListAllStoragePools` |
| Pool name/uuid/state | `virStoragePoolGetName`, `virStoragePoolGetUUIDString`, `virStoragePoolGetInfo` | `StoragePool.GetName`, `StoragePool.GetUUIDString`, `StoragePool.GetInfo` |
| List volumes in pool | `virStoragePoolListAllVolumes` | `StoragePool.ListAllStorageVolumes` |
| Validate pool by name | `virStoragePoolLookupByName` | `Connect.LookupStoragePoolByName` |
| Validate volume by name | `virStorageVolLookupByName` | `StoragePool.LookupStorageVolByName` |
| Validate path in pool | `virStorageVolLookupByPath` | `Connect.LookupStorageVolByPath` |
| Read volume path | `virStorageVolGetPath` | `StorageVol.GetPath` |
| Read volume size | `virStorageVolGetInfo` | `StorageVol.GetInfo` |
| Create volume from XML | `virStorageVolCreateXML` | `StoragePool.CreateStorageVolFromXML` |
| Clone volume | `virStorageVolCreateXMLFrom` or external `qemu-img` | Connector exposes copy/stream primitive; implementation may use libvirt APIs or `qemu-img` for cross-host |

- Volume create: Used by pool+path VM create when disk is auto-generated. Caller supplies volume XML (name, capacity, format).
- Volume copy/stream: Used by VM clone. Source and target may be on same host or different hosts. Connector must guarantee end-to-end result is a usable cloned disk at the resolved target path. For cross-host, implementation may use SSH-backed path operations, libvirt-assisted copy/stream, or `qemu-img` conversion.
- Binding references:
  - `specs/active/spec-vm-lifecycle-create/spec.md` §2.2 (create), §3.4 (clone), §10 (connector responsibilities).
- Validation behavior:
  - If pool is missing or inactive, return explicit validation failure.
  - If a requested path or name is missing, return explicit validation failure before any write/define/create action.
- Binding references:
  - decision-log §2 "Pool/path validation".
  - Architecture §2 storage/network queries.

## 2.7 Test Driver for CI

- Mandatory test mode:
  - Integration tests for the connector must target `test:///default`.
  - Optional custom scenario mode: `test:///path/to/config.xml`.
- Behavior requirements:
  - Use built-in per-process in-memory hypervisor state.
  - No hardware/KVM dependency in these tests.
  - No connector mocks; use real libvirt-go API calls against test driver URIs.
- Preconditions:
  - libvirt development libraries available at build time.
  - Test driver available in libvirt runtime (as part of standard libvirt support).
- Binding references:
  - decision-log §1 Test Driver for CI, decision-log §2 Test strategy, libvirt research on test driver behavior.

## 2.8 Dependencies & Out of Scope

- Dependencies:
  - `libvirt.org/go/libvirt` (primary connector binding).
  - `libvirt.org/libvirt-go-xml` (XML parsing and serialization helpers where needed).
  - Standard library context logging/error handling packages as required.
- Explicitly out of scope:
  - Connection pooling (MVP path is per-request connections).
  - VM provisioning/business orchestration.
  - Storage pool creation, network creation, or storage/network management beyond what VM create and clone require.
  - Any migration/backfill behaviors.

## 2.9 References

- `docs/prd/decision-log.md`
- `docs/prd/architecture.md`
- `docs/research/kui-libvirt-research.md`
