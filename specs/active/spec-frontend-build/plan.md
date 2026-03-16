# Spec: Frontend Build & Serving — Plan for spec.md

## Overview

Create `spec.md` that formally specifies the frontend build tooling, asset pipeline, and backend serving strategy for KUI MVP. The spec consolidates decision-log entries (§§0–4), stack, and research into an implementable requirements document. Target: <800 lines or <10 tasks; greenfield only; Winbox.js compatibility is a constraint.

**References cited:**
- `docs/prd/decision-log.md` §§0–4
- `docs/prd.md` §2 (Open Assumptions)
- `docs/prd/stack.md`
- `docs/prd/architecture.md`
- `docs/research/winbox-ux-research.md`
- `docs/research/xyflow-canvas-ui-research.md`

---

## Exploration Findings

### Winbox.js Constraints
- Vanilla JS, dependency-free, framework-agnostic (xyflow-canvas-ui-research.md)
- Mounts DOM elements or opens URLs in iframes; API: `new WinBox()`, window management
- Works with any framework (React, Vue, Svelte) or vanilla JS — no technical compatibility blocker

### Decision-Log Constraints
- Frontend: Recommend based on noVNC/SPICE + real-time updates; framework defer to spec; Winbox.js compatibility is a constraint (§2 Frontend)
- Frontend build: Defer to spec; Winbox.js compatibility is a constraint (§4 Frontend build)
- stack.md §3: Framework deferred to spec; chi for HTTP; noVNC + xterm.js for console
- UI complexity: Functional only (forms, tables, minimal styling); desktop-first

### Backend Context
- Go + chi router (stack.md); no existing frontend or build setup; greenfield

---

## Spec Structure (Implemented)

| Section | Content |
|---------|---------|
| 1. Scope & Constraints | Scope summary; greenfield, Winbox.js, <800 lines; no SSR/PWA |
| 2. Framework Recommendation | Vanilla JS + Vite primary; alternatives; Winbox compatibility patterns |
| 3. Build Tool & Configuration | Vite config; entry/output; proxy; chunking |
| 4. Asset Structure | JS, CSS, static assets; directory layout |
| 5. Backend Serving | Embed vs static dir; SPA fallback; WebSocket/SSE routing |
| 6. Development vs Production Build | Dev server + proxy; prod embed; env vars |
| 7. Out of Scope | Docker, SSR/PWA, xyflow, migration logic |

---

## Verification

- [x] spec.md exists at specs/active/spec-frontend-build/spec.md
- [x] All 7 sections present and populated
- [x] No migration/backfill/backwards-compatibility language (greenfield)
- [x] Line count <800 (actual: ~186)
- [x] Decision-log citations accurate
- [x] Winbox.js compatibility explicitly addressed
