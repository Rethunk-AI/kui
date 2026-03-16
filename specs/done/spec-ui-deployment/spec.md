# UI & Deployment Specification

## 1. Scope & Constraints

### 1.1 In Scope
- Define UI and deployment behavior for MVP KUI, including:
  - Winbox.js canvas and console workflow
  - Host selector behavior
  - Alerts panel and feedback patterns
  - First-run checklist and onboarding flow
  - VM list grouping and orphan-domain handling
  - Systemd deployment, TLS, and reverse-proxy options
- All requirements are implementation directives for code, not placeholders or compatibility modes.
- The spec is authoritative for UI/infra behavior in MVP.

### 1.2 Core Constraints
- Greenfield implementation only: no migration paths, no backfill, no compatibility mode. (decision-log §0 A10; `greenfield.mdc`)
- Single-tenant baseline (1–2 concurrent users, single admin) with desktop-first, functional UI. (decision-log §§2 `Concurrent users`, `Role model`, `UI complexity`, `Mobile support`)
- Console and list behavior must align with:
  - `docs/prd/architecture.md` component boundaries and data flow
  - `docs/prd/decision-log.md` §§2 and §§4
  - Winbox and xyflow research findings (decision: Winbox.js for Canvas, xyflow deferred)
- No more than 800 lines in this spec document. (plan and docs scope)

## 2. Winbox.js Canvas Integration

### 2.1 Requirements
- The VM console workspace uses Winbox.js as the primary canvas for live console windows. (decision-log §2 `VM list UI`; winbox-ux-research; xyflow-canvas-ui-research)
- Each opened console MUST render inside a Winbox.js child window with:
  - draggable, resizable behavior
  - independent focus/stacking
  - manual close behavior
- Multiple VM console windows MAY be open concurrently in the same browser session. (winbox-ux-research implications)
- The first and default integration path is Winbox.js MVP from day one. (decision-log §4 `Winbox timing`)
- Golden Layout / docking-pane persistence is not part of this spec; no docking, tabbing, or persisted layout in MVP. (decision-log §4 `Winbox.js`; §4 `Orphan handling`; xyflow-canvas-ui-research limitations)

### 2.2 Console content inside windows
- Console selection is per VM using KUI metadata first, then libvirt XML-derived preference flow. (decision-log §2 `Console selection`, `Console protocol`)
- NoVNC is the primary console protocol in MVP; xterm.js is the fallback. (decision-log §2 `Console protocol`, §3 console order)
- Console traffic is proxied by KUI for browser safety and remote host traversal. (decision-log `Console proxy`)
- SSH/SPICE-specific web gateway alternatives are not required for MVP. (decision-log `Console protocol order`; winbox-ux-research related component notes)

## 3. Host Selector

### 3.1 Defaults and overrides
- There is exactly one default host in config and no unset host state; all operations are host-scoped. (decision-log §2 `Config`, `Libvirt connection`, `Default host`)
- Users can persist a global default host override in preferences. (decision-log §2 `Default host`, `Preferences scope`)
- Host selection appears in contextual flows for:
  - create VM
  - clone VM
  - open console
  where this override can differ from the global default. (decision-log §2 `Default host`; architecture flow examples)
- The VM list and alerts MUST support unreachable/offline hosts with an explicit host-aware status state. (decision-log §4 `Host offline`, architecture topology)

### 3.2 UX placement
- The global default host control is located in primary navigation/context surface.
- Contextual host override controls appear inline in create/clone/console action workflows only.

## 4. Alerts Panel

### 4.1 Alert taxonomy
- In-session alerts MUST include at minimum:
  - host offline/unreachable
  - create failure
  - clone failure
  - console connection failure
  - VM state transitions/failures
  - host connection errors  
  (decision-log §2 `Notifications`, `Host offline`, `Error handling`)
- Alerts that are transient or low-friction MUST use toast notifications.
- Alerts that require operator awareness across the current session MUST also appear in an alerts panel. (decision-log `Notifications`)

### 4.2 Lifecycle and detail policy
- Alerts are in-memory only and must be cleared on refresh; no local persistence. (decision-log §4 `Notifications`)
- Optional "Show details" MAY expose sanitized technical context only.
- Raw stack traces and internal logs remain backend-only. (decision-log §2 `Error handling`)

## 5. First-Run Checklist

### 5.1 Checklist scope
- Onboarding checklist is shown when:
  - VM list is empty
  - `onboarding_dismissed` is false (from user preferences)
  - configuration/db are present but no VMs are tracked yet. (decision-log §2 `Empty state`, §4 `Empty state`; architecture flow)
- Checklist visibility: (VM list empty) AND (not `onboarding_dismissed`).
- Checklist tasks include only MVP VM creation paths:
  - create VM from pool/path
  - or clone VM  
  ("Create Template" is explicitly out of MVP first-run path, but exists as lower-priority backlog). (decision-log §§2/4 `Empty state`, §2 `VM create flow`, A4)

### 5.2 Completion rules
- Checklist is marked complete when:
  - user performs a VM creation action, or
  - user explicitly dismisses the checklist. (decision-log `Empty state`)
- Dismiss action: UI calls `PUT /api/preferences` with `list_view_options.onboarding_dismissed: true` to persist the dismissed state server-side.
- Setup wizard remains separate and only addresses initial config/admin/bootstrap prerequisites. (decision-log `First-run`, `Setup wizard scope`)

## 6. VM List UI & Grouping

### 6.1 List shape and data
- VM list is a single flat list across all hosts, not host-separated partitioning. (decision-log §4 `VM list grouping`)
- Data source includes:
  - libvirt domain inventory/state
  - vm_metadata for UI fields and behavior (display name, claimed state, host key metadata). (decision-log `VM metadata`, `VM list`)
- Grouping is month/year across VM rows.
- Group key is user-configurable:
  - primary: last-access date
  - fallback: creation date
  - default: last-access. (decision-log §2 `VM list`, §4 `VM list grouping`)
- Row display uses KUI `display_name` as primary label and shows concise relative-time badges. (decision-log `VM metadata`, `VM list UI`)

### 6.2 Orphans and claims
- Unclaimed/orphan domains MUST be surfaced in a separate section/tab from tracked VMs. (decision-log §4 `Orphan handling`, §2 `Orphan domains`)
- For each orphan, user MUST be able to import/claim into KUI-tracked metadata.
- Claimed host association for an imported orphan is the host where it was discovered. (decision-log `Orphan handling`)

### 6.3 View behavior
- The list is optimized for 5–20 VMs. (decision-log `VM scale (MVP)`)
- Search/filter UX is not part of MVP list requirements; canvas interaction remains preferred for multi-VM tasks. (decision-log `Search/filter`)

## 7. Systemd Deployment

### 7.1 Primary deployment target
- KUI SHALL be deployed via systemd as the primary mechanism. (decision-log §2 `Deployment`, `Docs scope`; A7)
- Docker/container deployment is explicitly post-MVP and not required in this spec. (decision-log §4 `Docker priority`)

### 7.2 Service contract
- Service should be created with:
  - `Type=simple` (or `Type=notify` when readiness signaling is implemented)
  - `Restart=on-failure`
  - config path default: `/etc/kui/config.yaml` unless explicitly overridden. (decision-log `Config`)
- Process must support local and remote libvirt hosts through the same service binary.

### 7.3 Storage and runtime paths
- Runtime data path:
  - `/var/lib/kui/` for git-backed templates/audit chain and SQLite DB when using defaults. (decision-log §§2 `Database`, `Config`; architecture persistence boundary)
- Configuration and host credentials are read from YAML config at runtime; Docker env override is only a post-build concern.

## 8. TLS & Reverse Proxy

### 8.1 Service mode
- KUI MAY serve TLS directly when configured.
- KUI MUST document and support reverse-proxy deployment (e.g., nginx/Caddy) for production TLS and hardening. (decision-log §2 `TLS`, `HTTPS`)
- HTTP-only mode remains valid for development. (decision-log §2 `TLS`, `Frontend`)

### 8.2 Compatibility and routing
- When reverse-proxying, routing must preserve WebSocket behavior required for:
  - console streaming
  - real-time VM and host-state updates.

## 9. Out of Scope

- xyflow-based topology/infrastructure canvas. (decision-log §§2/4 `VM list UI`, `xyflow`)
- Golden Layout-style docking/persistence/tabbed workspace for console containers in MVP.
- Layout persistence, docking, and advanced workspace macros.
- Template-based create flow (`Create VM from template`) in MVP; template creation is de-emphasized and scheduled for V2. (A4, §2 `VM templates`, §4 `MVP scope`)
- Docker deployment strategy is post-MVP. (A7, decision-log `Deployment`)
- Desktop/mobile parity or accessibility AAA enhancements beyond current scope; accessibility policy is deferred to later versions. (decision-log `Accessibility`)

## References

- `docs/prd/decision-log.md`
- `docs/prd/architecture.md`
- `docs/research/winbox-ux-research.md`
- `docs/research/xyflow-canvas-ui-research.md`
