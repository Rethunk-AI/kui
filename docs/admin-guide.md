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

<a id="contained-non-root-mode-prefix"></a>

## Contained / non-root mode (`--prefix`)

KUI supports a **runtime prefix**: `--prefix` on the CLI or the `KUI_PREFIX` environment variable, and optionally `runtime.prefix` in YAML. When the effective prefix is non-empty, the process resolves **local** filesystem paths as if that directory were `/` (chroot-style **re-rooting**, not a real `chroot(2)`). A path like `/var/lib/kui/kui.db` opens as `{prefix}/var/lib/kui/kui.db`. This is useful for non-root installs, relocatable trees, and tests without writing under the real FHS.

Paths that are **not** re-rooted include libvirt URIs, pool **names** (non-path strings), JWT/session settings, and host discovery paths used by KVM checks (for example `/dev/kvm`, `/etc/os-release`).

### Creating a writable prefix tree

Create directories that match the logical paths your config uses (defaults assume FHS-style locations). For a minimal layout that mirrors common defaults:

```bash
PREFIX="$HOME/kui-run"   # or: PREFIX="$(mktemp -d)"
mkdir -p "$PREFIX/etc/kui" "$PREFIX/var/lib/kui"
# If you rely on default pool directory heuristics under prefix:
mkdir -p "$PREFIX/var/lib/libvirt/images"
```

Place `config.yaml` under `{prefix}/etc/kui/` when using logical path `/etc/kui/config.yaml`, or adjust `mkdir` to match your YAML.

### Sample config (absolute-style paths under prefix)

Use normal absolute-looking strings in YAML; with a prefix set, they refer to **locations under the prefix**, not the host root:

```yaml
# Optional (lowest precedence vs --prefix / KUI_PREFIX); omit in most deployments:
# runtime:
#   prefix: /opt/kui-run

db:
  path: /var/lib/kui/kui.db

git:
  path: /var/lib/kui

hosts:
  - id: local
    uri: qemu:///system
    keyfile: null
  # Remote example: key path is also under prefix when prefix is set
  # - id: remote
  #   uri: qemu+ssh://user@host/system
  #   keyfile: /home/kui/.ssh/id_ed25519

jwt_secret: "<set-via-setup-or-generate>"
```

On disk with `PREFIX=/opt/kui-run`, the DB file above is opened at `/opt/kui-run/var/lib/kui/kui.db`.

### Prefix precedence (highest first)

1. **`--prefix`** — if you pass this flag, it wins.
2. **`KUI_PREFIX`** — used when the `--prefix` flag was not set on the command line.
3. **`runtime.prefix` in YAML** — used only when neither `--prefix` nor `KUI_PREFIX` supplies a non-empty value after trimming. It applies after the config file is loaded, so it does not change which path was used to **find** that file; use the flag or env when you want the config file itself to be read from under a prefix tree.

The bootstrap prefix (`--prefix` / `KUI_PREFIX`) is also used to resolve the **config path** before reading (for example `--prefix /tmp/r --config /etc/kui/config.yaml` reads `/tmp/r/etc/kui/config.yaml`).

### TLS PEMs and `KUI_WEB_DIR`

When a non-empty effective prefix is in use:

- **`--tls-cert`** and **`--tls-key`** are resolved under the prefix the same way as other filesystem paths (for example logical `/etc/kui/tls/cert.pem` → `{prefix}/etc/kui/tls/cert.pem`).
- **`KUI_WEB_DIR`**, when set, is resolved under the effective prefix used for static files (bootstrap prefix from flag/env, or `runtime.prefix` from YAML when the bootstrap prefix is empty).

### When not to use a prefix

Use **empty prefix** (omit `--prefix`, `KUI_PREFIX`, and YAML `runtime.prefix`) when:

- KUI should use **real host paths** for SQLite, git data, TLS material, and pool directories—typical **system libvirt** deployments on FHS (`/var/lib/libvirt/images`, `/var/lib/kui`, `/etc/kui`, …).
- You expect logical `/var/...` paths in config to mean the **actual** host filesystem. With a prefix set, default pool path probes (for example under `/var/lib/libvirt/images`) are evaluated under the prefix; **libvirt on the host** still uses host paths unless you mirror layout under the prefix, use bind mounts, or align pool configuration explicitly.

### Manual smoke checklist (after `go build`)

Optional steps for humans validating a local binary; **automated tests are canonical**—run `go test ./...` or `make all` for project verification.

1. `export PREFIX="$(mktemp -d)"` (or choose a fixed directory under `$HOME`).
2. `mkdir -p "$PREFIX/etc/kui" "$PREFIX/var/lib/kui"` and any other logical paths your config references (including default pool dirs if you exercise provisioning defaults).
3. Install or author a minimal `config.yaml` under `$PREFIX/etc/kui/` using absolute-style paths as in the sample above; confirm files you care about exist only under `$PREFIX/...`.
4. From the repo: `go build -o bin/kui ./cmd/kui`, then run (example) `./bin/kui --prefix "$PREFIX" --config /etc/kui/config.yaml --listen 127.0.0.1:0` (adjust `--listen` as needed).

**Note:** Full VM lifecycle still depends on **host** libvirt and KVM; a contained smoke run may only confirm HTTP listener, config load, and DB open. Libvirt URIs and host device access are unchanged by prefix.

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

## Host Provisioning

When a libvirt host has no storage pools or no networks, KUI can provision them. This is needed when:

- **Setup wizard** — validate-host returns "no storage pools" or "no networks".
- **Create VM** — the selected host has no pools or networks in the dropdown.

### Provisioning flow

1. **Audit** — KUI shows what would be created (pool path, network name/subnet).
2. **Review** — User confirms the proposed configuration.
3. **Execute** — KUI creates the pool and/or network.

### Default paths

| Condition | Pool path |
|-----------|-----------|
| `/var/lib/libvirt/images` exists and is non-empty | Use existing path |
| `/var/lib/libvirt/images` missing or empty | Propose `/var/lib/kui/images`; create dir before pool define |

### Permissions

The KUI process must be able to create `/var/lib/kui` and subdirs (e.g. `/var/lib/kui/images`). Typically run as `root` or a dedicated `kui` user with appropriate permissions in systemd.

### Local-only (MVP)

Provisioning is supported only for **local hosts** (`qemu:///system`). Remote hosts (`qemu+ssh://`) return 400 with "remote host provisioning not supported in this version". Remote provisioning is planned for a future release.

---

## Troubleshooting

| Issue | Check |
|-------|-------|
| Setup wizard not shown | Config exists at `KUI_CONFIG` (default `/etc/kui/config.yaml`). Remove or rename to re-run setup. |
| Provision fails (permission denied) | KUI must create `/var/lib/kui/images` when libvirt default path is empty. Ensure KUI runs as root or has write access. |
| Host unreachable | Verify `libvirtd` on remote host; SSH key in `authorized_keys`; `nc` installed. See [spec-libvirt-connector](../specs/done/spec-libvirt-connector/spec.md). |
| Console (VNC/serial) fails | Local hosts only in MVP; remote requires KUI on same host as libvirt or tunnel. See [deployment.md](deployment.md) for WebSocket proxy setup. |
| WebSocket/SSE not working | Reverse proxy must forward `Upgrade` and `Connection` headers; disable buffering for SSE. See [deployment.md](deployment.md). |
