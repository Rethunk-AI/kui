# Systemd Deployment

KUI is deployed via systemd as the primary mechanism. For TLS and reverse-proxy setup (nginx/Caddy), see [docs/deployment.md](../docs/deployment.md). For first-run setup and config, see [docs/admin-guide.md](../docs/admin-guide.md).

## Prerequisites

- `/var/lib/kui` must exist and be writable by the service user (DB, git templates, audit chain)
- Config at `/etc/kui/config.yaml` (or set `KUI_CONFIG` to override)
- KUI binary at `/usr/local/bin/kui` (or adjust `ExecStart` in the unit)

## Installation

1. Copy the unit file:

   ```bash
   sudo cp deploy/systemd/kui.service /etc/systemd/system/
   ```

2. If the binary is elsewhere (e.g. `/opt/kui/bin/kui`), edit the unit:

   ```bash
   sudo systemctl edit kui
   # Add:
   # [Service]
   # ExecStart=/opt/kui/bin/kui --config /etc/kui/config.yaml
   ```

3. Create runtime directory and config:

   ```bash
   sudo mkdir -p /var/lib/kui
   sudo chown kui:kui /var/lib/kui   # or your service user
   sudo mkdir -p /etc/kui
   # Create /etc/kui/config.yaml with hosts, jwt_secret, etc.
   sudo $EDITOR /etc/kui/config.yaml
   ```

4. Enable and start:

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable kui
   sudo systemctl start kui
   sudo systemctl status kui
   ```

## Service Contract

- **Type**: `simple` (will switch to `notify` when readiness signaling is implemented)
- **Restart**: `on-failure`
- **Config**: `--config /etc/kui/config.yaml` or `KUI_CONFIG` env var
- **Runtime data**: `/var/lib/kui/` for SQLite DB, git-backed templates, and audit chain

## Optional: contained layout with `--prefix`

The default unit assumes **real FHS paths** on the host (`/etc/kui`, `/var/lib/kui`). For a **relocatable tree** under a single directory, run KUI with `--prefix` (or set `KUI_PREFIX` in the unit). Logical paths stay the same in `ExecStart`; they are opened **under** the prefix on disk.

**Example:** If all state lives under `/opt/kui-run`, create `/opt/kui-run/etc/kui/config.yaml`, `/opt/kui-run/var/lib/kui`, and matching permissions for the service user. Then use a command equivalent to:

```ini
# Alternate (commented in kui.service): relocatable root
# ExecStart=/usr/local/bin/kui --prefix /opt/kui-run --config /etc/kui/config.yaml
```

That reads config from `/opt/kui-run/etc/kui/config.yaml` and places DB/git defaults under `/opt/kui-run/var/lib/kui/...` when config uses the usual absolute-style paths.

**`WorkingDirectory` vs `--prefix`:** `WorkingDirectory=` only sets the process cwd for relative paths in **other** tools and subprocess behavior; KUI’s prefix resolution does **not** use `WorkingDirectory` as the runtime root. Set `--prefix` when you want FHS-like paths to map under a relocatable directory. Full semantics and precedence (`--prefix`, `KUI_PREFIX`) are in [docs/admin-guide.md](../../docs/admin-guide.md#contained-non-root-mode-prefix).
