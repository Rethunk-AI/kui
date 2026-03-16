# Console & Real-time Spec

## Scope

This spec defines the console and real-time behavior for MVP only, consistent with the PRD and decision log.

- MVP console paths are **noVNC primary and xterm.js fallback**. SPICE is deferred to v2.
  - decision-log: `§2` (Console protocol: noVNC + xterm.js), `§3` (order of preference)
- Console traffic is always proxied through KUI; no direct browser-to-libvirt connections.
  - decision-log: `§0` and `§2` (A12, Console proxy), `§4` (Console proxy)
- Real-time events cover VM lifecycle state and host reachability only in v1.
  - decision-log: `§2` (Real-time updates), `§4` (Real-time mechanism)
- Greenfield-only behavior: no migration paths, versioned upgrade flows, or compatibility modes are introduced.
  - decision-log: `§0` (A10 No migration path needed — greenfield only)
- Endpoint model aligns with the architecture target: Browser ↔ Web UI/API over HTTP/WebSocket/SSE, API ↔ Libvirt Connector.
  - prd/architecture `§1`, `§2`
- Open assumption from PRD is resolved: use URL-based proxy paths such as `wss://kui/...` rather than opaque hostnames.
  - prd.md `§2` (Console proxy design)

## noVNC Proxy

### 2.1 URL-based route

- Console endpoint: `wss://{kui-host}/api/hosts/{host_id}/vms/{vm_id}/vnc`
- Path segments encode both host and VM identifiers for routing, authz checks, and logging context.
- Optional short-lived token may be accepted as query/handshake metadata only when browser transport constraints require it; JWT remains the primary authentication mechanism.

### 2.2 Upgrade and routing behavior

1. Browser connects to `/api/hosts/{host_id}/vms/{vm_id}/vnc`.
2. KUI upgrades to WebSocket and resolves:
   - host by `host_id`
   - VM by `vm_id`
3. KUI validates session and authorization before opening any backend socket.
4. KUI obtains VNC target endpoint from domain metadata:
   - explicit per-VM console preference from `vm_metadata` first (when present),
   - otherwise parse libvirt domain graphics metadata via domain XML.
5. KUI creates a backend stream and proxies bytes bidirectionally until either side closes.

This sequence follows the PRD console preference ordering and remote connectivity constraints in the decision log.
- decision-log: `§2` (Console protocol, Console proxy), `§3` (noVNC → xterm.js order), `§4` (Console selection, Console proxy)

### 2.3 Local and remote hosts

- Local host:
  - API opens libvirt connection to local URI as configured (e.g., `qemu:///system`) and maps to the local VNC listener.
- Remote host:
  - KUI uses SSH-based libvirt connectivity strategy and still forwards console only through its own process:
    - local tunnel/forward to `127.0.0.1:{vnc_port}`
    - then proxied stream to browser WebSocket
  - This is required by remote-console assumptions in PRD architecture and console proxy decisions.
  - architecture: local + remote support in deployment topology.
- WebSocket proxy failure behavior:
  - `401/403` for authz failures
  - `404` if host or VM is not found
  - `502` when backend VNC stream is unavailable or VM not running

### 2.4 Error and close behavior

- On backend connection errors, KUI sends a close frame with reason and closes both directions to avoid hanging browser sessions.
- Failed or dropped host connections are surfaced as `host.offline` events (also persisted to runtime alerts where present).

## xterm.js

### 3.1 Usage conditions

- Use noVNC first and xterm.js only as fallback.
  - decision-log: `§3` (noVNC then xterm.js fallback), `§2` (Console protocol)
- Activate xterm.js when:
  - domain does not expose usable VNC graphics, or
  - selected/declared console is serial-only.

### 3.2 URL design

- Serial endpoint: `wss://{kui-host}/api/hosts/{host_id}/vms/{vm_id}/serial`
- Same host-vm addressing as noVNC for predictable routing and consistent authorization checks.

### 3.3 Backend contract

- Backend opens serial stream from libvirt domain serial channel and streams raw bytes over WebSocket.
- Local and remote host handling follows the same host-connection strategy used by noVNC:
  - local: direct libvirt connector access
  - remote: SSH-mediated access via KUI, then WebSocket proxying
- Backend emits clean termination events when the domain channel closes or a user disconnects.

### 3.4 Frontend behavior

- Winbox.js console windows embed xterm.js with the same auth context model as noVNC.
- Control keys, resize, and reconnect behavior remain aligned with current console window manager behavior in the UI layer.

## Real-time

### 4.1 Decision

- **Recommended for spec:** Use **WebSocket for console streaming paths** and **SSE for status updates**.
  - This follows the explicit recommendation in the decision log (`§2`, `§4`) after evaluating single multiplexed WebSocket complexity.
- Decision rationale:
  - Console stream is bidirectional and low-latency, requiring dedicated WebSocket.
  - Status streams are one-way and can be efficiently consumed via SSE with simpler backpressure semantics for MVP scale.

### 4.2 Status endpoint

- SSE endpoint: `GET /api/events` under the API host. Aligns with `specs/done/spec-frontend-build/spec.md` §5.
- Event stream sends only canonical v1 events:
  - `vm.state_changed`
  - `host.online`
  - `host.offline`
- Clients may reconnect with `Last-Event-ID`; missed events during transient disconnects are acceptable in v1 given the small concurrent-user target.
- status stream is read-only to clients and does not carry control commands.

### 4.3 Delivery guarantees

- At least once delivery is best-effort with short-lived stream loss tolerated for MVP.
- UI should reconcile by fetching current VM list and host status when connecting/reconnecting.
- This behavior matches single-tenant MVP scale and existing real-time assumptions.

## Event Types

### 5.1 vm.state_changed

- Description: VM transitions for lifecycle updates visible in VM list and console windows.
- Canonical name: `vm.state_changed`
- Payload:
  - `host_id` (string)
  - `vm_id` (string)
  - `state` (e.g., `running`, `paused`, `shut off`)
- Origin: domain state update from libvirt events or fallback polling.
- Source references:
  - decision-log: `§2` (Real-time updates), `§4` (Real-time mechanism)

### 5.2 host.offline

- Description: Host connectivity loss / unavailable for libvirt calls.
- Canonical name: `host.offline`
- Payload:
  - `host_id` (string)
  - `reason` (optional string)
- Origin: libvirt connection failure, SSH transport failure, or stream setup failure.
- Source references:
  - decision-log: `§2` (Real-time updates), `§4` (Real-time mechanism)

### 5.3 host.online

- Description: Previously offline or unreachable host is back online.
- Canonical name: `host.online`
- Payload:
  - `host_id` (string)
- Origin: successful libvirt access recovery.
- Source references:
  - decision-log: `§2` (Real-time updates), `§4` (Real-time mechanism)

### 5.4 Out of v1 scope

- `clone_progress` and `create_progress` are explicitly v2 features and are not included in v1 schema.
- Any future event payload additions must remain behind explicit v2 spec work.

## Authz

- All console and real-time endpoints require valid JWT session token as defined by session assumptions.
  - decision-log: `§0` A12/A13, `§2` session.
- Authorization checks:
  - VM endpoints require access to that VM/host context.
  - Host endpoints require host access.
  - MVP may assume single admin access and still enforce explicit checks at each endpoint.
- No direct browser exposure of libvirt credentials or host SSH details.
- Failed checks return `401` (missing/invalid session) or `403` (insufficient scope).

## Out of Scope

- SPICE console path and Open365 integration (v2)
  - decision-log: `§2`, `§3`
- Clone/create progress streaming in real-time (v2)
  - decision-log: `§2`, `§4`
- API-level RBAC tiers, multi-role policy, and non-admin role modeling
  - decision-log: `§2` (single admin only in MVP)
- Offline migrations, compatibility flags, fallback implementation modes
- any behavior intended for post-MVP multi-tenant, v3 operations, or migration tooling
