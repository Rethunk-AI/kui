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

1. Build: `make all` (builds with `-tags libvirt` by default).
2. Run: `./bin/kui` (default config `/etc/kui/config.yaml`). When config is missing, KUI runs the setup wizard.
3. Open the UI in a browser; complete setup (admin account, hosts).
4. Log in and manage VMs. See [Admin Guide](docs/admin-guide.md) and [User Guide](docs/user-guide.md).

### Runtime prefix (`--prefix` / `KUI_PREFIX`)

KUI can treat a single directory as a **virtual filesystem root** for every local path it opens (config, SQLite, git data, TLS PEMs, `KUI_WEB_DIR`, SSH key paths in config, default pool directories, and related overrides). This is a **chroot-style mental model** only: the process is **not** placed in a literal `chroot(2)` jail; paths are rewritten before `open(2)`.

With a non-empty prefix, a logical path such as `/etc/kui/config.yaml` is read from `{prefix}/etc/kui/config.yaml` on disk (leading `/` does **not** mean the host root).

Minimal example:

```bash
kui --prefix "$HOME/kui-root" --config /etc/kui/config.yaml --listen :8080
```

Here the config file is loaded from `$HOME/kui-root/etc/kui/config.yaml`. Other absolute-style paths in config and env behave the same way. For a full layout, precedence (`--prefix` vs `KUI_PREFIX` vs optional YAML `runtime.prefix`), TLS/static assets, libvirt caveats, and a manual smoke checklist, see [Contained / non-root mode in the Admin Guide](docs/admin-guide.md#contained-non-root-mode-prefix).

## Build

Libvirt bindings require CGO and `libvirt-dev` headers. If those headers are unavailable, build and test without libvirt:

```bash
make build BUILD_TAGS=
make test BUILD_TAGS=
make vet BUILD_TAGS=
```

## Coverage

**Go**

```bash
make coverage          # Generate coverage.out
make coverage-report   # Generate coverage.out + coverage.html + text summary
```

**Web**

```bash
make web-coverage      # Run Vitest with coverage (output in web/coverage/)
```

Or from the web directory: `yarn test:coverage`.

CI uploads Go coverage (`coverage.out`, `coverage.html`) and web coverage (`web/coverage/`) as artifacts.

## Contributing

Open an issue or merge request. See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## License

[MIT](LICENSE)
