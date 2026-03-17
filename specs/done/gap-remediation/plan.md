# Gap Remediation Plan — Gaps 1–4

## Overview

Address four gaps from the gap-audit: (1) createVM network validation, (2) empty-pool guidance in Create/Clone modals, (3) Create VM button when hosts empty, (4) FirstRunChecklist copy. Greenfield only; no migrations.

---

## Task List

### Task 1 — Create VM: network validation (Gap 1)

**Where:** `internal/routes/routes.go` `createVM()` (lines 1818–1973)

**Current:** Pool is validated via `ValidatePool`; network defaults to `"default"` but is never validated. Libvirt fails with generic error if network does not exist.

**Implementation:**

1. After `conn.ValidatePool` (around line 1880) and before disk/volume logic, add network validation:
   - Call `conn.ListNetworks(req.Context())`
   - If error: return 500 with "failed to list networks"
   - Check if `network` exists in returned list (match by `n.Name`)
   - If not found: return 400 with message `"network invalid or does not exist on host"`
2. Use same pattern as `patchVMConfig` (lines 916–932), but return 400 (Bad Request) for invalid network input.

**Acceptance criteria:**

- [ ] Invalid or non-existent network returns 400 with `"network invalid or does not exist on host"`
- [ ] Valid network proceeds to DefineXML as today
- [ ] Unit test: `TestCreateVM_InvalidNetwork` — mock returns networks `["bridge0"]`, request uses `network: "default"` → 400

**Files:** `internal/routes/routes.go`, `internal/routes/routes_test.go`

---

### Task 2 — Create/Clone VM: empty-pool guidance (Gap 2)

**Where:** `web/src/components/CreateVMModal.ts`, `web/src/components/CloneVMModal.ts`

**Current:** When `pools.length === 0`, user sees "Select pool" with no options; no guidance.

**Implementation:**

**CreateVMModal.ts:**

1. After `loadPoolsAndNetworks` populates pool select, if `pools.length === 0`:
   - Show inline message below pool field: `"No storage pools on this host. Create one in virt-manager or virsh."`
   - Disable submit button until pools exist
2. Add a small container (e.g. `modal__hint` or `modal__empty-state`) below the pool label for this message; hide when pools exist.
3. On pool load success with empty result: set `submitBtn.disabled = true`; when pools.length > 0, set `submitBtn.disabled = false`.

**CloneVMModal.ts:**

1. Same pattern: after `loadPools` populates pool select, if `pools.length === 0`:
   - Show inline message: `"No storage pools on this host. Create one in virt-manager or virsh."`
   - Disable submit button until pools exist

**Acceptance criteria:**

- [ ] CreateVMModal: when host has no pools, message shown and submit disabled
- [ ] CloneVMModal: when target host has no pools, message shown and submit disabled
- [ ] When pools load with at least one pool, message hidden and submit enabled

**Files:** `web/src/components/CreateVMModal.ts`, `web/src/components/CloneVMModal.ts`

---

### Task 3 — Create VM button when hosts empty (Gap 3)

**Where:** `web/src/main.ts`, `web/src/components/VMList.ts`, `web/src/components/FirstRunChecklist.ts`

**Current:** Create VM button enabled even when `hosts.length === 0`. User opens modal, sees "No hosts" in selector, dead end.

**Implementation:**

1. **VMList.ts:** Add `hosts: Host[]` (or `hostCount: number`) to `VMListProps`. When rendering Create VM button, disable it when `hosts.length === 0`. Add `title` attribute when disabled: `"Add hosts in setup first"`.
2. **FirstRunChecklist.ts:** Add `hosts: Host[]` to `FirstRunChecklistProps`. When rendering Create VM button, disable it when `hosts.length === 0`. Same `title` when disabled.
3. **main.ts:** Pass `hosts` into `renderVMList` and `renderFirstRunChecklist` (already available in scope).
4. **Shortcuts:** In `main.ts`, pass `onCreateVM: () => { if (hosts.length > 0) openCreateModal(); }` instead of `openCreateModal` directly. No changes to `shortcuts.ts`.

**Acceptance criteria:**

- [ ] VMList Create VM button disabled when `hosts.length === 0`; tooltip explains why
- [ ] FirstRunChecklist Create VM button disabled when `hosts.length === 0`; tooltip explains why
- [ ] Shortcut (e.g. `n` for Create VM) does not open modal when hosts empty
- [ ] When hosts exist, buttons and shortcut work as today

**Files:** `web/src/main.ts`, `web/src/components/VMList.ts`, `web/src/components/FirstRunChecklist.ts`, `web/src/lib/shortcuts.ts` (if shortcut gating is there)

---

### Task 4 — FirstRunChecklist copy (Gap 4)

**Where:** `web/src/components/FirstRunChecklist.ts` (list item around lines 38–40)

**Current:** Says "Create VM from pool or disk path" but does not mention pools must exist.

**Implementation:**

1. Update the first list item from:
   - `"Create VM from pool or disk path"`
   - To: `"Create VM from pool or disk path. Ensure your host has at least one storage pool (create in virt-manager or virsh if needed)."`

**Acceptance criteria:**

- [ ] First list item includes guidance about storage pools

**Files:** `web/src/components/FirstRunChecklist.ts`

---

## Architecture

No new components or services. Changes are localized to:

- **Backend:** `createVM` handler adds pre-DefineXML network validation.
- **Frontend:** Modal empty-state messaging, button disable logic, copy update.

---

## API Contract

**POST /api/vms** (createVM):

- **New behavior:** If `network` (after defaults) is not in `ListNetworks` result → 400, `{"error":"network invalid or does not exist on host"}`.
- No request/response schema changes.

---

## Testing

| Task | Test type | Command |
|------|-----------|---------|
| 1 | Unit | `go test ./internal/routes/... -run TestCreateVM_InvalidNetwork -v` |
| 2 | Manual / existing modal tests | `npm run test` (CreateVMModal, CloneVMModal) |
| 3 | Manual / existing VMList/FirstRunChecklist tests | `npm run test` |
| 4 | Snapshot or assertion on list text | Optional; trivial copy change |

**Verification:** `make all` (build, test, vet).

---

## Decision Log

| Decision | Alternatives | Why |
|----------|---------------|-----|
| Return 400 for invalid network | 409 Conflict (like patchVMConfig) | Invalid input; 400 is correct for bad request payload. |
| Disable Create VM when hosts empty | Show modal with "Add hosts first" | User preference: disable button avoids dead-end modal. |
| Inline message for empty pools | Toast or alert | Inline keeps context; user sees fix path immediately. |
| Pass `hosts` to VMList/FirstRunChecklist | Derive from VMsResponse | `hosts` from fetchHosts is authoritative; VMsResponse.hosts is status map. |

---

## Ownership Boundaries

**In-scope:**

- `internal/routes/routes.go` — createVM network validation
- `internal/routes/routes_test.go` — TestCreateVM_InvalidNetwork
- `web/src/components/CreateVMModal.ts` — empty-pool message + submit disable
- `web/src/components/CloneVMModal.ts` — empty-pool message + submit disable
- `web/src/components/VMList.ts` — Create VM button disable when hosts empty
- `web/src/components/FirstRunChecklist.ts` — copy update + Create VM button disable
- `web/src/main.ts` — pass hosts to VMList, FirstRunChecklist; gate onCreateVM shortcut via wrapped callback

**Out-of-scope:**

- Template create / clone-from-template network validation (Gap 7)
- Domain XML edit network validation (Gap 8)
- 401 / session expiry audit (Gap 6)
- Migrations, backfill, backwards compatibility

---

## Assumptions

- `conn.ListNetworks` is available and returns `[]NetworkInfo` with `Name` field.
- CreateVMModal and CloneVMModal do not support disk-path fallback when pools empty; submit stays disabled.
- `Host` type from `fetchHosts` is the source of truth for "hosts exist."

---

## Changelog

- 2026-03-17: Initial plan (Gaps 1–4).

---

## Approval Checklist

- [ ] Scope matches intent (no extra features)
- [ ] File paths and ownership are clear
- [ ] Data model is correct and safe (greenfield: no migration/backfill)
- [ ] Authn/authz + context scoping are correct
- [ ] API contracts are specified (requests/responses/errors)
- [ ] Test plan + verification steps are included
- [ ] Rollout/ops notes are sufficient (if needed)
