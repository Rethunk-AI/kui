# Spec: Libvirt Connector — Plan for spec.md

## Overview

Create `spec.md` that formally specifies the Libvirt Connector component: connection management, URI resolution, domain/network/storage operations, and test driver usage for CI. The spec is the authoritative contract for `internal/libvirtconn` (or equivalent package). Target: <800 lines or <10 tasks; greenfield only; no mocks.

**References to cite:**
- `docs/prd/decision-log.md` §§1–4 (Remote Libvirt Credentials, Test Driver, Go Libvirt Bindings; Connection pooling, Pool/path validation)
- `docs/prd/architecture.md` §2 (Libvirt Connector component boundaries)
- `docs/research/kui-libvirt-research.md` (libvirt API, qemu+ssh, test driver)

---

## 1. Exploration Summary

### Decision-log entries (Libvirt Connector)

| Topic | Source | Key content |
|-------|--------|-------------|
| Remote credentials | §1 | qemu+ssh; keyfile URI param; SSH public key only; no password storage |
| Test driver | §1 | test:///default (built-in); test:///path/to/config.xml (custom); per-process fake hypervisor; in-memory |
| Go bindings | §1 | libvirt.org/go/libvirt (CGo); libvirt.org/libvirt-go-xml for XML structs |
| Connection pooling | §2, §4 | "Defer to spec"; "simplest approach"; Libvirt Connector owns connection management |
| Pool/path validation | §2, §4 | Validate pool exists and active via libvirt; validate path/volume in pool if user-specified |
| Test strategy | §2 | Libvirt test driver; no mocks |
| Host config | §2, §4 | Per-host URI + keyfile in YAML; env override for Docker |

### Architecture boundaries

- **Libvirt Connector:** Connection management, URI resolution, domain/network/storage queries.
- **Out of scope:** VM provisioning logic (API layer owns orchestration).

### Spec consumers

- `spec-vm-lifecycle-create` §9: Connector owns connection lifecycle, domain CRUD, storage ops.
- `api-auth`: Setup wizard validates host via Connector connection attempt.
- `spec-console-realtime`: Connector for VNC/serial stream setup.

---

## 2. Spec Structure

The spec.md shall have the following sections.

### 2.1 Scope & Constraints

- **Content:** Spec scope: connection management, URI resolution, domain/network/storage operations, test driver for CI. Constraints: greenfield, <800 lines, no mocks, no stub implementations.
- **Sources:** User-provided scope; decision-log §2 (Docs scope, Test strategy).

### 2.2 Package Structure & Interface

- **Content:**
  - Package path: `internal/libvirtconn` (or `pkg/libvirtconn` if shared).
  - Connector interface: `Connector` with methods for domain, network, storage operations.
  - Factory: `NewConnector(cfg HostConfig) (Connector, error)` or `Connect(uri string, keyfile string) (Connector, error)`.
  - Host config shape: `uri` (required), `keyfile` (optional, required for qemu+ssh).
  - No provisioning logic; pure libvirt API wrapper.
- **Sources:** architecture §2; decision-log §1 (Go bindings).

### 2.3 Connection Lifecycle

- **Content:**
  - **Strategy:** Per-request connections (simplest approach per decision-log §4 "Connection pooling: Defer to spec — simplest approach"). Each API request opens connection, performs ops, closes. No connection pooling in MVP.
  - **URI resolution:**
    - Local: `qemu:///system`
    - Remote: `qemu+ssh://user@host/system?keyfile=/path/to/private/key`
  - **Connect:** `libvirt.NewConnect(uri)`; keyfile embedded in URI for remote.
  - **Close:** Caller must `defer conn.Close()`; Connector methods accept context for cancellation.
  - **Error handling:** Return wrapped errors with host/URI context; no stack traces to API layer.
- **Sources:** decision-log §1 (Remote Libvirt Credentials); kui-libvirt-research.md (URI format, NewConnect).

### 2.4 Domain Operations

- **Content:**
  - **List:** List all domains (active + inactive); return name, UUID, state.
  - **Define:** Define domain from XML; return domain UUID.
  - **Create:** Create (start) domain from definition.
  - **Destroy:** Destroy (force stop) domain.
  - **Start:** Start domain (equivalent to Create for persistent).
  - **Stop:** Graceful shutdown; Connector may accept timeout param.
  - **Pause:** Suspend domain.
  - **Resume:** Resume domain.
  - **Get state:** Return domain state (running, paused, shut off, etc.).
  - **Lookup:** Lookup by UUID string.
  - Map to libvirt.org/go/libvirt: Domain.Create, Domain.Shutdown, Domain.Destroy, Domain.Suspend, Domain.Resume, Connect.ListAllDomains, Connect.LookupDomainByUUIDString.
- **Sources:** spec-vm-lifecycle-create §4.6, §9; kui-libvirt-research.md.

### 2.5 Network Operations

- **Content:**
  - **List networks:** List all networks (virNetworkListAll / Connect.ListAllNetworks equivalent).
  - Return: name, UUID, active state.
  - Use case: VM create flow needs network name for domain XML (vm_defaults.network).
- **Sources:** User-provided scope; architecture §2 (storage/network queries).

### 2.6 Storage Operations

- **Content:**
  - **List pools:** List all storage pools; return name, UUID, state (active/inactive).
  - **List volumes:** List volumes in a pool; return name, path, capacity.
  - **Validate pool:** Pool exists and is active (virStoragePoolLookupByName, virStoragePoolGetInfo).
  - **Validate path/volume:** Path or volume name exists in pool (virStorageVolLookupByPath, virStorageVolLookupByName).
  - Use case: Pool/path validation before create/clone; disk selection in create flow.
- **Sources:** decision-log §2, §4 (Pool/path validation); spec-vm-lifecycle-create §5.

### 2.7 Test Driver for CI

- **Content:**
  - **URI:** `test:///default` for unit tests.
  - **Behavior:** In-memory fake hypervisor; no KVM; no mocks of Connector.
  - **Usage:** CI runs connector tests against test driver; tests use real libvirt API.
  - **Custom config:** `test:///path/to/config.xml` optional for custom test scenarios.
  - **Prerequisites:** libvirt-dev at build; test driver built into libvirt.
- **Sources:** decision-log §1 (Test Driver), §2 (Test strategy); kui-libvirt-research.md (Libvirt Test Driver For Ci).

### 2.8 Dependencies & Out of Scope

- **Dependencies:** libvirt.org/go/libvirt, libvirt.org/libvirt-go-xml (for XML structs where helpful).
- **Out of scope:** Connection pooling, VM provisioning logic, storage/network creation, migration.

### 2.9 References

- Links to docs/prd/decision-log.md, docs/prd/architecture.md, docs/research/kui-libvirt-research.md.

---

## 3. Implementation Instructions for Developer

1. Create `specs/active/spec-libvirt-connector/spec.md`.
2. Write each section (2.1–2.9) per the structure above. Use declarative requirements; cite decision-log entries where binding.
3. Keep total length <800 lines. Use concise bullets and tables.
4. Do not invent new requirements; derive only from decision-log, architecture, and libvirt research.
5. For connection lifecycle §2.3: explicitly state per-request (no pooling) as the MVP choice.
6. For domain operations §2.4: document Go binding method names (Connect.ListAllDomains, Domain.Create, etc.); research pkg.go.dev if exact names differ.
7. For network §2.5 and storage §2.6: document equivalent C API names (virNetworkListAll, virStoragePoolLookupByName) and Go binding mappings.
8. Format: Markdown with clear headings (##, ###). Include "References" section.

---

## 4. Verification

- [ ] spec.md exists at specs/active/spec-libvirt-connector/spec.md
- [ ] All sections present and populated
- [ ] No migration/backfill/backwards-compatibility language (greenfield)
- [ ] No mocks; test driver only
- [ ] Line count <800
- [ ] Decision-log citations accurate
- [ ] Connection strategy (per-request) explicitly stated
