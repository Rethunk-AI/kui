# KUI

Web-based KVM User Interface for users who prefer a UI over CLI. Connects to libvirt hosts for VM lifecycle, provisioning, and monitoring.

## Documentation

| Doc | Audience | Content |
|-----|----------|---------|
| [Admin Guide](docs/admin-guide.md) | Operators | System requirements, first-run setup, config, deployment, troubleshooting |
| [User Guide](docs/user-guide.md) | End users | Login, VM list, create/clone, lifecycle, console, templates, orphans |
| [Systemd deployment](deploy/systemd/README.md) | Operators | Unit file, install, runtime dirs |
| [TLS & reverse proxy](docs/deployment.md) | Operators | HTTP, direct TLS, nginx/Caddy, WebSocket/SSE |
| [PRD](docs/prd.md) | All | Product overview, references to decision-log, architecture, stack |

## Quick Start

1. Build: `make all` (or `go build -tags libvirt -o bin/ ./cmd/...`).
2. Run: `./bin/kui` (default config `/etc/kui/config.yaml`). When config is missing, KUI runs the setup wizard.
3. Open the UI in a browser; complete setup (admin account, hosts).
4. Log in and manage VMs. See [Admin Guide](docs/admin-guide.md) and [User Guide](docs/user-guide.md).

## Build

Libvirt bindings require CGO and `libvirt-dev` headers. If those headers are unavailable, build and test the project without libvirt-enabled files using:

```bash
go build ./...
go test ./...
go vet ./...
```

Run the libvirt connector integration test against `test:///default` with:

```bash
go test -tags libvirt ./internal/libvirtconn/...
```

## Contributing

Open an issue or merge request. See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## License

[MIT](LICENSE)
