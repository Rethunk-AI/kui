# Contributing to KUI

Thank you for your interest in contributing. This document covers the development setup, coding standards, commit conventions, and PR process.

## Development Setup

### Prerequisites

- Go 1.22+ (CGO required for libvirt bindings)
- Node.js + Corepack (for the frontend)
- libvirt and libvirt-dev packages (for production builds; not required for CI builds)
- Make

### Clone and build

```bash
git clone https://github.com/Rethunk-AI/kui.git
cd kui
corepack enable         # activates Yarn 4 via Corepack (one-time)
make all                # frontend + backend + tests + vet
```

### Build without libvirt (CI / no KVM)

```bash
make build BUILD_TAGS=
make test BUILD_TAGS=
```

### Frontend development

```bash
cd web
corepack yarn install
corepack yarn run build       # production build
corepack yarn run dev         # dev server with HMR (set KUI_WEB_DIR to web/dist)
corepack yarn run test:coverage
```

---

## Project Structure

```
kui/
├── cmd/kui/           Entry point (main.go, routes.go, serial.go, vnc.go)
├── internal/          Core Go packages (config, db, routes, libvirtconn, ...)
├── web/               TypeScript/React frontend (Vite, Tailwind, shadcn/ui)
├── docs/              Operator and user documentation
├── specs/             Spec-Driven Development artifacts
│   ├── active/        In-progress specs
│   └── done/          Completed and verified specs
├── deploy/systemd/    systemd unit file
└── Makefile           Build automation
```

---

## Development Workflow

Non-trivial features follow the Spec-Driven Development (SDD) pattern:

1. Check `specs/active/` for an existing plan before starting.
2. Write a plan in `specs/active/[task-id]/plan.md`.
3. Implement against the plan.
4. Verify the implementation meets the spec (run `make all`).
5. Move the spec to `specs/done/` after verification.

Trivial changes (typo fixes, single-line config changes) can skip the plan step.

---

## Coding Guidelines

### Go

- Run `go fmt ./...` and `go vet ./...` before every commit. `make vet` covers the full tree.
- Use `slog` (log/slog) for structured logging. Log at handler/main boundaries only.
- Return errors; never swallow them.
- Add tests for new functionality. Use the libvirt test driver (`test:///default`) for unit tests — no mocks, no real KVM required.
- Document public functions and types.
- One concern per package. Keep `main.go` focused.
- No stub implementations. Never use `return nil` placeholders or no-op handlers. Use explicit `// TODO:` comments for deferred work.

### Frontend (TypeScript/React)

- Follow existing component patterns (shadcn/ui, Tailwind utility classes).
- Run accessibility checks: `make web-a11y`.
- Run `corepack yarn run test:coverage` for frontend tests.

### General

- **Greenfield project:** No migration paths, no backward-compatibility shims.
- **Security gate:** If your change touches auth, DB schema, VM lifecycle, or libvirt integration, flag it explicitly in your PR description.

---

## Commit Conventions

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short summary>

[optional body — explain WHY, not WHAT]
```

**Types:** `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf`, `ci`

**Scopes** (examples): `auth`, `config`, `db`, `routes`, `libvirt`, `frontend`, `provision`, `template`, `deploy`

**Examples:**

```
feat(routes): add VM clone endpoint with host selection
fix(config): reject jwt_secret shorter than 32 bytes at startup
docs(admin-guide): document --prefix precedence over KUI_PREFIX
chore(deps): update go-chi to v5.2.5
```

- Subject line: imperative mood, no trailing period, ≤72 characters.
- Body: explain motivation (why the change is needed), not a file list.
- One logical unit per commit.

---

## Pull Request Process

1. **Branch** from `main`. Use a descriptive branch name (e.g. `feat/vm-clone-host-select`, `fix/config-jwt-validation`).
2. **Make your changes**, following the coding guidelines above.
3. **Verify:**
   ```bash
   make all          # build + test + vet (with libvirt)
   # or, without libvirt:
   make build BUILD_TAGS= && make test BUILD_TAGS=
   ```
4. **Open a PR** against `main` with:
   - A clear title following commit conventions.
   - A description explaining the change and its motivation.
   - Reference to any related spec in `specs/done/` or `specs/active/`.
   - Security flag if the change touches auth, DB, or libvirt.
5. **Address review comments** — keep the history clean by amending or adding fixup commits as appropriate.
6. PRs are squash-merged unless the commit history is intentionally preserved.

---

## Reporting Issues

- **Bug:** Provide steps to reproduce, expected behavior, actual behavior, and KUI version or commit SHA.
- **Feature request:** Describe the use case and desired behavior.
- **Security issue:** See [SECURITY.md](SECURITY.md) for responsible disclosure. Do not open a public issue.

---

**Last updated:** 2026-04-27
