# KUI Deployment

This document covers TLS and reverse-proxy deployment for KUI. For systemd setup, see [deploy/systemd/README.md](../deploy/systemd/README.md). For first-run setup and config, see [admin-guide.md](admin-guide.md).

## Service modes

| Mode | Use case |
|------|----------|
| HTTP-only | Development, local testing |
| Direct TLS | KUI terminates TLS (optional) |
| Reverse proxy | Production: nginx/Caddy terminate TLS, proxy to KUI over HTTP |

## HTTP-only (development)

For local development, run KUI without TLS:

```bash
kui --config /etc/kui/config.yaml --listen :8080
```

Default listen address is `:8080`. Override with `--listen` or `KUI_LISTEN`.

## Direct TLS

KUI can terminate TLS when both `--tls-cert` and `--tls-key` are provided:

```bash
kui --config /etc/kui/config.yaml \
    --listen :8443 \
    --tls-cert /etc/kui/tls/cert.pem \
    --tls-key /etc/kui/tls/key.pem
```

- Both flags are required together; using only one returns an error.
- TLS is configured via flags only; environment variables are not supported.
- Use PEM-encoded certificate and private key files.

### systemd with direct TLS

```ini
[Service]
ExecStart=/usr/local/bin/kui --config /etc/kui/config.yaml \
    --listen 127.0.0.1:8443 \
    --tls-cert /etc/kui/tls/cert.pem \
    --tls-key /etc/kui/tls/key.pem
```

## Reverse proxy (production)

For production, terminate TLS at a reverse proxy (nginx or Caddy) and proxy to KUI over HTTP. This allows:

- Centralized certificate management (e.g. Let's Encrypt)
- Rate limiting, logging, and hardening at the proxy
- KUI listening on localhost only

### Requirements

The proxy **must** preserve:

1. **WebSocket upgrades** for console streaming:
   - `GET /api/hosts/{host_id}/vms/{uuid}/vnc` — noVNC WebSocket
   - `GET /api/hosts/{host_id}/vms/{uuid}/serial` — xterm.js serial WebSocket

2. **Server-Sent Events (SSE)** for real-time updates:
   - `GET /api/events` — long-lived stream (`Connection: keep-alive`, `Cache-Control: no-cache`, `Content-Type: text/event-stream`)

WebSocket upgrade headers (`Upgrade`, `Connection`, `Sec-WebSocket-Key`, `Sec-WebSocket-Version`, etc.) must be forwarded to KUI.

### nginx example

Place the `map` block in the `http {}` context (e.g. in `nginx.conf` or an included file):

```nginx
# In http {} block:
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

server {
    listen 443 ssl;
    server_name kui.example.com;

    ssl_certificate     /etc/ssl/certs/kui.example.com.crt;
    ssl_certificate_key /etc/ssl/private/kui.example.com.key;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support (VNC, serial console)
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 86400;
    }

    location /api/events {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE: keep connection open, disable buffering
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400;
        chunked_transfer_encoding off;
    }
}
```

KUI should listen on `127.0.0.1:8080` when behind this proxy:

```bash
kui --config /etc/kui/config.yaml --listen 127.0.0.1:8080
```

### Caddy example

```caddyfile
kui.example.com {
    tls /etc/caddy/certs/kui.example.com.crt /etc/caddy/certs/kui.example.com.key

    reverse_proxy http://127.0.0.1:8080 {
        header_up Host {host}
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}

        # WebSocket and SSE are handled automatically by Caddy's reverse_proxy
        # when Upgrade header is present; no extra config needed.
    }
}
```

Caddy automatically handles WebSocket upgrades and long-lived SSE connections. KUI should listen on `127.0.0.1:8080`.

**Proxy requirements:** Forward WebSocket upgrade headers (`Upgrade`, `Connection`, etc.) and avoid buffering SSE (`GET /api/events`). See examples above.
