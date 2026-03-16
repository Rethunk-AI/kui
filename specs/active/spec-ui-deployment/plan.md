# Spec: UI & Deployment — Plan for spec.md

## Overview

Create `spec.md` that formally specifies the UI and deployment design for KUI MVP. The spec consolidates decision-log entries (§§0–4), architecture, and research into an implementable requirements document. Target: <800 lines or <10 tasks; greenfield only; Winbox.js compatibility is a constraint.

**References to cite:**
- `docs/prd/decision-log.md` §§0–4
- `docs/prd/architecture.md`
- `docs/research/winbox-ux-research.md`
- `docs/research/xyflow-canvas-ui-research.md` (xyflow deferred)

---

## Spec Structure

The spec.md shall have the following sections. Each maps to decision-log topics and scope items.

### 1. Scope & Constraints

- **Content:** Spec scope summary (Winbox.js Canvas, host selector, alerts, first-run checklist, VM list, systemd deployment, TLS). Constraints: greenfield, <800 lines, Winbox.js compatibility; framework deferred to spec; xyflow deferred.
- **Sources:** User-provided scope; decision-log §2 (Docs scope, Frontend, VM list UI).

### 2. Winbox.js Canvas Integration

- **Content:**
  - Canvas as primary VM console workspace: draggable, resizable VM console windows.
  - MVP from day one (decision-log §4: Winbox timing).
  - Each VM console opens in a Winbox.js window; content: noVNC or xterm.js (per decision-log §3).
  - No docking/tabbing/layout persistence in MVP (xyflow research: Winbox.js limitation noted).
  - Golden Layout discarded for this use-case (decision-log §4: Winbox.js).
- **Sources:** decision-log §2 (VM list UI, Console protocol), §4 (Winbox.js, Winbox timing, xyflow); winbox-ux-research.md; xyflow-canvas-ui-research.md (Winbox.js vs xyflow comparison).

### 3. Host Selector

- **Content:**
  - Global default: config defines default; user override in preferences (SQLite).
  - Contextual override: create VM, clone VM, open console — each flow can override host.
  - Placement: global header/sidebar for default; inline in create/clone/console modals for override.
  - Flow: user sees current default; can change per-action or persist via preferences.
- **Sources:** decision-log §2 (Default host), §4 (Default host).

### 4. Alerts Panel

- **Content:**
  - Dismissible alerts for: host offline, clone/create failed, connection errors, VM state changes.
  - In-memory only; disappear on refresh (no persistence).
  - Toast for transient feedback; alerts panel for persistent-in-session items.
  - Optional "Show details" with sanitized technical summary (no stack trace).
- **Sources:** decision-log §2 (Notifications, Error handling), §4 (Notifications).

### 5. First-Run Checklist

- **Content:**
  - MVP: "Create VM" only (pool+path or clone); "Create Template" in v2.
  - Visible when VM list empty.
  - Completed when: user creates a VM OR dismisses checklist.
  - Separate from setup wizard (config/DB missing); checklist is onboarding when DB exists but no VMs.
- **Sources:** decision-log §2 (Empty state, First-run), §4 (Empty state).

### 6. VM List UI & Grouping

- **Content:**
  - Flat list: all VMs in one view; do not separate by host.
  - Group by month/year; user-configurable date (default: last-access, fallback: creation).
  - Within group: display name (from vm_metadata).
  - Unclaimed domains in separate section/tab; offer to import/claim.
  - Data: libvirt for domain list + state; vm_metadata for KUI fields (display_name, etc.).
  - MVP: simple relative-time badge (e.g. "started 2m ago").
- **Sources:** decision-log §2 (VM list, VM metadata, Orphan domains), §4 (VM list grouping, Orphan handling).

### 7. Systemd Deployment

- **Content:**
  - Primary deployment: systemd unit.
  - Unit file: Type=simple or notify; Restart=on-failure; config path /etc/kui/config.yaml.
  - Data paths: /var/lib/kui/ for git (templates, audit), SQLite.
  - Docker post-MVP; document as future.
- **Sources:** decision-log §2 (Deployment, Config), §4 (Docker priority); architecture §4.

### 8. TLS & Reverse Proxy

- **Content:**
  - KUI can optionally serve TLS (config-driven).
  - Reverse proxy recommended for prod (nginx, Caddy, etc.).
  - HTTP for dev.
  - Document both options.
- **Sources:** decision-log §2 (TLS), §4 (TLS/HTTPS).

### 9. Out of Scope (this spec)

- xyflow (topology/infrastructure maps) — deferred.
- Docker deployment — post-MVP.
- Layout persistence, docking, tabbing — post-MVP.
- Create Template flow — v2 (exists but de-emphasized in MVP).

---

## Implementation Instructions for Developer

1. Create `specs/active/spec-ui-deployment/spec.md`.
2. Write each section per the structure above. Use declarative requirements; cite decision-log entries where binding.
3. Keep total length <800 lines. Use concise bullets and tables.
4. Do not invent new requirements; derive only from decision-log, architecture, and research.
5. Format: Markdown with clear headings (##, ###). Include a brief "References" section linking to docs/prd/decision-log.md, architecture.md, and research docs.

---

## Verification

- [ ] spec.md exists at specs/active/spec-ui-deployment/spec.md
- [ ] All 9 sections present and populated
- [ ] No migration/backfill/backwards-compatibility language (greenfield)
- [ ] Line count <800
- [ ] Decision-log citations accurate
