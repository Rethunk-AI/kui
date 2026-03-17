# Setup Host Validation Plan

## Overview / Goals

When a host is added during setup, KUI does not validate that the KVM host has storage pools or networks. Users complete setup, then open "Create VM" and see no pools to select from, with no guidance. This plan adds validation so users get clear feedback before completing setup.

**Goals:**
1. Validate that each host has at least one storage pool and one network before setup completion.
2. Surface specific error messages (e.g. "Host X has no storage pools") so users know what to fix.
3. Integrate validation into the existing setup flow without migration paths (greenfield).

---

## Design Decisions

### When and Where to Validate

**Chosen approach: Option A + Option C**

| Option | Description | Rationale |
|-------|-------------|-----------|
| **A** | Extend `POST /api/setup/validate-host` to check pools/networks | User-initiated "Validate host" gives immediate feedback; natural extension of existing flow. |
| **C** | Validate on submit in `POST /api/setup/complete` | Safety net: users who skip validation still get rejected with clear errors before config is written. |

**Rejected: Option B** (require validation before submit, block "Complete setup") — Would require per-host "validated" UI state and still needs pool/network checks in validate-host. Option A+C provides the same protection with simpler UX.

### API Changes

**`POST /api/setup/validate-host`** (extend existing)

- After successful libvirt connection, before closing:
  1. Call `conn.ListPools(ctx)` and `conn.ListNetworks(ctx)`.
  2. If `len(pools) == 0` → return `{ valid: false, error: "Host {host_id} has no storage pools" }`.
  3. If `len(networks) == 0` → return `{ valid: false, error: "Host {host_id} has no networks" }`.
  4. If both empty → return `{ valid: false, error: "Host {host_id} has no storage pools and no networks" }`.
  5. If both non-empty → return `{ valid: true }` (unchanged).

- Response shape unchanged: `{ valid: bool, error?: string }`.
- Connection failure (existing behavior) still returns `valid: false` with connection error.

**`POST /api/setup/complete`** (add validation before persist)

- After normalizing hosts and before writing config:
  1. For each host, connect via `libvirtconn.Connect(ctx, uri, keyfile)`.
  2. Call `ListPools` and `ListNetworks`.
  3. If any host has no pools or no networks, collect errors and return `400` with a combined message, e.g.:
     - `"Host local has no storage pools"`
     - `"Host local has no networks"`
     - `"Host local has no storage pools. Host remote has no networks."`
  4. If all hosts pass, proceed with existing config write.

- Close each connection immediately after check; do not hold connections open.

### Error Message Format

- Single `error` string per response.
- For validate-host: one host per request → one specific message.
- For setup-complete: concatenate per-host failures with `. ` (period + space).

---

## Architecture

```
┌─────────────────┐     POST /api/setup/validate-host      ┌──────────────────┐
│  SetupWizard    │ ──────────────────────────────────────►│  validateHost()   │
│  (Validate btn) │     { host_id, uri, keyfile }          │  Connect → List   │
└─────────────────┘     { valid, error? }                  │  Pools/Networks   │
         │                                                 └──────────────────┘
         │
         │  POST /api/setup/complete
         ▼  { admin, hosts, default_host }
┌─────────────────┐                              ┌─────────────────────────────┐
│  SetupWizard    │ ───────────────────────────►│  setupComplete()             │
│  (Complete btn) │                              │  For each host: Connect →    │
└─────────────────┘                              │  ListPools/Networks → fail   │
                                                │  if empty; else persist      │
                                                └─────────────────────────────┘
```

---

## Tasks

### Backend

1. **Extend `validateHost()` in `internal/routes/routes.go`**
   - After `conn.Close()` check (or before close, using conn): call `conn.ListPools(ctx)` and `conn.ListNetworks(ctx)`.
   - If connection succeeds, perform pool/network checks before closing.
   - Return `valid: false` with specific error when pools or networks are empty.
   - Ensure `conn.Close()` is called in all paths (defer or explicit).

2. **Extend `setupComplete()` in `internal/routes/routes.go`**
   - After `normalizeHosts` and before config write, add validation loop:
     - For each host, connect via `libvirtconn.Connect(ctx, uri, keyfile)`.
     - Call `ListPools` and `ListNetworks`; close connection.
     - Collect failures (e.g. `"Host {id} has no storage pools"`, `"Host {id} has no networks"`).
   - If any failures, return `400` with combined error message; do not persist.
   - Reuse `sanitizeValidationError` for any libvirt error strings if appropriate.

### Frontend

3. **Update `SetupWizard.ts`**
   - No structural changes required: `validateHost` response `error` is already displayed in `validateResult.textContent`.
   - Ensure `res.error` is shown when `valid: false` (already done).
   - `setupComplete` errors are shown in `errorEl` via `ApiError` (already done).

### Tests

4. **Backend tests in `internal/routes/routes_test.go`**
   - `TestValidateHost_InvalidURIDuringSetup` — already covers connection failure; keep as-is.
   - Add `TestValidateHost_NoPools` — mock or use test driver that returns empty pools (if available); expect `valid: false`, `error` contains "no storage pools".
   - Add `TestValidateHost_NoNetworks` — same pattern for networks.
   - Add `TestSetupComplete_RejectsHostWithNoPools` — setupComplete returns 400 when host has no pools.
   - Add `TestSetupComplete_RejectsHostWithNoNetworks` — same for networks.
   - Note: Tests may need libvirt mock or `test:///default` if it has pools/networks; otherwise use a stub/mock connector. Check existing `routes_test.go` patterns for libvirt usage.

5. **Verification**
   - `go build ./cmd/...`
   - `go test ./internal/routes/...`

---

## Acceptance Criteria

- [ ] When user clicks "Validate host" and the host has no storage pools, the UI shows "Host X has no storage pools".
- [ ] When user clicks "Validate host" and the host has no networks, the UI shows "Host X has no networks".
- [ ] When user clicks "Validate host" and the host has both pools and networks, the UI shows "Valid".
- [ ] When user clicks "Complete setup" and any host has no pools or no networks, setup is rejected with a 400 error and a clear message listing which hosts lack what.
- [ ] When all hosts have at least one pool and one network, setup completes successfully.

---

## Out of Scope (Future Work)

- **CreateVMModal empty-state guidance**: When pools or networks list is empty after setup, show guidance (e.g. "No storage pools on this host. Create one in virt-manager or virsh."). Tracked separately.
- **Storage/network management in KUI**: PRD decision — use existing only; no provisioning UI.
- **Migration or backfill**: None; greenfield only.

---

## Decision Log

| Decision | Alternatives | Why Chosen | Risks / Mitigations |
|----------|--------------|------------|---------------------|
| Extend validate-host + validate on submit | A only, B only, C only | A gives immediate feedback; C ensures no bad state even if user skips validation | Slightly more backend logic; mitigated by shared validation logic pattern |
| Single `error` string in response | `errors: string[]`, structured `{ no_pools, no_networks }` | Keeps existing API shape; frontend already displays `error` | None |
| Validate before persist in setupComplete | Validate after persist | Prevents config write when hosts are unusable for VM creation | None |

---

## Ownership Boundaries

**In scope:**
- `internal/routes/routes.go` — `validateHost()`, `setupComplete()`
- `internal/routes/routes_test.go` — new/updated tests
- `web/src/components/SetupWizard.ts` — only if display logic needs adjustment (likely none)

**Out of scope:**
- `web/src/components/CreateVMModal.ts` — empty-state guidance
- `internal/libvirtconn/` — no interface changes; use existing `ListPools`, `ListNetworks`
- `web/src/lib/api.ts` — types unchanged (`ValidateHostResponse` already has `error?: string`)

---

## Assumptions

- `libvirtconn.Connect` + `ListPools`/`ListNetworks` work with the same URI/keyfile used for connection validation.
- Test environment can exercise validate-host and setup-complete; routes_test may use mocks or `test:///default` if available.
- No change to setup status or auth; setup endpoints remain unauthenticated during setup phase.
