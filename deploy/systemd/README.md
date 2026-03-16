# Systemd Deployment

KUI is deployed via systemd as the primary mechanism. For TLS and reverse-proxy setup (nginx/Caddy), see [docs/deployment.md](../docs/deployment.md).

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
