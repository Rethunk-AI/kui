# Gap 401 — Session Expiry / 401 During Use

## Overview

When a user's session expires mid-session (e.g. while filling the Create VM form), subsequent API calls return 401. Currently only the initial bootstrap `Promise.all` catches 401 and redirects to login. All other `apiFetch`/`fetch` paths surface 401 as generic errors (alerts) or unhandled rejections. EventSource (SSE) does not distinguish auth failure from network errors and has no reconnect/auth-check logic.

This plan: (1) audits all authenticated API paths for 401 handling, (2) adds a global 401 handler so any 401 triggers login redirect, (3) adds SSE auth-failure detection and redirect. Greenfield only; no migrations.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│ main.ts bootstrap                                                        │
│   fetchSetupStatus (no auth) → setup wizard or continue                  │
│   apiFetch /vms, /preferences, fetchHosts → 401 catch → renderLoginPage  │
│   renderMain → subscribeToEvents, VMList, modals, etc.                  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        ▼                           ▼                           ▼
┌───────────────┐         ┌─────────────────┐         ┌─────────────────┐
│ apiFetch()    │         │ fetchDomainXML()│         │ EventSource     │
│ (api.ts)      │         │ putDomainXML()  │         │ /api/events     │
│               │         │ (raw fetch)     │         │ (events.ts)     │
└───────┬───────┘         └────────┬────────┘         └────────┬────────┘
        │                          │                            │
        │ 401                      │ 401                        │ onerror
        ▼                          ▼                            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ Global 401 handler (setOn401)                                            │
│   → renderLoginPage(app, bootstrap)                                     │
│   → clear alerts (optional, avoid stale alerts after re-login)          │
└─────────────────────────────────────────────────────────────────────────┘
```

**Current state:** Only bootstrap's `Promise.all` catch handles 401. All other paths either show a generic alert or let the error propagate (unhandled rejection).

**Target state:** Any 401 from any authenticated API path triggers the global handler → login page. SSE `onerror` triggers an auth-check fetch; if 401, redirect to login.

---

## Scope

**In scope:**
- All `apiFetch` usages (api.ts and callers)
- `fetchDomainXML` and `putDomainXML` (raw fetch in api.ts)
- EventSource `/api/events` (events.ts)
- Callers that catch `ApiError` and must not show alert on 401

**Out of scope:**
- `setupFetch` (setup API) — no auth
- `login` — auth endpoint
- WebSocket / VNC / serial console — separate auth; session expiry on console is a different flow (user can close and re-open)

---

## Audit Findings

### 1. apiFetch paths (all throw ApiError on !res.ok)

| Path | Caller | Current 401 handling |
|------|--------|----------------------|
| `/vms` | main.ts bootstrap | ✅ Caught, renderLoginPage |
| `/preferences` | main.ts bootstrap | ✅ Caught, renderLoginPage |
| `/preferences` | main.ts host selector onChange | ❌ Alert only |
| `/preferences` | main.ts FirstRunChecklist onDismissed | ❌ No try/catch |
| `/preferences` | FirstRunChecklist Dismiss | ❌ Alert only |
| `/hosts` | main.ts bootstrap, fetchHosts | ✅ (bootstrap) / ❌ (onDismissed) |
| `/hosts/{id}/vms/{uuid}/claim` | VMList handleClaim | ❌ Alert only |
| `/hosts/{id}/vms/{uuid}/recover` | VMList handleRecover | ❌ Alert only |
| `/orphans/claim` | VMList handleBulkClaim | ❌ Alert only |
| `/orphans/destroy` | VMList handleBulkDestroy | ❌ Alert only |
| `/vms` POST | CreateVMModal | ❌ Alert only |
| `/hosts/{id}/vms/{uuid}/clone` | CloneVMModal | ❌ Alert only |
| `/hosts/{id}/pools` | CreateVMModal, CloneVMModal | ❌ Alert only |
| `/hosts/{id}/pools/{pool}/volumes` | CreateVMModal | ❌ Alert only |
| `/hosts/{id}/networks` | CreateVMModal | ❌ Alert only |

### 2. Raw fetch paths (domain XML)

| Path | Caller | Current 401 handling |
|------|--------|----------------------|
| GET `/hosts/{id}/vms/{uuid}/domain-xml` | DomainXMLEditor | ❌ Alert only |
| PUT `/hosts/{id}/vms/{uuid}/domain-xml` | DomainXMLEditor | ❌ Alert only |

### 3. EventSource (SSE)

| Path | File | Current behavior |
|------|------|------------------|
| GET `/api/events` | events.ts | `onerror` → close EventSource. No auth check, no redirect. EventSource does not expose HTTP status. |

### 4. Backend

- `/api/events` is protected by Auth middleware (not in SkipExactPaths/SkipPrefixPaths).
- Auth middleware returns 401 JSON on missing/invalid/expired JWT.
- On initial EventSource connect, 401 would be returned before stream starts; EventSource would fire `onerror` (connection failed).
- Mid-stream session expiry: connection stays open; server does not re-check auth. Reconnect after `onerror` would hit 401 on new request.

---

## Implementation Tasks

### Task 1 — Add global 401 handler in api.ts

**Where:** `web/src/lib/api.ts`

**Implementation:**

1. Add module-level `on401Handler: (() => void) | null = null`.
2. Export `setOn401(handler: () => void): void` to register the handler.
3. In `apiFetch`, when `!res.ok` and `res.status === 401`:
   - If `on401Handler` is set, call it.
   - Throw `ApiError` as today (callers may still catch; handler has already redirected).
4. In `fetchDomainXML` and `putDomainXML`, when `!res.ok` and `res.status === 401`:
   - If `on401Handler` is set, call it.
   - Throw `ApiError` as today.

**Acceptance criteria:**

- [x] `setOn401` is exported and stores the handler.
- [x] `apiFetch` calls handler on 401 before throwing.
- [x] `fetchDomainXML` and `putDomainXML` call handler on 401 before throwing.
- [x] Unit test: mock fetch returns 401, handler called, ApiError thrown.

**Files:** `web/src/lib/api.ts`, `web/src/lib/api.test.ts`

---

### Task 2 — Register handler and redirect in main.ts

**Where:** `web/src/main.ts`

**Implementation:**

1. Before `bootstrap()` is invoked, call `setOn401(() => { clearAlerts(); renderLoginPage(app, bootstrap); })`.
2. Add `clearAlerts()` to `web/src/lib/alerts.ts` if not present; export it.
3. Simplify bootstrap's existing 401 catch to `if (e instanceof ApiError && e.status === 401) return;` (handler already ran) then `throw e;`.
4. Add try/catch to FirstRunChecklist `onDismissed` callback in main.ts; in catch, `if (err instanceof ApiError && err.status === 401) return;`.

**Acceptance criteria:**

- [x] `setOn401` called with redirect logic before `bootstrap()`.
- [x] On 401 from any path, login page is shown.
- [x] Alerts cleared on redirect (no stale alerts after re-login).

**Files:** `web/src/main.ts`, `web/src/lib/alerts.ts` (if clearAlerts added)

---

### Task 3 — SSE: auth check on EventSource error

**Where:** `web/src/lib/events.ts`

**Implementation:**

1. Import `apiFetch` from `./api`. EventSource does not expose HTTP status; on `onerror` we cannot distinguish 401 from network errors. Trigger an auth check: call `apiFetch("/api/auth/me")`. If 401, `apiFetch` invokes the global handler (redirect to login). If 200, no action (EventSource stays closed; user can refresh).
2. In `es.onerror`, after `es.close()`, call `apiFetch("/api/auth/me").catch(() => {})`. Do not await; fire-and-forget. The fetch completes asynchronously; if 401, the global handler runs.
3. Optional: throttle auth check if `onerror` fires repeatedly (e.g. network down) — max once per 5s. See Task 5.

**Acceptance criteria:**

- [x] EventSource onerror triggers auth check via `apiFetch("/api/auth/me")`.
- [x] If 401, global handler runs (redirect to login).
- [x] If 200, no action (EventSource stays closed; user can refresh).
- [x] Unit test: mock EventSource onerror, mock apiFetch 401, assert handler called (or assert redirect via some seam).

**Files:** `web/src/lib/events.ts`, `web/src/lib/events.test.ts`

---

### Task 4 — Callers: skip alert on 401

**Where:** All components that catch and show alert for ApiError.

**Implementation:**

At the start of each catch block that handles ApiError, add:
```ts
if (err instanceof ApiError && err.status === 401) return;
```
This prevents showing a generic "Failed to..." alert when the session expired — the user is already on the login page.

**Call sites to update:**

1. `web/src/main.ts` — host selector onChange (line ~151)
2. `web/src/components/FirstRunChecklist.ts` — Dismiss (line ~68)
3. `web/src/components/CreateVMModal.ts` — loadPoolsAndNetworks, loadVolumes, createVM submit (lines ~247, ~268, ~360)
4. `web/src/components/CloneVMModal.ts` — loadPools, cloneVM submit (lines ~150, ~212)
5. `web/src/components/DomainXMLEditor.ts` — fetchDomainXML, putDomainXML (lines ~136, ~162)
6. `web/src/components/VMList.ts` — handleClaim, handleRecover, handleBulkClaim, handleBulkDestroy (lines ~431, ~451, ~493, ~536)

**Note:** `SetupWizard.ts` uses `/api/setup/*` (no auth) — no 401 handling needed.

**Note:** main.ts FirstRunChecklist `onDismissed` (lines 240–246) has no try/catch. Add try/catch; in catch, if 401 return. This avoids unhandled rejection when session expires during the post-dismiss fetch.

**Acceptance criteria:**

- [x] Each listed catch block returns early on 401.
- [x] No "Failed to..." alert shown when session expired.

**Files:** `web/src/main.ts`, `web/src/components/FirstRunChecklist.ts`, `web/src/components/CreateVMModal.ts`, `web/src/components/CloneVMModal.ts`, `web/src/components/DomainXMLEditor.ts`, `web/src/components/VMList.ts`

---

### Task 5 — Optional: throttle SSE auth check

**Where:** `web/src/lib/events.ts`

**Implementation:**

If `onerror` fires repeatedly (e.g. network down), throttle the auth check to avoid many fetches. Use a simple debounce: only run auth check if last check was > 5 seconds ago.

**Acceptance criteria:**

- [ ] Rapid onerror events do not trigger auth check more than once per 5 seconds.

**Files:** `web/src/lib/events.ts`

**Note:** Can be deferred if Task 3 is sufficient.

---

## Acceptance Criteria (Overall)

- [x] Session expiry during any authenticated API call triggers redirect to login.
- [x] No generic "Failed to..." alert when the session expired — user sees login.
- [x] EventSource error triggers auth check; 401 redirects to login.
- [x] Alerts cleared on redirect (no stale alerts after re-login).
- [x] All audit findings documented in this plan.

---

## Verification Steps

1. **Unit tests**
   - `api.test.ts`: 401 triggers handler, ApiError thrown.
   - `events.test.ts`: EventSource onerror triggers auth check; mock 401, assert handler invoked.

2. **Manual test**
   - Log in, wait for session to expire (or manually clear cookie / invalidate JWT), then:
     - Change host selector → should redirect to login.
     - Submit Create VM form → should redirect to login.
     - Submit Clone VM form → should redirect to login.
     - Submit Dismiss on FirstRunChecklist → should redirect to login.
     - Claim orphan, recover VM, bulk claim/destroy → should redirect to login.
     - Load domain XML editor, save domain XML → should redirect to login.
     - With EventSource connected: invalidate session, trigger server to close SSE (or wait for network blip) → auth check runs, 401 → redirect to login.

3. **Build**
   - `cd web && yarn build` succeeds.
   - `make all` passes.

---

## Files Summary

| File | Changes |
|------|---------|
| `web/src/lib/api.ts` | setOn401, call from apiFetch/fetchDomainXML/putDomainXML |
| `web/src/lib/api.test.ts` | Test 401 handler |
| `web/src/lib/alerts.ts` | clearAlerts (if not present) |
| `web/src/lib/events.ts` | onerror → auth check |
| `web/src/lib/events.test.ts` | Test auth check on error |
| `web/src/main.ts` | setOn401, clearAlerts on redirect, onDismissed try/catch, caller 401 skip |
| `web/src/components/FirstRunChecklist.ts` | 401 skip in catch |
| `web/src/components/CreateVMModal.ts` | 401 skip in catch |
| `web/src/components/CloneVMModal.ts` | 401 skip in catch |
| `web/src/components/DomainXMLEditor.ts` | 401 skip in catch |
| `web/src/components/VMList.ts` | 401 skip in catch |
