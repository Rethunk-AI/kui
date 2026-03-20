# feat-runtime-prefix — Specification

## Intent

Operators can run KUI with a single **`--prefix` / `KUI_PREFIX`** so that the process treats that directory as a **filesystem root** for every path KUI resolves to local files or directories. This matches a **chroot-style** mental model (not a literal `chroot(2)`): absolute-looking paths such as `/var/lib/kui/db.sqlite` are opened as `{prefix}/var/lib/kui/db.sqlite`.

**Primary goals:** non-root-friendly installs, reproducible **`t.TempDir()`**-style testing without writing under system FHS paths, and one clear knob instead of many `*_PREFIX` variables.

## Acceptance criteria

1. When prefix is **non-empty**, after `filepath.Clean`, **every application-managed local path** (defaults, YAML, env overrides for `db.path`, `git.path`, TLS PEMs, `KUI_WEB_DIR`, SSH keyfiles, config path candidates, provision default pool **directory** heuristics, `KUI_TEST_PROVISION_POOL_PATH`, temp files beside config) is resolved **under** `prefix` using the rule: strip a leading separator from the cleaned path and `filepath.Join(prefix, …)` (relative paths join the same way; no “absolute bypasses prefix”).
2. When prefix is **empty**, behavior matches **pre-prefix** semantics: relative paths remain CWD-relative where they are today; absolute paths are used as-is.
3. **Out of prefix (unchanged):** libvirt URIs, pool **names** (non-path strings), JWT/session semantics, and **host discovery** paths in `internal/kvmcheck` (e.g. `/dev/kvm`, `/etc/os-release`) — these are not “KUI install tree” paths.
4. **Documentation** states the chroot analogy, the testing story, and that **production** use with real system libvirt may require a prefix tree that mirrors expected pool layout, **or** empty prefix with normal FHS paths.
5. Startup fails fast if prefix is set but not a usable directory; optional containment checks document symlink behavior under `prefix`.
6. **Operator / contributor docs** include a **contained non-root runbook**: how to create a minimal directory tree under a writable prefix, which flags and env vars to set (`--prefix`, optional `--config` with absolute-looking paths), and how that maps to real paths on disk. At minimum **`README.md`** (short pointer + example), **`docs/admin-guide.md`** (full section), and **`docs/deployment.md`** (deploy + systemd alignment) are updated; **`deploy/systemd/README.md`** (and unit example if present) show an optional `--prefix` service layout.
7. **Testing path** is explicit in docs and in this plan: **automated** coverage via `go test` (including `t.TempDir()` + `--prefix` / `KUI_PREFIX` via `run()` or config-load seams—no requirement to run the compiled binary in CI); **manual** optional smoke steps documented for humans who want to run `kui` from `bin/` in contained mode after `go build` (directory layout checklist + sample invocation). Project automation policy may forbid agents from executing `bin/kui`; the spec still requires **humans** to have a documented path.

## Non-goals

- Rewriting paths inside **guest domain XML** or libvirt’s own storage definitions (unless a separate spec adds that).
- Literal `chroot(2)` / namespace isolation in the Go process.
- Migration from old “absolute bypasses prefix” behavior (greenfield; this spec defines target behavior before shipping).
- Replacing automated tests with mandatory manual-only verification; manual steps are **supplementary** documentation, not a substitute for unit/integration tests.
