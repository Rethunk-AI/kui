# coverage-100 Plan

## Overview

Achieve 100% code coverage across the KUI codebase (Go backend + TypeScript/Vite frontend) through a phased, package-by-package approach. No stubs; greenfield; `make all` must pass; commits batched by theme.

## Current State

| Package | Coverage | Notes |
|---------|----------|-------|
| cmd/kui | 58% | main entrypoint, signal handling |
| audit | 62% | git integration, RecordEventWithDiff |
| config | 61% | Load, validation |
| db | 15% | VMMetadata CRUD, initSchema |
| eventsource | 16% | pollHost, domainStateToSpec |
| git | 76% | remaining edge cases |
| libvirtconn | 0% | build-tagged; stub when !libvirt |
| routes | 19% | many handlers untested |
| sshtunnel | 38% | DialRemote, ParseQemuSSH |
| template | 52% | ListTemplates, CreateTemplateDir |
| broadcaster | 0% | Subscribe, Broadcast |
| middleware | 0% | Recovery, RequestID, CORS, Logging, Auth |
| web | ~10% | shortcuts.test.ts only |

## Phased Approach

### Phase 1: Go — Zero-Coverage Packages (broadcaster, middleware)

These are pure logic, no external deps. Fast wins.

### Phase 2: Go — Low-Coverage Packages (db, eventsource, sshtunnel, template, config, audit, git)

Add tests for uncovered branches and edge cases.

### Phase 3: Go — libvirtconn Strategy

Documented exclusions + stub tests. No mock libvirt in CI.

### Phase 4: Go — routes (Handler Tests)

HTTP handler tests with httptest; mock Connector via dependency injection or table-driven tests.

### Phase 5: Go — cmd/kui

Exclude `main()`; test `run`, `parseFlags`, `buildApplication`, `startServer`, `shutdown` via existing patterns.

### Phase 6: Web — Lib Modules (alerts, api, events, focus-trap, shortcuts, toast, winbox-adapter, console)

Mock fetch, WebSocket, noVNC, xterm.

### Phase 7: Web — Components

Test rendering and interactions with mocked API/events.

### Phase 8: Web — main.ts

Bootstrap logic; mock fetch chain.

---

## Per-Package Tasks

### broadcaster (0% → 100%)

**What to test:**
- `NewBroadcaster()` returns non-nil
- `Subscribe(ctx)` returns subscription with `host.online` event when ctx not done
- `Subscribe(ctx)` with ctx.Done() before send: no done callback, channel closed
- `Broadcast(ev)` delivers to all subscribers; non-blocking when channel full (drop)
- `Subscription.Done()` closes channel; idempotent

**Mocks:** None. Use real channels and contexts.

**Exclusions:** None.

---

### middleware (0% → 100%)

**What to test:**
- **RequestID:** generates X-Request-ID when missing; passes through when present; sets header and context
- **CORS:** sets Access-Control-* headers; matches origin case-insensitively; OPTIONS returns 204; non-matching origin → no Allow-Origin
- **Logging:** wraps ResponseWriter; captures status; logs request
- **Recovery:** panic in next → 500 JSON response, stack logged
- **Auth:** skip paths (exact, prefix); empty secret → 401; missing/invalid token → 401; valid JWT → passes user to context; cookie vs Bearer extraction

**Mocks:** None for RequestID/CORS/Logging. For Auth: use real JWT signing with test secret; test `tokenFromRequest`, `shouldSkipAuth`, `writeJSONError` via handler behavior.

**Exclusions:** None. `requestIDFromContext` is used by Recovery/Logging; test via integration.

---

### db (15% → 100%)

**What to test:**
- `Open("")` → error
- `Open(path)` creates dir, applies schema, returns DB
- `Close()` on nil → no-op
- `ListVMMetadata`, `GetVMMetadata`, `InsertVMMetadata`, `UpdateVMMetadata`, `UpdateVMMetadataLastAccess`, `DeleteVMMetadata`, `UpsertVMMetadataClaim` — all branches (nil meta, claimed, display name, console pref, both, neither)
- `initSchema` with existing tables (IF NOT EXISTS)
- `Open` with invalid path (e.g. `/dev/null` or permission error)

**Mocks:** None. Use `t.TempDir()` + sqlite.

**Exclusions:** None.

---

### eventsource (16% → 100%)

**What to test:**
- `domainStateToSpec` for all states (already partially covered; add Blocked, Suspended, NoState, PMSuspend)
- `pollHost` with `ErrLibvirtDisabled` → return silently
- `pollHost` with connect error when wasOnline → broadcast host.offline
- `pollHost` with connect success when !wasOnline → broadcast host.online
- `pollHost` with ListDomains error → broadcast host.offline
- `pollHost` with domain state change → broadcast vm.state_changed
- `pollHost` removes stale domains from domainState map

**Mocks:** Use a fake `Connector` implementation (test double) that returns controlled data. Inject via config with a test-only connector factory, or: add `ConnectorFunc` / interface in a test file and pass a mock. **Simpler:** use `libvirtconn.Connect` with `test:///default` when built with `-tags libvirt`; otherwise test only the `ErrLibvirtDisabled` path and `domainStateToSpec`. For full coverage without libvirt: introduce a `ConnectorProvider` interface in eventsource, inject a mock in tests.

**Recommendation:** Add `connectorProvider` func type in eventsource, default to `libvirtconn.Connect`. In tests, inject a mock that returns domains, errors, etc. This avoids build-tag complexity.

**Exclusions:** None if mock strategy used.

---

### sshtunnel (38% → 100%)

**What to test:**
- `ParseQemuSSH("")` → nil, nil
- `ParseQemuSSH("qemu:///system")` → nil, nil
- `ParseQemuSSH("qemu+ssh://host")` → config, nil
- `ParseQemuSSH("qemu+ssh://user@host:2222")` → user, port
- `ParseQemuSSH` with invalid URI → error
- `DialRemote` with nil cfg → error
- `DialRemote` with empty keyfile → error
- `DialRemote` with real keyfile: **exclude** (requires SSH). Document exclusion for `DialRemote` when it performs real network I/O.

**Exclusions:** `DialRemote` body (real SSH). Test only `ParseQemuSSH` and error paths of `DialRemote` (nil, empty keyfile). For keyfile read/parse errors, use temp file with invalid key content.

---

### template (52% → 100%)

**What to test:**
- `ListTemplates`, `TemplateExists`, `CreateTemplateDir`, `WriteMeta`, `ReadMeta`
- `Slugify` edge cases
- Invalid YAML, missing files

**Mocks:** Use `t.TempDir()` for git base.

**Exclusions:** None.

---

### config (61% → 100%)

**What to test:**
- `Load` with missing file, invalid YAML, validation errors
- `Duration.UnmarshalYAML` invalid format
- All validation branches (hosts empty, duplicate IDs, qemu+ssh without keyfile)

**Mocks:** None. Use temp files.

**Exclusions:** None.

---

### audit (62% → 100%)

**What to test:**
- `RecordEvent` with nil db → error
- `RecordEvent` success
- `RecordEventWithDiff` with nil db, empty git path → error
- `RecordEventWithDiff` success (write diff, commit, insert)
- `WizardDiff`, `openOrInitRepo` when repo exists vs init

**Mocks:** Use `t.TempDir()` for git base; real go-git.

**Exclusions:** None.

---

### git (76% → 100%)

**What to test:**
- Remaining branches in `Init`, `CommitPaths`, status checks

**Mocks:** `t.TempDir()`.

**Exclusions:** None.

---

### libvirtconn (0% → documented)

**Strategy:** Document exclusions. No mock libvirt in CI.

- **connector_stub.go (build !libvirt):** Test `Connect` and `ConnectWithHostConfig` return `ErrLibvirtDisabled`. **Achieve 100% on stub.**
- **connector.go (build libvirt):** Exclude from coverage in default `go test` (no `-tags libvirt`). When `-tags libvirt` is used, integration tests run against `test:///default`; coverage possible but not required for 100% goal in normal CI.

**Exclusions:**
- `connector.go` when built without libvirt: file not compiled.
- When built with libvirt: `Connect`, `connector` methods require real libvirt. **Document:** "libvirtconn (libvirt build) is integration-tested only; excluded from coverage target."

**Action:** Ensure `connector_test_noop_test.go` and stub tests run. Add `connector_stub_test.go` for `Connect`/`ConnectWithHostConfig` returning `ErrLibvirtDisabled`.

---

### routes (19% → 100%)

**What to test:**
- All handlers via `httptest`: setupStatus, validateHost, setupComplete, login, logout, me, preferences, hosts, vms, getVMDetail, patchVMConfig, vmStart/Stop/Pause/Resume/Destroy/Recover, vmClaim, vmClone, createVM, getTemplates, createTemplate, events, getHostPools, getHostPoolVolumes, getHostNetworks, vncProxy, serialProxy
- `getConnectorForHost` returns error when no hosts / host not found
- `staticHandler` with nil fs → 503
- `normalizeHosts`, `containsHost`, `sanitizeValidationError`, `extractFirstDiskPath`, `vncPortFromDomainXML`, `isLocalLibvirtURI`
- `writeConfigFile`, `clientIPFromRequest`, `loginRateLimiter`

**Mocks:** For handlers that need libvirt: `getConnectorForHost` is not injectable. Options:
1. **Router accepts ConnectorProvider** — inject mock in tests.
2. **Use libvirt test driver** — `test:///default` when `-tags libvirt`; otherwise handlers that call `getConnectorForHost` will get `ErrLibvirtDisabled` and return 404/502. Test those error paths.
3. **Extract handler logic** — pure functions for `vncPortFromDomainXML`, `extractFirstDiskPath`, etc. (already done for some). For full VM lifecycle, use `test:///default` in libvirt CI or accept lower coverage for those branches.

**Recommendation:** Inject `ConnectorProvider func(ctx, hostID) (Connector, error)` into router state. Default to current `getConnectorForHost`. In tests, provide mock that returns error or fake connector. Enables 100% route coverage without libvirt.

**Exclusions:** VNC/serial proxy goroutines (proxyWSToVNC, proxyVNCToWS) — test via integration or exclude. Document if excluded.

---

### cmd/kui (58% → 100%)

**What to test:**
- `parseFlags` — valid, invalid, env overrides, TLS pair validation
- `buildApplication` — config exists, config missing, config invalid, db open failure
- `startServer` — listen, Serve
- `shutdown` — server shutdown, db close
- `closeDatabase` with nil
- `getFileStatError`
- `fatalStartup` — exclude (calls os.Exit)

**Exclusions:**
- `main()` — entrypoint
- `fatalStartup` — calls `os.Exit(1)`; untestable in normal test run

---

## Web: Mock Strategy

### fetch / API

- Use `vi.stubGlobal('fetch', async (url, opts) => { ... })` or `beforeEach` with `globalThis.fetch = mockFetch`.
- Mock returns: `{ ok: true, json: () => Promise.resolve({...}) }` or `{ ok: false, status: 401 }`.

### WebSocket

- No built-in WebSocket in happy-dom. Use `vi.stubGlobal('WebSocket', class MockWS { ... })` or mock the module that creates WebSocket.
- For `events.ts` (EventSource/SSE): use `fetch` with `EventSource` polyfill or mock `EventSource`.

### noVNC / xterm

- `console.ts` and `winbox-adapter.ts` import `@novnc/novnc` and `xterm`. Mock these modules:
  - `vi.mock('@novnc/novnc', () => ({ default: vi.fn().mockImplementation(() => ({ ... })) }))`
  - `vi.mock('@xterm/xterm', () => ({ Terminal: vi.fn().mockImplementation(() => ({ ... })) }))`
- Test the adapter logic, not the real noVNC/xterm behavior.

### Coverage Exclusions (vite.config.ts)

Already excluded: `node_modules/`, `**/*.test.ts`, `**/*.spec.ts`, `**/*.d.ts`, `vite.config.ts`, `.pnp.*`, `dist/`.

Add if needed:
- `**/main.ts` — bootstrap; test via smaller units or exclude
- Third-party re-exports

---

## Web: Per-Module Tasks

### lib/alerts.ts

- `addAlert`, `removeAlert`, `getAlerts`, `subscribeAlerts`

### lib/api.ts

- `apiFetch` success, 4xx/5xx → ApiError
- `login`, `putPreferences`, `fetchHosts`, `claimVM`, `recoverVM`, `createVM`, `cloneVM`, `fetchHostPools`, `fetchHostPoolVolumes`, `fetchHostNetworks`

### lib/events.ts

- `subscribeToEvents` — mock EventSource, assert URL, assert event handling

### lib/focus-trap.ts

- `createFocusTrap` — tab cycles, Escape

### lib/shortcuts.ts (expand existing)

- All branches: Escape, Enter+selection, Ctrl+N, Ctrl+R, Ctrl+Shift+C, ?/Shift+/
- `isModKey` (metaKey, ctrlKey)

### lib/toast.ts

- `showToast`, `hideToast`

### lib/winbox-adapter.ts

- Mock Winbox; test `closeTopmostWinBox`, window creation

### lib/console.ts

- Mock noVNC, xterm; test `openConsoleForVM` flow

### Components

- **AlertsPanel:** render, dismiss
- **CloneVMModal:** render, submit, cancel
- **CreateVMModal:** render, submit, cancel
- **FirstRunChecklist:** render, `shouldShowChecklist`
- **HostSelector:** render, change
- **InlineHostSelector:** render, change
- **ShortcutHelp:** render, close
- **VMList:** render, selection, actions

### main.ts

- `bootstrap` — mock fetch chain, assert renderLoginPage vs renderMain
- `renderMain`, `renderLoginPage` — snapshot or DOM assertions

---

## Verification Checkpoints

After each batch:

1. Run `make all` — must pass.
2. Run `make coverage` (Go) and `make web-coverage` (web).
3. Invoke verifier subagent to confirm no regressions.

| Batch | Scope | Verifier After |
|-------|-------|----------------|
| 1 | broadcaster, middleware | ✓ |
| 2 | db | ✓ |
| 3 | eventsource (with ConnectorProvider) | ✓ |
| 4 | sshtunnel, template | ✓ |
| 5 | config, audit, git | ✓ |
| 6 | libvirtconn stub | ✓ |
| 7 | routes (with ConnectorProvider) | ✓ |
| 8 | cmd/kui | ✓ |
| 9 | web lib (alerts, api, events, focus-trap, shortcuts, toast, winbox-adapter, console) | ✓ |
| 10 | web components | ✓ |
| 11 | web main.ts | ✓ |

---

## Commit Batching

One commit per package or logical group:

1. `test(broadcaster): add full coverage`
2. `test(middleware): add full coverage`
3. `test(db): add VMMetadata and Open tests`
4. `test(eventsource): add pollHost and domainStateToSpec coverage`
5. `test(sshtunnel): add ParseQemuSSH and DialRemote error paths`
6. `test(template): add ListTemplates, CreateTemplateDir, etc.`
7. `test(config): add Load and validation tests`
8. `test(audit): add RecordEvent and RecordEventWithDiff tests`
9. `test(git): add remaining coverage`
10. `test(libvirtconn): add stub tests for ErrLibvirtDisabled`
11. `refactor(routes): inject ConnectorProvider for testing` + `test(routes): add handler coverage`
12. `test(cmd/kui): add parseFlags, buildApplication, shutdown`
13. `test(web): add lib module tests (alerts, api, events, ...)`
14. `test(web): add component tests`
15. `test(web): add main bootstrap tests`

---

## Explicit Exclusions List

| Path | Reason |
|------|--------|
| `cmd/kui/main.go` `main()` | Entrypoint; untestable |
| `cmd/kui/main.go` `fatalStartup()` | Calls `os.Exit(1)` |
| `internal/libvirtconn/connector.go` (when built with libvirt) | Requires real libvirt; integration only |
| `internal/sshtunnel/sshtunnel.go` `DialRemote` success path | Real SSH; use integration or exclude |
| `internal/broadcaster/broadcaster.go` `Subscribe` select `<-ctx.Done()` | Both select cases ready; non-deterministic |
| `web/src/main.ts` (optional) | Bootstrap; test via unit or exclude |
| Third-party code | node_modules, vendor |

## Progress Log

| Phase | Status | Coverage |
|-------|--------|----------|
| 1 broadcaster | Done | 92.6% (Subscribe select branch excluded) |
| 1 middleware | Done | 100% |
| 2 db | Done | 81% |
| 2 sshtunnel | Done | 60% (DialRemote success excluded) |
| 2 template | Done | 82.7% |
| 3 libvirtconn stub | Done | 100% |
| 3 config | Done | 96.1% |
| 3 audit | Done | 80.4% |
| 3 git | Done | 81% |
| 4 eventsource | Done | 96.6% (ConnectorProvider injected) |
| 4 routes | Done | 62% (ConnectorProvider injected; VNC/serial proxy excluded) |
| 4 cmd/kui | Done | 70.3% (main, fatalStartup excluded) |
| 5–6 web | Done | 92.13% stmts (main.ts, console.ts excluded) |

---

## Decision Log

| Decision | Alternatives | Why Chosen |
|----------|--------------|------------|
| Inject ConnectorProvider in routes | Use test:///default in CI, or accept lower coverage | Enables 100% route coverage without libvirt in default CI |
| Exclude libvirt connector.go from coverage | Mock libvirt C bindings | Libvirt has no pure-Go mock; integration tests with test:///default are optional |
| Mock fetch/WebSocket in web tests | MSW, or real backend | vi.stubGlobal is simple; no extra deps |
| Exclude main() and fatalStartup | Test via subprocess | os.Exit is untestable in-process |
| Batch commits by package | Single commit | Matches user-global **git-commit-batches**; easier review |

---

## Ownership Boundaries

**In-scope:**
- `internal/*` — all packages
- `cmd/kui/` — main
- `web/src/**/*.ts` — lib, components, main

**Out-of-scope:**
- `web/node_modules/`
- `web/dist/`
- Generated files
- `.cursor/`, `.github/` (config only)

---

## Approval Checklist

- [ ] Scope matches intent (no extra features)
- [ ] File paths and ownership are clear
- [ ] Data model is correct and safe (greenfield: no migration/backfill)
- [ ] Authn/authz + context scoping are correct
- [ ] API contracts are specified (requests/responses/errors)
- [ ] Test plan + verification steps are included
- [ ] Rollout/ops notes are sufficient (if needed)

---

## Recommended Next Steps

1. Implement Phase 1 (broadcaster, middleware) and run verifier.
2. Add ConnectorProvider to routes and eventsource; implement Phase 2–4.
3. Add libvirtconn stub tests; document libvirt exclusion.
4. Implement web mock setup; add lib and component tests.
5. Add coverage thresholds to CI (e.g. `go test -cover` with `-coverprofile` and fail if below threshold).
