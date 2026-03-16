# Frontend Build & Serving

## 1. Scope & Constraints

### Scope
- Define frontend framework choice, build tooling, bundling, asset layout, and backend serving strategy for MVP.
- Cover development and production workflows for the single-binary Go backend and browser-first UI.
- Specify how WebSocket/SSE and API traffic are routed so console and real-time updates remain reliable.
- Align with functional UI expectations (forms, tables, minimal styling, desktop-first).

### Constraints
- Greenfield-only implementation: no migration paths, backfills, compatibility flags, or legacy feature modes.
  - decision-log: `§0 A10`, `§2` (no migration/compatibility language in binding docs)
- Winbox.js is a hard compatibility constraint and must remain compatible with whichever frontend stack is selected.
  - prd/decision-log: `§2` (`Frontend`), `§4` (`Winbox timing`, `Winbox.js`)
  - decision-log `§4` states frontend build deferred to spec while keeping Winbox.js compatibility explicit.
- Scope includes build outputs up to a maximum of 10 tasks and <800 lines in this spec document.
- Frontend framework choice is deferred to this spec; no framework lock exists in architecture-level stack docs.
  - `docs/prd/stack.md` (`Frontend` deferred), `docs/prd.md §2`.
- No SSR/PWA/service worker features in MVP.
  - prd/decision-log: `§4` (accessibility later, UI kept lightweight)

## 2. Framework Recommendation

### Recommendation
- Primary: **Vanilla JavaScript + Vite**.
- Alternatives (only when project preference requires): React, Vue, or Svelte, with the same build pipeline and explicit Winbox adapter.
  - `docs/prd/stack.md` permits frontend deferment; decision-log explicitly permits standard browser frameworks and no constraints on framework from protocol perspective.

### Rationale
- Winbox.js has no dependencies and supports mounting DOM content or iframe URLs; no framework coupling.
  - research/xyflow-canvas-ui-research.md (Winbox.js overview and API shape)
- noVNC and xterm.js integration are also DOM-agnostic and function equally in vanilla or framework-based apps.
  - decision-log `§2` (`Console protocol`), `§3` (`Order of preference: noVNC → xterm.js`)
- Functional-only UI mandate (`forms`, `tables`, minimal controls) favors low-overhead runtime for fast iteration in MVP.
  - decision-log `§2` (`UI complexity`), `§4` (`UI complexity`)
- Vite supports all listed frontend options without forcing framework-specific assumptions.

### Explicit Winbox.js compatibility
- Required invariant: any frontend rendering model must emit/own a real DOM node for each console container.
- Winbox pattern A (vanilla): create/own container element, pass through Winbox `mount`.
- Winbox pattern B (framework):
  - create framework root element (React/Vue/Svelte root) in a host node,
  - mount framework app into that host node,
  - pass the same host node to Winbox `mount`.
- Constraint: no UI approach may assume Winbox can only host iframes; DOM mounting must be supported.

## 3. Build Tool & Configuration

### Primary Choice
- Use **Vite** as the build tool.
  - Native ESM dev server + production build.
  - Supports vanilla JS and all major frameworks.
  - Simple config surface for API/WebSocket proxy and SPA fallback path behavior.

### Alternative Option
- **esbuild** is acceptable only if dependency footprint needs to be minimized and no framework plugins are required.

### Vite Configuration Requirements (MVP)
- Source root: `web/`.
- Entry: `web/src/main.ts|main.js`.
- Production output: `web/dist`.
- Recommended chunking:
  - `console-vendors` chunk for `@novnc/novnc` and `xterm` to improve cacheability.
- Environment modes:
  - `development`: local dev server behavior and API proxy.
  - `production`: optimized, minified build for embedding.
- Required runtime constants:
  - API base path (for same-origin default).
  - WebSocket base path for real-time endpoint.
- No stub configuration placeholders; all directives must be concrete and usable by Vite.

### Minimal Required `vite.config` behavior
- `build.outDir = 'dist'`.
- `build.emptyOutDir = true`.
- `server.port` fixed default (e.g., 5173) and host accessible for local dev.
- `server.proxy` for at least:
  - `/api` → Go backend.
  - WebSocket upgrade route(s) as required by real-time and console endpoints.

## 4. Asset Structure

### JS Modules
- `web/src/main.ts|main.js` as SPA bootstrap.
- `web/src/lib/` for:
  - API client.
  - WebSocket/SSE client.
  - Console launcher (noVNC/xterm window orchestration).
  - Winbox integration adapter.
- Dependency sources:
  - `winbox` via npm.
  - `@novnc/novnc` via npm or package-compatible source.
  - `xterm` and `xterm-addon-fit` via npm.

### CSS & Styling
- `web/src/styles.css` for global shell tokens and base layout.
- `web/src/components/**/*.css` for module-level styles.
- Keep styling minimal and functional-first to match MVP constraints.
  - decision-log: `§2` (`UI complexity`)

### Static Assets
- `web/public/` for:
  - `favicon.ico`
  - branding/icon assets
  - immutable snapshot placeholders
- Reference paths from compiled runtime via `/assets/...` for cache-safe resolution.

### Directory Layout
- `web/`
  - `public/`
  - `src/`
  - `dist/` (generated)
- `web/src/main.ts|main.js`
- `web/vite.config.(js|ts)`
- `web/index.html`

## 5. Backend Serving

### Supported Serving Options
1. Embed `dist` into the Go binary via `//go:embed` for production.
2. Serve static files from a runtime directory for local dev/test modes.
3. Keep a separate Vite dev server for iterative frontend work.

### Recommended Production/Default Runtime
- Build with Vite → `web/dist` and embed assets into Go binary.
- Serve SPA and static assets from embedded filesystem.
- Keep API and WS endpoints under `/api` and `/ws` (or explicit socket path) to avoid route collision.
- For any non-API path that is not a file, return `index.html` (SPA fallback) unless the path maps to API/WebSocket routes.
- API-first precedence:
  - routes under `/api` are handled by chi handlers first,
  - websocket and SSE handlers are registered before SPA catch-all.

### WebSocket/SSE Integrity
- Do not intercept or downgrade streaming endpoints:
  - `/api/events` (status stream)
  - console/serial websocket routes for noVNC/xterm
- Websocket/SSE paths are explicit and should never pass through static file middleware.

### Security/Deployment Notes
- Embedded serving defaults to single binary + single origin path in production.
- Static paths remain read-only at runtime in embedded mode.
- `config` remains YAML-driven and separate from UI assets.

## 6. Development vs Production Build

### Development
- Run Go API server on a backend port (e.g., `:8080`).
- Run Vite dev server on separate port (default `:5173`).
- Use Vite proxy for:
  - `^/api/` and websocket stream routes to Go.
  - CORS-free local iteration from same-origin-like behavior.
- This flow supports hot reload for frontend and direct backend API feedback.

### Production
- `vite build` from `web/` into `web/dist`.
- Backend `go:embed` serves assets from `dist` and serves SPA fallback.
- Runtime command remains a single binary.
  - architecture/stack alignment: Go backend remains central executable.
- Real-time and console behavior remains backend-owned; no direct console protocol to external hosts.

### Environment Variables
- Build-time frontend vars:
  - `VITE_API_BASE` (default `/api`)
  - `VITE_WS_BASE` (default `/api/ws` or SSE path)
- Runtime backend knobs:
  - optional `KUI_WEB_DIR` for non-embedded static serving mode.
- Keep production defaults to embedded mode so deployment remains deterministic.

## 7. Out of Scope

- Container image/Docker deployment orchestration in this spec.
  - decision-log: `§2`, `§4` (Docker deferred post-MVP)
- SSR, PWA, service worker behavior.
- xyflow integration in canvas/workspace for now.
  - decision-log: `§2` (`VM list UI` uses Winbox), `§4` (`xyflow` deferred)
- Deep optimization and migration-style runtime fallback logic beyond required MVP build serving.
  - greenfield-only.

## References

- `docs/prd/decision-log.md`
- `docs/prd.md`
- `docs/prd/stack.md`
- `docs/prd/architecture.md`
- `docs/research/winbox-ux-research.md`
- `docs/research/xyflow-canvas-ui-research.md`
