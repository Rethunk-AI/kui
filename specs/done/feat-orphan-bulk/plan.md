# feat-orphan-bulk — Plan

## Overview

Add bulk operations for orphan VMs: list (reuse existing), bulk claim, and bulk destroy. Users select orphans in the UI, perform bulk actions, and receive structured feedback. Greenfield; no migration paths.

**References:** `docs/prd/backlog.md` (v2 item 4), `docs/prd/decision-log.md` (orphan bulk claim + conflict resolution), existing single-VM claim flow (`POST /api/hosts/{host_id}/vms/{libvirt_uuid}/claim`), `specs/done/feat-stuck-vm/plan.md` (recover = destroy + undefine).

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Web UI (VMList)                                                              │
│  Orphans section: checkboxes per row, "Select all", bulk action bar          │
│  Actions: "Claim selected", "Destroy selected"                               │
│  Feedback: toast + inline summary (claimed N, failed M with reasons)        │
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ API                                                                          │
│  GET  /api/vms                    — existing; returns orphans               │
│  POST /api/orphans/claim           — bulk claim (items[] → claimed/conflicts)│
│  POST /api/orphans/destroy         — bulk destroy (items[] → destroyed/failed)│
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Libvirt Connector (existing)     │  DB (existing)                            │
│  Destroy, Undefine, LookupByUUID  │  UpsertVMMetadataClaim, audit             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Scope

### In scope

- **Bulk claim:** Claim multiple orphans in one request; return per-item success/conflict.
- **Bulk destroy:** Destroy + undefine multiple orphans; return per-item success/failure.
- **Conflict resolution (claim):** Detect and report display_name collision, UUID on multiple hosts, already-claimed; do not auto-resolve—return conflicts for user awareness.
- **UI:** Checkboxes, select-all, bulk action bar, structured feedback.
- **Audit:** Record bulk operations as multiple `vm_lifecycle` events (one per succeeded item) or a single bulk event (decision below).

### Out of scope

- Dedicated `GET /api/orphans` endpoint (orphans remain in `GET /api/vms`).
- Auto-resolution of conflicts (user must fix or retry individually).
- Pagination of orphans (use existing flat list).
- Migration, backfill, backwards compatibility.

---

## API Design

### 1. List orphans

**Existing:** `GET /api/vms` returns `{ vms, hosts, orphans }`. Orphans are `{ host_id, libvirt_uuid, name }`. No change.

### 2. Bulk claim

**Endpoint:** `POST /api/orphans/claim`

**Request:**
```json
{
  "items": [
    { "host_id": "local", "libvirt_uuid": "uuid-1", "display_name": "VM One" },
    { "host_id": "local", "libvirt_uuid": "uuid-2" }
  ]
}
```

- `items`: array of `{ host_id, libvirt_uuid, display_name? }`. `display_name` optional; if omitted, use libvirt domain name.
- Empty `items` → 400 Bad Request.

**Response 200:**
```json
{
  "claimed": [
    { "host_id": "local", "libvirt_uuid": "uuid-1", "display_name": "VM One" }
  ],
  "conflicts": [
    {
      "host_id": "local",
      "libvirt_uuid": "uuid-2",
      "reason": "already_claimed"
    }
  ]
}
```

- `claimed`: successfully claimed orphans.
- `conflicts`: items that could not be claimed, with `reason`:
  - `already_claimed` — domain already in vm_metadata with claimed=true
  - `not_found` — domain not found on host (offline, deleted, or wrong host)
  - `host_offline` — host unreachable
  - `display_name_collision` — optional; if we enforce unique display_name (defer to implementation; may not exist today)

**Errors:**
- 400: empty items, invalid JSON
- 401: unauthorized
- 503: setup required

**Processing:** For each item, in order: resolve host connector → lookup domain → if orphan (not claimed), upsert claim. On success add to `claimed`; on failure add to `conflicts` with reason. Continue on partial failure; return combined result.

### 3. Bulk destroy

**Endpoint:** `POST /api/orphans/destroy`

**Request:**
```json
{
  "items": [
    { "host_id": "local", "libvirt_uuid": "uuid-1" },
    { "host_id": "local", "libvirt_uuid": "uuid-2" }
  ]
}
```

- `items`: array of `{ host_id, libvirt_uuid }`.
- Empty `items` → 400 Bad Request.

**Response 200:**
```json
{
  "destroyed": [
    { "host_id": "local", "libvirt_uuid": "uuid-1" }
  ],
  "failed": [
    {
      "host_id": "local",
      "libvirt_uuid": "uuid-2",
      "reason": "not_found"
    }
  ]
}
```

- `destroyed`: successfully destroyed (Destroy + Undefine) orphans.
- `failed`: items that could not be destroyed, with `reason`:
  - `not_found` — domain not found on host
  - `host_offline` — host unreachable
  - `claimed` — domain is claimed; reject to avoid accidental destruction of tracked VMs (use single recover for claimed)

**Processing:** For each item: resolve host connector → lookup domain → if orphan (not claimed), Destroy then Undefine. On success add to `destroyed`; on failure add to `failed`. Continue on partial failure; return combined result.

**Orphan-only:** Only unclaimed domains may be bulk-destroyed. Claimed VMs must use single `POST /api/hosts/{host_id}/vms/{libvirt_uuid}/recover`.

---

## Security

- Same auth as existing VM endpoints: `mw.UserFromContext` required.
- Bulk destroy rejects claimed VMs; no accidental destruction of tracked VMs.
- Audit: record one `vm_lifecycle` event per succeeded claim/destroy (consistent with single-VM flows).

---

## Data Model

- No schema change. Uses existing `vm_metadata`, `audit_events`.
- Bulk claim: `UpsertVMMetadataClaim` per item.
- Bulk destroy: no metadata (orphans have none); Destroy + Undefine only.

---

## UI Flow

### Selection

- Each orphan row has a checkbox.
- "Select all" checkbox in orphans section header (when expanded).
- Selected state stored in component state; cleared after successful bulk action or on refresh.

### Bulk action bar

- When ≥1 orphan selected: show bar above orphans list with "Claim selected (N)", "Destroy selected (N)".
- Clicking "Claim selected": call `POST /api/orphans/claim` with selected items.
- Clicking "Destroy selected": show confirmation ("Destroy N orphan(s)? This cannot be undone."); on confirm, call `POST /api/orphans/destroy`.

### Feedback

- **Success (all):** Toast "Claimed N orphan(s)" or "Destroyed N orphan(s)"; refresh list.
- **Partial success:** Toast "Claimed N, M failed" or "Destroyed N, M failed"; show inline summary (e.g. expandable list of reasons). Refresh list.
- **All failed:** Toast with first reason; show inline summary. Refresh list.
- **Network error:** Toast "Request failed"; keep selection for retry.

### Display name for bulk claim

- Use libvirt `name` from current orphans data unless user provides override. MVP: no per-item display name edit in bulk UI; use existing names. Optional enhancement: allow inline edit before bulk claim (defer to implementation).

---

## Implementation Approach

### Backend

1. **Routes** (`internal/routes/routes.go`):
   - `POST /api/orphans/claim` → `state.orphansBulkClaim()`
   - `POST /api/orphans/destroy` → `state.orphansBulkDestroy()`

2. **Handlers:**
   - Parse request body; validate non-empty items.
   - For each item: get connector, lookup domain, check claimed vs orphan.
   - Claim: call `UpsertVMMetadataClaim`; record audit.
   - Destroy: call `conn.Destroy` then `conn.Undefine`; record audit.
   - Collect results; return 200 with `claimed`/`conflicts` or `destroyed`/`failed`.

3. **Reuse:** `getConnectorForHost`, `getVMs` orphan discovery logic (domains not in vm_metadata or claimed=false).

### Frontend

1. **API client** (`web/src/lib/api.ts`):
   - `bulkClaimOrphans(items: { host_id, libvirt_uuid, display_name? }[]): Promise<BulkClaimResponse>`
   - `bulkDestroyOrphans(items: { host_id, libvirt_uuid }[]): Promise<BulkDestroyResponse>`

2. **VMList** (`web/src/components/VMList.ts`):
   - Add state: `selectedOrphanIds: Set<string>` (key: `host_id:libvirt_uuid`).
   - Add checkboxes to orphan rows; "Select all" in orphans header.
   - Add bulk action bar when selection non-empty.
   - Wire handlers: call API, show toast/summary, refresh list.

3. **Styles** (`web/src/styles.css`):
   - Add styles for orphan checkboxes, bulk action bar.

---

## Testing

- **Unit:** Handler logic with mock connector (partial success, conflicts, failed).
- **Integration:** Route tests for `POST /api/orphans/claim` and `POST /api/orphans/destroy` with auth, valid/invalid items.
- **Frontend:** VMList tests for checkbox rendering, selection state, bulk bar visibility.

---

## Verification

```bash
go build -o bin/ ./cmd/...
go test ./internal/routes/...
make all
```

- [ ] POST /api/orphans/claim with valid items returns 200 and claimed/conflicts
- [ ] POST /api/orphans/destroy with valid items returns 200 and destroyed/failed
- [ ] Claimed VMs rejected by bulk destroy
- [ ] Audit events recorded per succeeded item
- [ ] UI shows checkboxes, bulk bar, feedback on action

---

## Decision Log

| Decision | Alternatives | Rationale |
|----------|--------------|----------|
| POST /api/orphans/claim, destroy | Extend GET /api/vms with query params | RESTful; bulk ops are mutations; clear semantics |
| Partial success, return claimed/conflicts | Fail entire on first error | User sees what succeeded; can retry failed items |
| Reject claimed in bulk destroy | Allow claimed | Avoid accidental destruction of tracked VMs |
| One audit event per item | Single bulk event | Consistent with single-VM flows; easier audit queries |
| Reuse GET /api/vms for orphans | New GET /api/orphans | Simpler; no pagination needed for MVP |
| No per-item display name in bulk UI | Inline edit | MVP; use libvirt name; can add later |

---

## Ownership

- **In scope:** `internal/routes/routes.go`, `web/src/lib/api.ts`, `web/src/components/VMList.ts`, `web/src/styles.css`
- **Out of scope:** `internal/db/`, `internal/libvirtconn/` (no changes; reuse existing), schema

---

## Changelog

- 2025-03-16: Initial plan
