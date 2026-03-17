# Plan: Resolve Remaining Gap-Audit Items (9–14)

## Overview

Address Setup Wizard and host validation gaps: specific backend error messages (9, 10, 13), frontend UX fixes (11, 12, 14). Gaps 1–8 already implemented.

---

## Architecture

| Component | Change |
|-----------|--------|
| `normalizeHosts` (routes.go) | Return `([]config.Host, error)` with specific messages instead of `(bool)` |
| `setupComplete` (routes.go) | Use `normalizeHosts` error in 400 response |
| `validateHost` (routes.go) | Pre-check qemu+ssh + keyfile before connecting |
| `SetupWizard.ts` | Host ID listeners, password confirmation, empty-ID validation |

---

## Tasks

### Task 1: Backend — Specific errors from normalizeHosts (Gaps 9, 10)

**File:** `internal/routes/routes.go`

1. Change `normalizeHosts` signature to `([]struct{...}) ([]config.Host, error)`.
2. Return specific errors (order of checks preserved):
   - Empty ID: `"host id is required"`
   - Empty URI: `"host uri is required"`
   - Duplicate ID: `"duplicate host id: X"` (use the duplicate id)
   - qemu+ssh no keyfile: `"Host X: keyfile required for qemu+ssh URI"` (use host id)
3. In `setupComplete`, replace `if !ok` with `if err != nil` and use `writeJSONError(w, http.StatusBadRequest, err.Error())`.

**Acceptance criteria:**
- Duplicate host ID → 400 with `"duplicate host id: local"` (or the actual id)
- qemu+ssh URI with empty keyfile → 400 with `"Host X: keyfile required for qemu+ssh URI"`
- Empty id/uri → 400 with `"host id is required"` / `"host uri is required"`

---

### Task 2: Backend — validateHost qemu+ssh pre-check (Gap 13)

**File:** `internal/routes/routes.go`

1. In `validateHost`, after URI-required check and before `setupConnect`:
   - Compute `hostID := strings.TrimSpace(payload.HostID); if hostID == "" { hostID = "host" }`.
   - If `strings.HasPrefix(strings.TrimSpace(payload.URI), "qemu+ssh://")` and `strings.TrimSpace(payload.Keyfile) == ""`:
   - Return `writeJSON(w, 200, validateHostResponse{Valid: false, Error: fmt.Sprintf("Host %s: keyfile required for qemu+ssh URI", hostID)})` and return.

**Acceptance criteria:**
- Validate-host with qemu+ssh URI and empty keyfile → 200 with `valid: false`, `error: "Host X: keyfile required for qemu+ssh URI"` (no connection attempt).

**Gap 13 resolution:** Fixing Gap 10 covers setup submit. This task covers the validate-host button path. Both paths now return the same conceptual error.

---

### Task 3: Frontend — Default host select on ID change (Gap 11)

**File:** `web/src/components/SetupWizard.ts`

1. In `addHostRow`, after attaching remove/validate listeners, add:
   - `inps.id.addEventListener("input", updateDefaultHostSelect);`
   - `inps.id.addEventListener("change", updateDefaultHostSelect);`
2. Initial row is created via `addHostRow`, so it receives the listener.

**Acceptance criteria:**
- Editing a host ID in place updates the default-host dropdown options immediately.
- Selecting a host, editing its ID, then submitting with the new ID works (no stale default_host).

---

### Task 4: Frontend — Password confirmation (Gap 12)

**File:** `web/src/components/SetupWizard.ts`

1. Add a second password field in admin section:
   - `id="setup-admin-password-confirm"`, `name="admin_password_confirm"`, `autocomplete="new-password"`, label "Confirm password".
2. In submit handler, before building body:
   - If `adminPassword !== (form.querySelector("#setup-admin-password-confirm") as HTMLInputElement)?.value`, set `errorEl.textContent = "Passwords do not match"` and return.
3. Both fields remain `required` for HTML5 validation.

**Acceptance criteria:**
- Mismatched passwords → "Passwords do not match" before submit.
- Matching passwords → submit proceeds.

---

### Task 5: Frontend — Empty host ID validation (Gap 14)

**File:** `web/src/components/SetupWizard.ts`

1. In submit handler, when collecting hosts:
   - Do not auto-fill empty IDs. If `inps.id.value.trim() === ""`, set `errorEl.textContent = "Host ID is required"` and return.
2. Optional: add per-row inline message. Minimal fix: block submit and show error in `errorEl`. For inline UX, add a `span.setup-host-id-error` per row, show "Host ID is required" when field is empty (on blur or input). Plan: implement submit check only; inline span is optional follow-up.

**Acceptance criteria:**
- Any host with empty ID field → submit blocked, "Host ID is required" in error area.

---

## Testing

| Area | Action |
|------|--------|
| `TestNormalizeHosts` | Update for new `([]config.Host, error)` signature; assert `err != nil` and optionally error message content for dup/qemu+ssh cases. |
| `TestSetupComplete_*` | Add or extend tests for duplicate ID and qemu+ssh no keyfile: assert 400 and body contains specific error string. |
| `TestValidateHost_*` | Add `TestValidateHost_QemuSSHNoKeyfile`: assert 200, `valid: false`, `error` contains "keyfile required for qemu+ssh". |
| SetupWizard | Manual or DOM tests for password match, empty host ID, default-host refresh. Add `SetupWizard.test.ts` if patterns exist. |

**Verification:** `make all` (build, test, vet).

---

## Decision Log

| Decision | Alternatives | Rationale |
|----------|---------------|-----------|
| `normalizeHosts` returns `error` | Keep `bool`, add separate error-builder | Single return path; caller uses `err.Error()` directly. |
| validateHost pre-check before connect | Rely on frontend only | Consistent error across validate and setup; no connection attempt for invalid input. |
| Password confirmation in SetupWizard only | Backend validation | Setup is one-time; frontend check sufficient. No backend change. |
| Empty host ID: submit check only | Per-row inline + blur | Minimal scope; inline can be added later if needed. |

---

## Ownership

**In-scope:**
- `internal/routes/routes.go` — normalizeHosts, setupComplete, validateHost
- `internal/routes/routes_test.go` — tests for above
- `web/src/components/SetupWizard.ts` — host ID listeners, password confirm, empty-ID check

**Out-of-scope:**
- `internal/config/` — no schema changes
- Other web components
- API types (no new fields; password confirm is form-only)

---

## Rollout

No config or deployment changes. Frontend and backend changes are backward-compatible for valid inputs.

---

## Changelog

- 2025-03-17: Initial plan for gaps 9–14.
