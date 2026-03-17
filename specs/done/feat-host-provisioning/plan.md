# Host Provisioning Plan

## Overview / Goals

Add the ability to provision KVM hosts (create storage pools and networks) when they lack them. Users get clear feedback during setup and post-setup, with an audit-first flow: show what would be created, user approves, then create.

**Goals:**
1. Allow one-click provisioning of default storage pool and NAT network when a host has none.
2. Audit-first: API returns proposed actions (pool path, network config); frontend shows; user confirms; backend executes.
3. Inline UX: when validate-host returns "no pools" or "no networks", show audit + provision in the same panel.
4. Post-setup: provision action available when host lacks pools/networks (host selector or VM list context).
5. Partial failure: report what succeeded and what failed; allow retry.
6. MVP: local hosts only (`qemu:///system`); reject remote with 400.

---

## Design Decisions

### Pool Path Logic

| Condition | Action |
|-----------|--------|
| `/var/lib/libvirt/images` exists and has content (non-empty dir) | Define dir pool there; use as default path |
| `/var/lib/libvirt/images` missing or empty | Propose `/var/lib/kui/images`; create dir via `os.MkdirAll` (0755) before pool define |
| Both paths unusable | Fail with clear error |

**Rationale:** Reuse libvirt default when present; otherwise use KUI-owned path. Directory creation is required for dir-type pools; libvirt `pool-build` may create target but behavior varies—explicit `os.MkdirAll` ensures consistency.

### Audit Flow: Option B (Single POST with dry_run)

| Option | Description | Chosen |
|--------|-------------|--------|
| **A** | Two-phase: `GET /audit` returns proposal; `POST` executes | No |
| **B** | Single `POST` with `dry_run: true` returns audit; `dry_run: false` executes | **Yes** |

**Rationale:** Simpler API surface; one endpoint; frontend calls same URL for audit and execute.

### Local-Only (MVP)

- **Local:** `qemu:///system` or equivalent (no `qemu+ssh://`).
- **Remote:** `qemu+ssh://` or any non-local URI → `400` with `"remote host provisioning not supported in this version"`.

### Fill-Gap Logic

**Decision:** Support partial provisioning in MVP. If host has pools but no networks, provision only network (and vice versa). Audit returns only what is missing; execute creates only those.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Setup Wizard                                                                │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │  Validate host  →  no pools / no networks  →  Show audit panel          │ │
│  │       ↓                    ↓                        ↓                  │ │
│  │  validateHost()      res.error matches         provisionHost(dry_run)   │ │
│  │                       "no storage pools"       → show pool/network;      │ │
│  │                       or "no networks"        → "Provision" button      │ │
│  │                                                    ↓                    │ │
│  │                                            provisionHost(dry_run=false)  │ │
│  │                                                    ↓                    │ │
│  │                                            validateHost() (re-validate) │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  Post-setup (host selector / VM list)                                       │
│  Host lacks pools/networks  →  Provision button  →  Same audit + execute    │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  Backend                                                                    │
│  POST /api/setup/provision-host  (setup, unauthenticated)                  │
│  POST /api/hosts/:id/provision   (authenticated)                            │
│       ↓                                                                     │
│  Check URI local?  →  no: 400   yes: continue                                │
│       ↓                                                                     │
│  dry_run=true  →  compute audit (pool path, network config); return 200      │
│  dry_run=false →  mkdir if needed; CreateStoragePoolFromXML; CreateNetwork  │
│                   →  return 200 with created/failed per resource            │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Tasks

### Task 1: Libvirt Connector — CreateStoragePoolFromXML, CreateNetworkFromXML

**File:** `internal/libvirtconn/connector.go` (and `connector_stub.go` for build tag `!libvirt`)

**Interface additions:**

```go
CreateStoragePoolFromXML(ctx context.Context, xml string) (StoragePoolInfo, error)
CreateNetworkFromXML(ctx context.Context, xml string) (NetworkInfo, error)
```

**Implementation:**

- **Storage pool:** Use `conn.StoragePoolCreateXML(xml, 0)` — creates, builds, and starts in one call. For dir-type pools, ensure target directory exists before calling (caller responsibility or internal helper). Return `StoragePoolInfo` from the created pool.
- **Network:** Use `conn.NetworkCreateXML(xml, 0)` — creates and starts. Return `NetworkInfo`.
- Use `libvirt.org/go/libvirtxml` for `StoragePool` and `Network` structs to build XML; marshal to string.

**Stub:** Add methods to `connector_stub.go` returning `ErrLibvirtDisabled`.

**Acceptance criteria:**
- [ ] Connector interface has both methods.
- [ ] Real connector creates dir pool and NAT network from XML.
- [ ] Stub returns ErrLibvirtDisabled.

---

### Task 2: Pool Logic — Path Selection and Directory Creation

**File:** New `internal/provision/provision.go` (or inline in routes handler)

**Logic:**

1. **Pool path selection:**
   - If `/var/lib/libvirt/images` exists and is non-empty (e.g. `os.ReadDir` returns >0 entries), use it.
   - Else use `/var/lib/kui/images`.
2. **Directory creation:** For `/var/lib/kui/images`, call `os.MkdirAll(path, 0o755)` before pool define. Fail with clear error on permission denied.
3. **Pool XML:** Build dir-type pool XML via `libvirtxml.StoragePool`; `Type="dir"`, `Target.Path` = chosen path, `Name` = `"default"`.
4. **Network XML:** Build NAT network via `libvirtxml.Network`; default `192.168.122.0/24`, name `"default"`, `Forward.Mode="nat"`.

**Acceptance criteria:**
- [ ] Path selection follows rules above.
- [ ] Directory created with 0755 when using `/var/lib/kui/images`.
- [ ] Pool and network XML valid for libvirt.

---

### Task 3: API — POST /api/setup/provision-host

**File:** `internal/routes/routes.go`

**Route:** `router.Post("/api/setup/provision-host", state.provisionHostSetup())`

**Request:**

```json
{
  "host_id": "local",
  "uri": "qemu:///system",
  "keyfile": "",
  "dry_run": true
}
```

**Response (dry_run=true, 200):**

```json
{
  "audit": {
    "pool": { "path": "/var/lib/kui/images", "type": "dir", "name": "default" },
    "network": { "name": "default", "subnet": "192.168.122.0/24", "type": "nat" }
  },
  "local_only": true
}
```

If host already has pools and networks, return `audit: null` or empty; no provisioning needed.

**Response (dry_run=false, 200):**

```json
{
  "pool": { "created": true, "name": "default" },
  "network": { "created": true, "name": "default" }
}
```

**Partial failure:**

```json
{
  "pool": { "created": true, "name": "default" },
  "network": { "created": false, "error": "network 'default' already exists" }
}
```

**Error responses:**
- `400`: Non-local URI → `"remote host provisioning not supported in this version"`.
- `400`: Invalid request body.
- `400`: URI required.
- `500`: Connection or libvirt error (sanitized).

**Auth:** Same as `validate-host` — available only during setup (no config + no admin). Block when setup complete.

**Acceptance criteria:**
- [ ] dry_run=true returns audit without creating.
- [ ] dry_run=false creates pool and network (or only missing ones).
- [ ] Non-local URI returns 400.
- [ ] Partial failure returns per-resource status.

---

### Task 4: API — POST /api/hosts/:id/provision

**File:** `internal/routes/routes.go`

**Route:** `router.Post("/api/hosts/{host_id}/provision", state.provisionHost())`

**Auth:** JWT required. Host must exist in config.

**Request:**

```json
{
  "dry_run": true,
  "pool_path": "/var/lib/kui/images",
  "network_name": "default",
  "network_subnet": "192.168.122.0/24"
}
```

All body fields optional; defaults as in Task 2. `dry_run` defaults to `false` if omitted.

**Response:** Same shape as Task 3 (audit vs created/failed).

**Error responses:**
- `400`: Host not found in config.
- `400`: Non-local URI for host.
- `401`: Unauthenticated.
- `500`: Connection or libvirt error.

**Acceptance criteria:**
- [ ] Authenticated users can provision configured hosts.
- [ ] Host ID from path; URI from config.
- [ ] Rejects remote hosts with 400.

---

### Task 5: Frontend — SetupWizard Audit + Provision

**File:** `web/src/components/SetupWizard.ts`

**Logic:**

1. When `validateHost` returns `valid: false` and `res.error` contains `"no storage pools"` or `"no networks"` (or both):
   - Show audit panel below the validate result.
   - Call `provisionHost({ host_id, uri, keyfile, dry_run: true })` to fetch audit.
   - Display proposed pool path and network name/subnet.
   - Add "Provision" button.
2. On "Provision" click:
   - Call `provisionHost({ host_id, uri, keyfile, dry_run: false })`.
   - On success: show success message; re-run `validateHost`; if valid, update validate result to "Valid".
   - On partial failure: show what succeeded and what failed; allow retry.
   - On error: show error message.
3. If validation returns `valid: true`, do not show audit panel.

**API client:** Add `provisionHost` in `web/src/lib/api.ts`:

```ts
export interface ProvisionHostRequest {
  host_id: string;
  uri: string;
  keyfile: string;
  dry_run: boolean;
}

export interface ProvisionHostAuditResponse {
  audit: {
    pool: { path: string; type: string; name: string };
    network: { name: string; subnet: string; type: string };
  } | null;
  local_only: boolean;
}

export interface ProvisionHostResultResponse {
  pool: { created: boolean; name?: string; error?: string };
  network: { created: boolean; name?: string; error?: string };
}
```

**Acceptance criteria:**
- [ ] Audit panel appears when validation fails for pools/networks.
- [ ] Audit shows pool path and network config.
- [ ] Provision button executes and re-validates on success.
- [ ] Partial failure displayed; retry possible.

---

### Task 6: Frontend — Provision Action in Host Management

**Placement:** Add provision action where host lacks pools/networks. Options:
- **A:** Host selector dropdown — when selected host has no pools/networks, show "Provision host" link/button next to selector.
- **B:** CreateVMModal — when pool or network list is empty for selected host, show "Provision host" before pool/network selects.
- **C:** Dedicated hosts page or host detail — if such UI exists.

**Decision:** Use **Option B** (CreateVMModal) as primary: when user opens "Create VM" and selected host has no pools or no networks, show inline "Provision host" CTA. Option A can be added as secondary (e.g. tooltip or link in host selector when pools/networks empty).

**Implementation:**
- In `CreateVMModal`, when `loadPoolsAndNetworks(hostId)` returns empty pools or networks, render "Provision host" section with audit + provision flow (same as SetupWizard).
- Reuse `provisionHost` API; for post-setup use `POST /api/hosts/:id/provision` with JWT (different endpoint, same response shape).
- Add `provisionHostPostSetup(hostId, options?)` in api.ts for authenticated provision.

**Acceptance criteria:**
- [ ] CreateVMModal shows provision CTA when host has no pools or no networks.
- [ ] Provision flow works with authenticated endpoint.
- [ ] On success, pools/networks reload; user can proceed to create VM.

---

### Task 7: Tests

**File:** `internal/routes/routes_test.go`

**Unit / handler tests:**

1. **TestProvisionHostSetup_RejectsRemoteURI** — `qemu+ssh://...` → 400.
2. **TestProvisionHostSetup_DryRunReturnsAudit** — mock connector with empty pools/networks; dry_run=true → 200 with audit.
3. **TestProvisionHostSetup_ExecuteCreatesPoolAndNetwork** — mock with `CreateStoragePoolFromXML` and `CreateNetworkFromXML`; dry_run=false → 200 with created.
4. **TestProvisionHostSetup_PartialFailure** — mock pool success, network failure → 200 with pool created, network error.
5. **TestProvisionHostSetup_BlockedWhenSetupComplete** — config + admin present → 403 or equivalent.
6. **TestProvisionHostPostSetup_RequiresAuth** — no JWT → 401.
7. **TestProvisionHostPostSetup_RejectsRemoteHost** — host with qemu+ssh URI → 400.

**Mock connector:** Extend `mockConnector` in routes_test.go with `CreateStoragePoolFromXML` and `CreateNetworkFromXML`; track calls for assertions.

**File:** `internal/libvirtconn/connector_test.go` (if exists) or new `internal/provision/provision_test.go` for pool path logic.

**Acceptance criteria:**
- [ ] All handler tests pass.
- [ ] `go test ./internal/routes/...` passes.
- [ ] `go build ./cmd/...` succeeds.

---

### Task 8: Docs — Admin Guide Provisioning Flow

**File:** `docs/admin-guide.md` or `docs/provisioning.md` (create if needed)

**Content:**
- When provisioning is needed (validate-host fails, Create VM shows no pools).
- Default paths: `/var/lib/libvirt/images` vs `/var/lib/kui/images`.
- Permissions: KUI process must be able to create `/var/lib/kui` and subdirs (typically root or kui user in systemd).
- Local-only in MVP; remote in v2.
- Audit-first flow: review before execute.

**Acceptance criteria:**
- [ ] Admin can follow doc to understand and troubleshoot provisioning.

---

## Request/Response Shapes (Reference)

### POST /api/setup/provision-host

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| host_id | string | yes | Host identifier |
| uri | string | yes | Libvirt URI (must be local) |
| keyfile | string | no | SSH key path (empty for local) |
| dry_run | bool | yes | true = audit only; false = execute |

### POST /api/hosts/:id/provision

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| dry_run | bool | no | Default false |
| pool_path | string | no | Override default pool path |
| network_name | string | no | Override network name |
| network_subnet | string | no | Override subnet (e.g. 192.168.122.0/24) |

---

## Error Handling

| Scenario | HTTP | Body |
|----------|------|------|
| Non-local URI | 400 | `{ "error": "remote host provisioning not supported in this version" }` |
| Invalid JSON | 400 | `{ "error": "invalid request body" }` |
| URI required | 400 | `{ "error": "uri required" }` |
| Connection failed | 500 | `{ "error": "..." }` (sanitized) |
| Pool create failed | 200 | `{ "pool": { "created": false, "error": "..." }, "network": {...} }` |
| Setup complete, provision-host | 403 | `{ "error": "provision-host is only available during setup" }` |
| Unauthenticated, hosts/:id/provision | 401 | `{ "error": "unauthorized" }` |

---

## Acceptance Criteria Checklist

- [ ] Local hosts can be provisioned with default pool and network.
- [ ] Remote hosts return 400 with clear message.
- [ ] Audit-first: dry_run returns proposed config; user approves before execute.
- [ ] SetupWizard shows audit + provision when validate fails for pools/networks.
- [ ] CreateVMModal (or host management) shows provision when host lacks pools/networks.
- [ ] Partial failure reported; retry possible.
- [ ] All tests pass; `make all` succeeds.

---

## Out of Scope (v2)

- Remote host provisioning (qemu+ssh).
- Full storage/network management UI (edit, delete).
- Migration or backfill.
- User override of pool path/network in SetupWizard audit (can be added later).

---

## Decision Log

| Decision | Alternatives | Why Chosen | Risks / Mitigations |
|----------|--------------|------------|---------------------|
| Audit flow Option B | Option A (two-phase GET+POST) | Simpler API; one endpoint | None |
| Local-only MVP | Support remote | Reduces scope; SSH/mkdir complexity deferred | Document; v2 for remote |
| Fill-gap (partial provision) | Only when both missing | More useful; user can fix one at a time | Slightly more logic |
| Pool path: libvirt/images vs kui/images | Always kui/images | Reuse libvirt default when present | None |
| Provision in CreateVMModal | Host selector only | User hits empty state when creating VM; natural CTA | Could add both |

---

## Ownership Boundaries

**In scope:**
- `internal/libvirtconn/connector.go`, `connector_stub.go`
- `internal/routes/routes.go`, `routes_test.go`
- `internal/provision/` (new, if extracted)
- `web/src/components/SetupWizard.ts`, `CreateVMModal.ts`
- `web/src/lib/api.ts`
- `docs/` (provisioning flow)

**Out of scope:**
- `internal/eventsource/` — no changes
- `internal/config/` — no schema changes
- Migration/backfill — none

---

## Assumptions

- `libvirt.org/go/libvirt` provides `StoragePoolCreateXML` and `NetworkCreateXML` (or equivalent).
- `libvirt.org/go/libvirtxml` has `StoragePool` and `Network` structs for XML building.
- KUI process has permission to create `/var/lib/kui` (or runs as root).
- Test mocks can simulate CreateStoragePoolFromXML/CreateNetworkFromXML for handler tests.
- Validate-host error strings remain `"Host X has no storage pools"` and `"Host X has no networks"` for frontend matching.
