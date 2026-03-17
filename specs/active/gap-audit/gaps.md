# Gap Audit — Obvious Omissions

Post-setup-host-validation audit. These are gaps that "any dumbshit would have thought of" — obvious failure modes that shipped or would ship.

---

## Critical (blocks core flows)

### 1. **Create VM: no network validation**
- **Where:** `internal/routes/routes.go` `createVM()`
- **What:** Pool is validated via `ValidatePool`. Network is NOT validated. We default to `"default"` when empty; if that network doesn't exist on the host, libvirt fails with a generic "failed to create VM".
- **Fix:** Add `ListNetworks` check (or equivalent) before `DefineXML`; return 400 with "network invalid or does not exist on host" if network not in list. Same pattern as `patchVMConfig` (lines 916–932).

### 2. **Create VM / Clone VM: empty pools — no guidance**
- **Where:** `CreateVMModal.ts`, `CloneVMModal.ts`
- **What:** When pools load as `[]`, user sees "Select pool" with no options. No message explaining why or how to fix (e.g. "Create a storage pool in virt-manager or virsh").
- **Fix:** When `pools.length === 0`, show inline message: "No storage pools on this host. Create one in virt-manager or virsh." Disable submit until pools exist (or add disk-path fallback if spec allows).

### 3. **Create VM button when hosts empty**
- **Where:** `main.ts` → `openCreateModal`, `VMList.ts`, `FirstRunChecklist.ts`
- **What:** Create VM button is shown even when `hosts.length === 0`. User opens modal, sees "No hosts" in selector, cannot proceed. Dead end.
- **Fix:** Disable or hide Create VM when `hosts.length === 0`; or show modal with "Add hosts in setup first" message.

---

## High (confusing / poor UX)

### 4. **FirstRunChecklist copy**
- **Where:** `FirstRunChecklist.ts`
- **What:** Says "Create VM from pool or disk path" but doesn't mention that pools must exist or how to create them.
- **Fix:** Add: "Ensure your host has at least one storage pool (create in virt-manager or virsh if needed)."

### 5. **Clone VM: same empty-pool UX**
- **Where:** `CloneVMModal.ts`
- **What:** Same as Create VM — no pools → "Select pool" with no options, no guidance.
- **Fix:** Same empty-state message as Create VM.

### 6. **Session expiry / 401 during use**
- **Where:** `main.ts` bootstrap, `apiFetch` usage
- **What:** On 401 we render login. But if user's session expires mid-session (e.g. while filling Create VM form), does the next API call surface 401 and trigger re-login? Need to verify all `apiFetch` paths handle 401 and redirect to login. Event source / SSE may also need reconnection on auth failure.
- **Fix:** Audit: ensure every authenticated API path surfaces 401 to bootstrap/login flow; add global 401 handler if needed.

---

## Medium (edge cases)

### 7. **Template create: network from template**
- **Where:** `internal/routes/routes.go` template-based create, clone from template
- **What:** Template may reference a network. If that network doesn't exist on target host, clone/create fails. No pre-validation.
- **Fix:** Validate template's network exists on target host before create; return clear error.

### 8. **Domain XML edit: network change**
- **Where:** `putDomainXML` / domain XML edit
- **What:** `patchVMConfig` validates network when provided. Domain XML edit allows arbitrary XML — user could change network to non-existent one; libvirt fails on apply.
- **Fix:** Either validate network in edited XML before apply, or document that libvirt validates (accept opaque error).

---

## Summary

| # | Gap | Severity | Effort |
|---|-----|----------|--------|
| 1 | Create VM: validate network exists | Critical | Small |
| 2 | Create/Clone VM: empty-pool guidance | Critical | Small |
| 3 | Create VM when hosts empty | Critical | Trivial |
| 4 | FirstRunChecklist copy | High | Trivial |
| 5 | Clone VM empty-pool (same as #2) | High | Same as #2 |
| 6 | 401 / session expiry audit | High | Medium |
| 7 | Template network validation | Medium | Small |
| 8 | Domain XML network validation | Medium | Small |

---

## Recommended order

1. **#1** — Backend network validation in createVM (matches patchVMConfig pattern).
2. **#3** — Disable Create VM when hosts empty (one-line conditional).
3. **#2 + #5** — Empty-pool guidance in CreateVMModal and CloneVMModal.
4. **#4** — FirstRunChecklist copy update.
5. **#6** — 401/session audit (explore subagent).
6. **#7, #8** — Template/XML network validation (separate spec if needed).

---

## Setup Wizard & Host Validation (Additional Gaps)

### 9. **Duplicate host IDs — generic backend error**
- **Where:** `SetupWizard.ts`, `normalizeHosts()` in routes.go
- **What:** Backend rejects duplicate host IDs via `normalizeHosts` but returns `"invalid host payload"`. User has no idea it's a duplicate-ID problem.
- **Fix:** Have `normalizeHosts` return a specific error (e.g. `"duplicate host id: local"`); surface in 400 response. Or add frontend pre-validation: check for duplicates before submit and show inline error.

### 10. **qemu+ssh without keyfile — generic backend error**
- **Where:** `SetupWizard.ts`, `normalizeHosts()` in routes.go
- **What:** Backend rejects `qemu+ssh://` URIs with empty keyfile but returns `"invalid host payload"`. User doesn't know keyfile is required.
- **Fix:** Return specific error from `normalizeHosts` (e.g. `"Host X: keyfile required for qemu+ssh URI"`). Or add frontend validation: when URI starts with `qemu+ssh://`, require keyfile and show error before submit.

### 11. **Default host select stale when host ID edited**
- **Where:** `SetupWizard.ts` — `updateDefaultHostSelect()`
- **What:** `updateDefaultHostSelect` runs only on add/remove host. If user edits a host ID in place (e.g. "local" → "host1"), the default-host dropdown stays stale. User can submit with `default_host: "local"` while hosts have `"host1"` — backend rejects with "default_host must be in hosts", but the dropdown is misleading.
- **Fix:** Add `input` or `change` listener on host ID fields to call `updateDefaultHostSelect()` when IDs change.

### 12. **No password confirmation**
- **Where:** `SetupWizard.ts` — admin password field
- **What:** Single password field. User can typo and lock themselves out. No confirmation step.
- **Fix:** Add password confirmation field; validate match before submit. Optional: minimum length hint (backend accepts any non-empty).

### 13. **Validate host doesn't check qemu+ssh + keyfile**
- **Where:** `validateHost()` in routes.go
- **What:** User can click "Validate host" with `qemu+ssh://user@host/system` and empty keyfile. Backend connects via libvirt; connection fails. Error is sanitized but generic (e.g. connection refused, auth failed). No upfront "keyfile required for SSH" message.
- **Fix:** Either (a) add pre-check in validateHost: if URI starts with `qemu+ssh://` and keyfile empty, return `valid: false, error: "keyfile required for qemu+ssh URI"` before connecting, or (b) rely on frontend validation (#10) to block submit; validate-host stays as-is.

### 14. **Empty host ID**
- **Where:** `normalizeHosts`, `SetupWizard.ts`
- **What:** Backend rejects empty host ID. Frontend uses `inps.id.value.trim() || \`host-${hosts.length}\`` so we auto-fill. But if user clears the field, we'd send `host-0`, etc. Actually we'd send `host-${hosts.length}` which could collide (e.g. two rows both "host-0" if first is empty). Wait — we iterate rows, so hosts.length increments. First row empty → id = "host-0". Second row empty → id = "host-1". No collision. But "host-0" might not be what user wants. Low severity.
- **Fix:** Optional: show inline validation "Host ID is required" when empty. Backend already rejects.

### Summary (Setup/Host)

| #  | Gap                               | Severity | Effort |
|----|-----------------------------------|----------|--------|
| 9  | Duplicate host IDs — generic error| High     | Small  |
| 10 | qemu+ssh no keyfile — generic     | High     | Small  |
| 11 | Default host select stale         | Medium   | Trivial|
| 12 | No password confirmation          | Medium   | Small  |
| 13 | Validate host: qemu+ssh keyfile   | Medium   | Small  |
| 14 | Empty host ID (optional)          | Low      | Trivial|
