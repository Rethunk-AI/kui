# KUI Admin Guide

Setup, deployment, and configuration for KUI administrators. For product overview and architecture, see [PRD](prd.md) and [Architecture](prd/architecture.md).

---

## System Requirements

| Requirement | Notes |
|-------------|-------|
| **OS** | Linux with libvirt (KVM) |
| **libvirt** | `libvirt` and `libvirt-dev` packages; `libvirtd` running |
| **Remote hosts** | `nc` (netcat) installed; SSH key auth to `qemu+ssh://` |
| **Storage** | `/var/lib/kui` writable (DB, templates, audit); `/etc/kui` for config |
| **Build** | Go 1.22+; CGO for libvirt bindings (`-tags libvirt`) |

See [spec-libvirt-connector](../specs/done/spec-libvirt-connector/spec.md) and [decision-log §1](prd/decision-log.md) for remote libvirt and test driver.

---

## First-Run Setup

When config is missing, KUI runs in **setup mode** and serves a setup wizard. No auth is required until setup completes.

1. **Start KUI** (see [Deployment](#deployment)).
2. **Open the UI** in a browser. You are redirected to the setup flow.
3. **Configure:**
   - **Admin account** — username and password (stored in SQLite).
   - **Hosts** — at least one libvirt host. For each host:
     - `id` — short identifier (e.g. `local`, `prod`)
     - `uri` — `qemu:///system` (local) or `qemu+ssh://user@host/system?keyfile=/path/to/key`
     - `keyfile` — path to SSH private key for remote hosts (null for local).
4. **Default host** — select which host is used by default for VM operations.
5. **Complete setup.** KUI writes config to disk. Restart KUI (e.g. `systemctl restart kui`) to load the new config; then log in.

Setup is idempotent: once config exists, the wizard is unavailable. See [api-auth spec](../specs/done/api-auth/spec.md) and [schema-storage](../specs/done/schema-storage/spec.md).

---

## Config Reference

Config is written by the setup wizard. Full YAML structure and env overrides are in [schema-storage spec](../specs/done/schema-storage/spec.md) §2.6.

**Required:**
- `hosts` — list of libvirt connection targets.
- `jwt_secret` — min 32 bytes; set by setup wizard.

**Common optional:**
- `vm_defaults` — CPU, RAM, network for VM create (default: 2 vCPU, 2048 MB, `default` network).
- `default_host` — default host ID.
- `vm_lifecycle.graceful_stop_timeout` — timeout before force stop (default: 30s).

---

## Deployment

| Topic | Document |
|-------|----------|
| **systemd** | [deploy/systemd/README.md](../deploy/systemd/README.md) — unit file, install, runtime dirs |
| **TLS & reverse proxy** | [deployment.md](deployment.md) — HTTP, direct TLS, nginx/Caddy, WebSocket/SSE |

KUI listens on `:8080` by default. Behind a reverse proxy, configure WebSocket and SSE passthrough per [deployment.md](deployment.md).

---

## Build and Run

```bash
# With libvirt (default, production)
make all

# Without libvirt (CI, no KVM)
make build BUILD_TAGS=
make test BUILD_TAGS=
```

Frontend: `corepack enable` (once), then `cd web && corepack yarn install && corepack yarn run build`. Uses Yarn 4 via Corepack (`packageManager` in package.json). Embedded in binary via `//go:embed`; or set `KUI_WEB_DIR` to serve from disk. See [Makefile](../Makefile).

---

## Troubleshooting

| Issue | Check |
|-------|-------|
| Setup wizard not shown | Config exists at `KUI_CONFIG` (default `/etc/kui/config.yaml`). Remove or rename to re-run setup. |
| Host unreachable | Verify `libvirtd` on remote host; SSH key in `authorized_keys`; `nc` installed. See [spec-libvirt-connector](../specs/done/spec-libvirt-connector/spec.md). |
| Console (VNC/serial) fails | Local hosts only in MVP; remote requires KUI on same host as libvirt or tunnel. See [deployment.md](deployment.md) for WebSocket proxy setup. |
| WebSocket/SSE not working | Reverse proxy must forward `Upgrade` and `Connection` headers; disable buffering for SSE. See [deployment.md](deployment.md). |
