# feat-stuck-vm — Plan

## Overview

Implement stuck VM detection and recovery with escalating actions: force stop (Destroy) → force destroy (Destroy, idempotent) → undefine (remove domain definition). Libvirt state only; no migration paths.

**References:** `docs/prd/backlog.md` (v2 item 3), `docs/prd/decision-log.md` A19–A20.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Web UI                                                                    │
│  VM list: "Stuck" badge + "Recover" button when state in {crashed,blocked}│
│  or when lifecycle op fails                                               │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ API                                                                      │
│  POST /api/hosts/{host_id}/vms/{libvirt_uuid}/recover                     │
│  Escalation: Destroy → Undefine (if Destroy succeeded or domain gone)     │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ Libvirt Connector                                                        │
│  Destroy(ctx, uuid)  — existing                                          │
│  Undefine(ctx, uuid) — NEW                                               │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Scope

### In scope

- **Stuck detection:** Domain state `crashed` or `blocked`; or user-initiated recovery when lifecycle ops fail
- **Escalation:** 1) Destroy (force stop) 2) Undefine (remove definition). After Destroy, domain is shut off; Undefine removes persistent definition. If Destroy fails, attempt Undefine with force flag if supported.
- **API:** `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/recover`
- **UI:** Show "Stuck" or "Recover" for VMs in crashed/blocked; Recover button triggers escalation

### Out of scope

- Automatic detection/background recovery (user-initiated only)
- Migration, backfill, backwards compatibility

---

## Stuck Detection

| Domain State | Considered Stuck | Reason |
|--------------|------------------|--------|
| `crashed` | Yes | Guest crashed; may need recovery |
| `blocked` | Yes | Domain blocked on resource |
| `running` | No | Normal |
| `shutoff` | No | Normal stopped |
| `paused` | No | Normal paused |
| `unknown` | Maybe | Treat as recoverable if user requests |

**Decision:** Expose Recover for `crashed` and `blocked`. Optionally: show Recover when last Stop/Destroy failed (defer to implementation; keep plan simple).

---

## Escalation Logic

1. **Destroy** — `conn.Destroy(ctx, uuid)`. Force-stops the domain. Idempotent if already shut off (libvirt may return error or no-op; handle gracefully).
2. **Undefine** — `conn.Undefine(ctx, uuid)`. Removes persistent domain definition. Domain must be stopped. If Destroy failed (e.g. zombie), attempt Undefine with `VIR_DOMAIN_UNDEFINE_FORCE` or equivalent if libvirt allows.

**Flow:**
```
1. Get state
2. If running/crashed/blocked: Destroy
3. If Destroy succeeded or domain already shut off: Undefine
4. If Undefine succeeds: remove from vm_metadata (or mark as undefined; VM no longer in libvirt)
3. If Destroy failed: try Undefine with force flag if available
```

**Post-recovery:** After Undefine, the domain no longer exists in libvirt. KUI should either:
- Remove vm_metadata row (VM is gone), or
- Mark as "undefined" and show in UI as "deleted" — user can remove from list

**Decision:** After successful Undefine, delete vm_metadata row. VM is gone; no need to keep orphan record. Audit log records the recovery action.

---

## Components

| Component | Responsibility |
|-----------|----------------|
| `internal/libvirtconn/connector.go` | Add `Undefine(ctx, uuid) error` |
| `internal/libvirtconn/connector_stub.go` | Add `Undefine` stub (return ErrLibvirtDisabled) |
| `internal/routes/routes.go` | Add `vmRecover()` handler; POST recover route |
| `web/src/lib/api.ts` | Add `recoverVM(hostId, libvirtUuid)` |
| `web/src/components/VMList.ts` | Show "Stuck" badge + Recover button for crashed/blocked |

---

## Implementation Tasks

### 1. Connector: Undefine

**File:** `internal/libvirtconn/connector.go`

- Add `Undefine(ctx context.Context, uuid string) error` to Connector interface
- Implement: lookup domain by UUID, call `domain.Undefine()` or `domain.UndefineFlags()` with appropriate flags
- Use `libvirt.DOMAIN_UNDEFINE_NVRAM` or similar if needed for cleanup
- If domain is running, libvirt may reject; try `UndefineFlags` with force flag if available (research libvirt-go API)

**File:** `internal/libvirtconn/connector_stub.go`

- Add `Undefine` to interface; stub returns `ErrLibvirtDisabled`

### 2. API: Recover endpoint

**File:** `internal/routes/routes.go`

- Add route: `router.Post("/api/hosts/{host_id}/vms/{libvirt_uuid}/recover", state.vmRecover())`
- Handler: auth, get VM metadata, get connector, run escalation:
  1. Destroy (ignore error if already shut off)
  2. Undefine
  3. On success: delete vm_metadata row
  4. Audit: `vm_lifecycle` with action `recover`
- Return 200 with `{ "status": "undefined" }` or 500 on failure

### 3. DB: Delete vm_metadata

**File:** `internal/db/` (or schema)

- Ensure `DeleteVMMetadata(host_id, libvirt_uuid)` exists or add it
- Use in recover handler after successful Undefine

### 4. Frontend: API client

**File:** `web/src/lib/api.ts`

- Add `recoverVM(hostId: string, libvirtUuid: string): Promise<void>`
- POST to `/api/hosts/{host_id}/vms/{libvirt_uuid}/recover`

### 5. Frontend: VM list UI

**File:** `web/src/components/VMList.ts`

- For `vm.status === "crashed"` or `vm.status === "blocked"`: show "Stuck" badge and "Recover" button
- Recover button: call `recoverVM`, on success refresh list
- Handle errors: toast on failure

---

## Data Model

- No schema change. `vm_metadata` delete on successful Undefine.
- Audit: new event payload `action: "recover"` with `from_state`, `to_state` (undefined).

---

## API Contract

### POST /api/hosts/{host_id}/vms/{libvirt_uuid}/recover

**Request:** No body.

**Response 200:**
```json
{ "status": "undefined" }
```

**Errors:**
- 401 Unauthorized
- 404 VM not found
- 500 Recovery failed (with message)

---

## Security

- Same auth as other VM lifecycle endpoints
- Recovery is destructive (removes VM); ensure user intent (explicit Recover click)

---

## Testing

- **Unit:** Connector Undefine with test driver (if test driver supports undefine)
- **Integration:** Route test with stub connector
- **Manual:** Create VM, crash it (or simulate), run recover

---

## Verification Steps

```bash
go build -o bin/ ./cmd/...
go test ./internal/libvirtconn/...
go test ./internal/routes/...
```

- [ ] Connector.Undefine exists and removes domain
- [ ] POST recover returns 200 with status undefined
- [ ] vm_metadata deleted after successful recover
- [ ] Audit log records recover action
- [ ] UI shows Recover for crashed/blocked VMs
- [ ] Recover button triggers API and refreshes list

---

## Decision Log

| Decision | Alternatives | Rationale |
|----------|--------------|-----------|
| Destroy → Undefine | Destroy only | Backlog specifies undefine; full cleanup |
| Delete vm_metadata | Keep orphan record | VM is gone; no need to keep |
| User-initiated only | Auto-detect | Simpler; user explicitly recovers |
| crashed, blocked = stuck | All states | Matches decision-log A19 (libvirt state) |

---

## Ownership

- **In scope:** `internal/libvirtconn/`, `internal/routes/`, `internal/db/` (if DeleteVMMetadata needed), `web/src/lib/api.ts`, `web/src/components/VMList.ts`
- **Out of scope:** Schema, migrations, other domains

---

## Changelog

- 2025-03-16: Initial plan
