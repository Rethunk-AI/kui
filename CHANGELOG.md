# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] — 2026-04-27

Initial release of KUI — a web-based KVM User Interface for VM lifecycle management via libvirt.

### Added

- **Authentication** — JWT session cookies, bcrypt password hashing, setup wizard for first-run admin account creation (`api-auth`, `feat-setup-wizard-ui`)
- **VM lifecycle** — start, stop (graceful + force), pause, resume, delete, and clone operations across local and remote libvirt hosts (`spec-vm-lifecycle-create`)
- **VM create** — create VMs from existing storage volumes or new disks; configurable vCPU, RAM, and network (`spec-vm-lifecycle-create`)
- **Orphan management** — claim and bulk-claim libvirt domains not yet tracked by KUI (`feat-orphan-bulk`)
- **Console access** — VNC (noVNC WebSocket) and serial (xterm.js WebSocket) browser-based console (`spec-console-realtime`)
- **Real-time events** — Server-Sent Events (`/api/events`) for live VM status updates across all connected clients
- **Multi-host support** — connect to multiple libvirt hosts (local `qemu:///system` and remote `qemu+ssh://`); per-user host preference (`spec-libvirt-connector`)
- **Host provisioning** — create default storage pools and networks on local hosts that have none (`feat-host-provisioning`)
- **Template management** — git-backed VM template CRUD (`spec-template-management`)
- **Domain XML editing** — view and edit raw libvirt domain XML (`feat-domain-xml-edit`)
- **Audit log** — structured audit trail for all VM and administrative operations (`gap-audit`, `spec-audit-integration`)
- **Stuck VM detection** — detect and recover VMs stuck in transitional states (`feat-stuck-vm`)
- **Keyboard shortcuts** — browser UI keyboard navigation (`feat-keyboard-shortcuts`)
- **Runtime prefix** — `--prefix` / `KUI_PREFIX` for relocatable installs and non-root deployments (`spec-application-bootstrap`)
- **Direct TLS** — `--tls-cert` / `--tls-key` flags for KUI-terminated TLS
- **Reverse proxy support** — WebSocket and SSE passthrough documented; nginx and Caddy examples (`spec-ui-deployment`)
- **systemd deployment** — unit file, runtime directories, and install guide (`deploy/systemd/`)
- **Accessibility** — pa11y-ci checks integrated into the build (`feat-a11y`)
- **Frontend** — React + Vite + Tailwind CSS + shadcn/ui; embedded in binary via `//go:embed` (`spec-frontend-build`, `feat-shadcn-ui`)
- **SQLite persistence** — VM metadata, user accounts, and audit log stored in a single SQLite database (`schema-storage`)
- **Full test coverage** — libvirt test driver (`test:///default`) used in all unit tests; no real KVM required in CI (`coverage-100`)
- **CI** — GitHub Actions and GitLab CI pipelines

[0.1.0]: https://github.com/Rethunk-AI/kui/releases/tag/v0.1.0
