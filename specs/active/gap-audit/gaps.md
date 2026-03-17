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
