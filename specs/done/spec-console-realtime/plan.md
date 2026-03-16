# Plan: Console & Real-time Spec

## Overview

Create `spec.md` for the Console & Real-time feature. The spec defines noVNC proxy behavior, xterm.js integration, real-time mechanism, and event types. Target: <800 lines or <10 tasks. Greenfield only. MVP: noVNC + xterm.js only (SPICE deferred to v2).

## Decision-Log References (to cite in spec)

| Section | Topic | Key content |
|---------|-------|-------------|
| §0 A12 | Console proxy | KUI proxies all console traffic (browser → KUI → libvirt host) |
| §2 | Console protocol | MVP: noVNC + xterm.js; SPICE v2. Selection: per-VM preference in vm_metadata first; else libvirt domain XML → noVNC → xterm.js |
| §2 | Console proxy | For remote: KUI opens SSH tunnel and forwards console stream. KUI can run anywhere. Prefer URL-based; needs exploration in spec |
| §2 | Real-time updates | v1: VM state + host online/offline. v2: clone/create progress. Prefer single WebSocket; else WebSocket for console, SSE for status |
| §3 | Console protocol order | 1. noVNC (primary) 2. xterm.js (fallback) 3. SPICE (v2) |
| §4 | Console proxy | Same as §2; URL-based exploration |
| §4 | Real-time mechanism | Same as §2 |
| §4 | Console selection | Per-VM preference + libvirt fallback (SPICE → noVNC → xterm.js) |

## PRD Open Assumptions (to resolve in spec)

- **Console proxy design** (prd.md §2): Prefer URL-based (wss://kui/...); proxy implementation may be required; needs exploration in spec.

## Spec Structure (sections for spec.md)

### 1. Scope & Constraints

- MVP: noVNC + xterm.js only; SPICE out of scope
- Greenfield; no migration
- Target <800 lines or <10 tasks
- References: decision-log §§0–4, §3; prd.md §2; architecture.md

### 2. noVNC Proxy Behavior

**2.1 URL design**

- URL-based: `wss://{kui-host}/api/hosts/{host_id}/vms/{vm_id}/vnc` (or equivalent path)
- Path encodes host_id and vm_id for authz and routing
- Query params: optional (e.g. `?token=...` for short-lived JWT if needed)

**2.2 WebSocket handling**

- KUI HTTP server upgrades WebSocket requests on the console path
- KUI validates JWT/session; resolves host + VM; checks user can access VM
- Local host: KUI connects to libvirt, gets VNC port via domain XML/DisplayInfo, connects to `127.0.0.1:{port}` (or unix socket if applicable)
- Remote host: KUI establishes SSH tunnel to libvirt host, forwards to VNC port on remote
- KUI proxies bytes bidirectionally: browser ↔ KUI ↔ VNC backend

**2.3 SSH tunnel for remote**

- Use host config: `qemu+ssh://user@host/system?keyfile=...`
- KUI spawns SSH tunnel (e.g. `ssh -L local_port:127.0.0.1:vnc_port user@host`) or uses Go SSH client to forward
- Tunnel lifecycle: create on first console connect; tear down when WebSocket closes

**2.4 Error handling**

- 401/403 on auth failure
- 404 if VM or host not found
- 502 if VNC backend unreachable (VM not running, port not available)
- Graceful close with reason on tunnel/proxy failure

### 3. xterm.js Integration

**3.1 When to use**

- Fallback when VNC not available (domain has no VNC graphics; only serial console)
- Per decision-log §3: "for bare serial or console access"

**3.2 URL design**

- `wss://{kui-host}/api/hosts/{host_id}/vms/{vm_id}/serial` (or `/console`)

**3.3 Backend**

- Libvirt serial console: `virDomainOpenChannel` or equivalent for serial devices
- Local: direct libvirt connection
- Remote: SSH tunnel to libvirt host; KUI proxies serial stream over WebSocket

**3.4 Frontend**

- xterm.js instance in Winbox.js window; connects to WebSocket URL
- Same authz as noVNC path

### 4. Real-time Mechanism

**4.1 Choice**

- Prefer single WebSocket for status + console
- Fallback: WebSocket for console, SSE for status
- Spec must decide: recommend single WebSocket if feasible (simpler); else document split

**4.2 Endpoint**

- If single WS: `wss://{kui-host}/realtime` — client subscribes to events; console uses separate path or same WS with message routing
- If split: `GET /api/events` (SSE) for status; `wss://.../api/.../vnc` and `wss://.../api/.../serial` for console

**4.3 Recommendation for spec**

- Single WebSocket is complex (multiplexing console + status). Simpler: **WebSocket for console only**; **SSE for status** (one-way, simpler, fits VM state + host online/offline). Document rationale.

### 5. Event Types (Real-time v1)

**5.1 VM state**

- `vm.state_changed` — payload: `{ host_id, vm_id, state }` (e.g. running, paused, shut off)
- Emitted when domain state changes (libvirt domain event or poll)

**5.2 Host online/offline**

- `host.offline` — payload: `{ host_id, reason? }`
- `host.online` — payload: `{ host_id }`
- Emitted when libvirt connection fails or recovers

**5.3 v2 (out of scope for spec)**

- clone/create progress — document as future; no schema in this spec

### 6. Authz & Security

- All console and real-time endpoints require valid JWT
- VM access: user must have permission (MVP: single admin; all VMs)
- Host access: user must have permission to host (MVP: all hosts)
- No direct browser→libvirt; all traffic via KUI

### 7. Out of Scope (explicit)

- SPICE
- Clone/create progress events (v2)
- Multi-user RBAC (MVP: single admin)

## Ownership

- **Spec author**: Developer subagent
- **Target file**: `specs/done/spec-console-realtime/spec.md`
- **Do not create**: plan.md for implementation (that comes after spec)

## Verification

- Spec <800 lines
- Spec has ≤10 tasks if task list included
- All decision-log refs cited
- Console proxy design resolved (URL-based chosen)
- Real-time mechanism decided (WS vs SSE)
