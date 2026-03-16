# KUI Software Stack

Chosen libraries and tooling. Decisions: [decision-log.md](decision-log.md). Architecture: [architecture.md](architecture.md).

---

## §1 Backend

| Layer | Choice | Notes |
|-------|--------|-------|
| Language | Go | See [decision-log.md](decision-log.md) §2 (Backend stack) |
| Libvirt | `libvirt.org/go/libvirt` (CGo) | Official bindings; requires libvirt-dev at build |
| XML | `libvirt.org/libvirt-go-xml` | Domain/network/storage pool structs |
| HTTP router | `github.com/go-chi/chi/v5` | Minimal, no magic middleware |
| Logging | `log/slog` | Structured, stdlib |

---

## §2 Database

| Item | Choice |
|------|--------|
| Engine | SQLite + Git |
| SQLite | users, preferences, vm_metadata |
| Git | templates (full audit chain), audit (diffs per entity). Path configurable; default /var/lib/kui/ |

---

## §3 Frontend & Console

| Item | Choice |
|------|--------|
| Canvas / layout | Winbox.js — draggable VM console windows |
| Console protocol | MVP: noVNC + xterm.js; SPICE in v2. See [decision-log.md](decision-log.md) §3 |
| Real-time | Prefer single WebSocket for status + console; else WebSocket for console, SSE for status |

Framework deferred to spec. xyflow deferred for future topology/infrastructure maps. See [decision-log.md](decision-log.md) §2 (Frontend, VM list UI) and §3 (Console protocol).

---

## §4 Config & Tooling

| Item | Choice |
|------|--------|
| Config format | YAML |
| Path resolution | Default /etc/kui/config.yaml; then `--config` or `KUI_CONFIG` |
| CLI | stdlib `flag` |
| Testing | Libvirt test driver `test:///default` (no mocks) |
