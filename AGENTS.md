# KUI — AI/LLM Instructions

**Canonical configuration:** `.cursor/rules/`, `.cursor/agents/`, `.cursor/skills/`. Edit only those.

**Do not edit:** `.claude/`, `.codex/` — they are symlinks to `.cursor/`. See [docs/project-config-canonical.md](docs/project-config-canonical.md) for the full mapping.

---

## Project Overview

KUI is a web-based KVM User Interface. It connects to one or more libvirt hosts (local or remote via `qemu+ssh://`) and provides browser-based VM lifecycle management, console access (VNC + serial), real-time event streaming (SSE), storage provisioning, and an audit log.

Primary languages: Go (backend) and TypeScript/React (frontend, embedded in binary).

---

## Repository Layout

```
kui/
├── cmd/kui/           Main entry point (main.go, routes.go, serial.go, vnc.go)
├── internal/
│   ├── audit/         Audit log (write, query)
│   ├── broadcaster/   SSE event fan-out to connected clients
│   ├── config/        YAML config load, validation, env overrides, --prefix support
│   ├── db/            SQLite schema and queries (database/sql)
│   ├── domainxml/     libvirt domain XML parsing and construction
│   ├── eventsource/   Server-Sent Events helpers
│   ├── git/           Git-backed template management
│   ├── kvmcheck/      Host KVM capability detection
│   ├── libvirtconn/   libvirt connection management (local + SSH tunnel)
│   ├── middleware/     HTTP middleware (JWT auth, logging)
│   ├── prefix/        Runtime prefix path re-rooting logic
│   ├── provision/     Storage pool and network provisioning
│   ├── routes/        HTTP route handlers (REST API)
│   ├── sshtunnel/     SSH tunnel for remote libvirt connections
│   └── template/      VM template CRUD
├── web/               TypeScript/React frontend (Vite, Tailwind, shadcn/ui)
│   └── embed.go       Embeds dist/ into the binary (//go:embed)
├── docs/              Operator and user documentation
│   ├── admin-guide.md
│   ├── deployment.md
│   ├── user-guide.md
│   └── prd/           PRD, architecture, decision log, research
├── specs/
│   ├── active/        In-progress specifications (plan.md per task)
│   └── done/          Completed and verified specifications
├── deploy/systemd/    systemd unit file and install instructions
├── Makefile           Build automation
└── go.mod             Module: github.com/kui/kui, Go 1.25
```

---

## Key Dependencies

| Dependency | Role |
|------------|------|
| `libvirt.org/go/libvirt` | libvirt C bindings (CGO; `-tags libvirt` required for production) |
| `libvirt.org/go/libvirtxml` | Domain XML serialisation |
| `github.com/go-chi/chi/v5` | HTTP router |
| `github.com/golang-jwt/jwt/v5` | JWT session tokens |
| `github.com/mattn/go-sqlite3` | SQLite driver (CGO) |
| `github.com/gorilla/websocket` | WebSocket (VNC, serial console) |
| `github.com/go-git/go-git/v5` | Git-backed template storage |
| `golang.org/x/crypto` | bcrypt password hashing |
| `gopkg.in/yaml.v3` | Config file parsing |

Frontend: React + Vite + Tailwind CSS + shadcn/ui components, served embedded in the binary or via `KUI_WEB_DIR` (disk override).

---

## Build and Verification

```bash
# Production build (requires libvirt dev headers)
make all              # frontend + backend + tests + vet

# Backend only
go build -tags libvirt -o bin/ ./cmd/...

# Without libvirt (CI, no KVM)
make build BUILD_TAGS=
make test BUILD_TAGS=

# Frontend only
cd web && corepack yarn install && corepack yarn run build

# Coverage
make coverage         # produces coverage.out
make coverage-report  # produces coverage.html

# Accessibility checks
make web-a11y
```

**Libvirt test driver:** Unit tests use `test:///default` (in-memory fake hypervisor). No real KVM or libvirt daemon is required in CI. Pass `-tags libvirt` — the test driver is part of the libvirt library.

**Do not run compiled binaries.** Verification uses `go test`, `go vet`, and linters only. Running `./bin/kui` is disallowed for automated verification.

---

## Development Workflow (SDD)

All non-trivial work follows the Spec-Driven Development (SDD) pattern:

1. **Check** `specs/active/` for an existing plan before starting.
2. **Plan** — invoke the `planner` subagent; it writes `specs/active/[task-id]/plan.md`.
3. **Implement** — invoke the `developer` subagent against the plan.
4. **Verify** — invoke the `verifier` subagent; confirms implementation meets spec.
5. **Complete** — move spec to `specs/done/` only after verifier confirms.

Trivial changes (single-line fixes, typos) may skip the planner.

Completed specs are indexed at `specs/done/`. Run `make specs-list` for a sorted list.

---

## Architectural Decisions

- **SQLite** for persistence (single file, 1–2 concurrent users). Path from config; env `KUI_DB_PATH` for Docker override.
- **chi router** for HTTP — minimal, no magic middleware.
- **CGO required** for production: libvirt and sqlite3 both use CGO. For no-CGO builds, omit `-tags libvirt` (uses test driver stubs).
- **Frontend embedded** in the binary via `//go:embed`. Override with `KUI_WEB_DIR` to serve from disk (useful during frontend development).
- **JWT cookies** for session auth. Secret set during setup wizard; minimum 32 bytes.
- **SSE** (`/api/events`) for real-time VM status updates. Reverse proxies must disable response buffering for this endpoint.
- **WebSocket** for VNC (`/api/hosts/{id}/vms/{uuid}/vnc`) and serial (`/api/hosts/{id}/vms/{uuid}/serial`) consoles.
- **Runtime prefix** (`--prefix` / `KUI_PREFIX`) re-roots all local filesystem paths without `chroot(2)`. Libvirt URIs are not affected.
- **Git-backed templates** stored in `internal/git` for reproducible VM definitions.
- **Greenfield:** No migration shims, no backward-compatibility code. Implement target design directly.

---

## Go Coding Standards

- Follow `.cursor/rules/go-standards.mdc`.
- Use `slog` (log/slog) for structured logging. Log at boundaries (handlers, main). Avoid deep-helper logging.
- Return errors; never swallow them.
- Use `slog.Error` + `os.Exit(1)` for startup failures.
- Run `go fmt` and `go vet` before committing. `make vet` covers the full tree.
- Document public APIs. Add tests for new functionality.
- One concern per package. Keep `main.go` focused; extract to packages when it grows.

---

## AI Constraints

- **Edit only canonical sources:** `.cursor/rules/`, `.cursor/agents/`, `.cursor/skills/`. The `.claude/` and `.codex/` directories are symlinks — edits there go nowhere useful.
- **No stub implementations:** Never use `return nil` placeholders, no-op handlers, or `// TODO: implement`. Implement functional behavior directly. Use explicit `// TODO:` comments for deferred work.
- **Security gate:** Invoke `security-auditor` whenever changes touch DB schema, VM lifecycle handlers, auth/authorization, or libvirt integration.
- **No binary execution:** Do not run `./bin/kui` or any compiled artifact for verification. Use `go test` and `go vet`.
- **Greenfield:** No migrations, no backfill steps, no backward-compatibility shims.
- **Verification evidence required:** Do not claim "done" without running verification commands and checking output in the current session.
- **Config precedence:** `--prefix` > `KUI_PREFIX` > empty. `--config` > `KUI_CONFIG` > default path. The YAML `runtime:` block is not read for prefix — use the flag or env only.

---

## Links

- Workflow: `.cursor/rules/workflow.mdc`
- Go standards: `.cursor/rules/go-standards.mdc`
- Greenfield rules: `.cursor/rules/greenfield.mdc`
- Canonical config: `.cursor/rules/canonical-config.mdc`
- Admin guide: [docs/admin-guide.md](docs/admin-guide.md)
- Deployment: [docs/deployment.md](docs/deployment.md)
