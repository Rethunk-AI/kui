# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| `main` | Yes |

Only the `main` branch receives security fixes. There are no versioned release branches at this time.

## Scope

This policy covers vulnerabilities in the KUI application itself:

- Authentication and session management (JWT, login, setup wizard)
- Authorization bypass (accessing VMs or hosts without valid session)
- VM lifecycle operations (start, stop, delete, clone) performed without authorization
- Console access (VNC/serial WebSocket) without authorization
- libvirt integration (command injection, unsafe URI handling)
- SQLite data exposure (path traversal, config read)
- File path traversal via `--prefix` / `KUI_PREFIX` or config fields
- Server-Sent Events (SSE) data leakage to unauthenticated clients
- Dependency vulnerabilities in Go modules or frontend npm packages

**Out of scope:**

- Vulnerabilities in libvirt, QEMU, or the host OS
- Issues requiring physical access to the host machine
- Denial-of-service attacks that require authenticated access

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Report security issues privately using one of the following channels:

- **GitHub Security Advisories:** Repository → Security tab → Advisories → "Report a vulnerability"
- **Email:** security@rethunk.ai (PGP not required; plain text is fine)

Include as much detail as possible:

- A description of the vulnerability and its impact
- Steps to reproduce (proof-of-concept if available)
- Affected component (e.g. auth middleware, libvirt connector, SSE endpoint)
- KUI version or commit SHA

## Response Timeline

| Milestone | Target |
|-----------|--------|
| Acknowledgement | Within 48 hours of report |
| Initial assessment | Within 7 days |
| Fix or mitigation | Depends on severity; critical issues prioritized |
| Public disclosure | Coordinated with reporter after fix is available |

We follow responsible disclosure. We will work with you to understand and address the issue before any public disclosure.
