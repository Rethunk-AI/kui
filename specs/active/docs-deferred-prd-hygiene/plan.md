# docs-deferred-prd-hygiene — Plan

## Objective summary

Align PRD/stack documentation with the canonical Go module (`go.mod`: **Go 1.24.0**, libvirt XML module **`libvirt.org/go/libvirtxml`** v1.12001.0). Introduce a single, factual **deferred product backlog** table that links each deferred item to PRD/decision references and to existing specs where they exist, marking gaps explicitly as **no spec yet**.

**Default scope:** documentation only. Change application code only if an unavoidable contradiction with `go.mod` or imports is discovered during implementation (unlikely given current facts).

---

## Architecture overview

No runtime architecture change. This work is **documentation + PRD hygiene**:

- **Canonical source of truth** for dependencies: repository root `go.mod` and import paths under `internal/`.
- **PRD surface**: `docs/prd/*` remains the narrative home; the new deferred table centralizes cross-links without duplicating full spec text.

---

## Exact files to create or edit

| Action | Path | Purpose |
|--------|------|---------|
| **Create** | `docs/prd/deferred.md` | Home for the deferred backlog table + short intro (what “deferred” means in this repo: greenfield MVP vs later work). |
| **Edit** | `docs/prd/backlog.md` | Add a **one-line link** near the top to `deferred.md` (e.g. “See [deferred.md](deferred.md) for items explicitly deferred with spec/PRD pointers.”). |
| **Edit** | `docs/prd/stack.md` | §1 Backend: fix XML row to **`libvirt.org/go/libvirtxml`** (match `go.mod` and code). Add or align **Go toolchain** statement with **`go 1.24.0`** from `go.mod` (wording: **1.24+** or explicit **1.24.0** — pick one and use consistently with `admin-guide.md`). |
| **Edit** | `docs/admin-guide.md` | Replace **Go 1.22+** with wording aligned to `go.mod` (same as `stack.md`). |

**Out of scope (do not edit unless contradiction found):** `go.mod`, `go.sum`, any `internal/**` or `cmd/**` source.

---

## Deferred backlog — markdown table template

Copy into `docs/prd/deferred.md` and fill **Notes** with terse factual text only (no marketing). **Spec ID** is the directory name under `specs/done/` or the literal `no spec yet`.

| Item | PRD / decision reference (path) | Spec ID | Notes |
|------|----------------------------------|---------|-------|
| OIDC / Authentik SSO | `docs/prd/decision-log.md` (A5 Session, Auth rows; cite exact §/anchors if present) | `api-auth` | `specs/done/api-auth/spec.md` defers Authentik/OIDC to a **separate future spec**; MVP is local auth + JWT. |
| SPICE v2 in-browser client | `docs/prd/decision-log.md` + `specs/done/spec-console-realtime/spec.md` (SPICE v2 out of scope for that spec) | `spec-console-realtime` | Console MVP is noVNC + xterm; SPICE deferred per PRD/spec boundary. |
| Backup/restore / import-export (v3) | `docs/prd/backlog.md` (v3 section) + `docs/prd/decision-log.md` (backup/restore defer) | `no spec yet` | `specs/done/schema-storage/spec.md` may mention advanced import/export tools deferred; no dedicated backup/restore spec identified. |
| Maintenance mode (KUI maintenance for upgrade/DB) | `docs/prd/decision-log.md` (A15, KUI maintenance mode rows) | `no spec yet` | No spec directory found referencing maintenance mode; table records the gap. |

**Optional (developer judgment):** If `docs/prd/README.md` or `docs/prd/index` exists, add one line pointing to `deferred.md`. If no such index exists, skip (stay minimal).

---

## Acceptance criteria

1. **`docs/prd/stack.md` §1** lists XML module as **`libvirt.org/go/libvirtxml`**, consistent with `go.mod` and imports (`internal/provision`, `internal/routes/vnc`, etc.).
2. **Go version** in `stack.md` and `admin-guide.md` matches **`go.mod`** (`go 1.24.0` — use **1.24+** or **1.24.0** consistently in both files; no lingering **1.22+**).
3. **`docs/prd/deferred.md`** exists and contains the table (or equivalent structure) with columns **Item | PRD / decision reference | Spec ID | Notes**, populated for the four rows above with factual notes.
4. **`docs/prd/backlog.md`** links to `deferred.md` from the top (or immediately after title/front matter).
5. **Links** to existing specs use repo-relative paths (e.g. `../specs/done/api-auth/spec.md` from `docs/prd/` or path style already used in sibling docs).
6. **No new claims** that contradict `specs/done/*/spec.md` or `go.mod`; where work is not specified, state **`no spec yet`** rather than inventing scope.
7. **Greenfield:** no migration/backfill language, no “temporary compatibility” for wrong module names.

---

## Verification

- **Manual review:** Read edited sections for typos, broken relative links, and table formatting.
- **Re-grep consistency:**
  - `grep` / ripgrep in `docs/` for **`libvirt-go-xml`** (legacy wrong path) — expect **zero** hits after fix (except possibly historical changelog text; if any, fix or remove).
  - Confirm **`libvirtxml`** appears in `stack.md` aligned with narrative.
  - Confirm **no `Go 1.22`** remains in `docs/admin-guide.md` unless intentionally documenting historical context (prefer removal/update).
- **`make all` / `go test`:** **Not required** for this task unless documentation work uncovers a contradiction that forces code or module changes (then run `make all` once after those changes).

---

## Security and performance

Not applicable beyond avoiding misleading operators (wrong Go or module paths could cause failed builds or wrong dependency assumptions).

---

## Rollout / ops

None. Merge documentation updates; no config or deployment changes.

---

## Assumptions and open questions

- **Assumption:** `docs/prd/decision-log.md` contains locatable rows for A5, A15, Session, Auth, backup/restore defer, and maintenance mode — cite the smallest precise reference (section heading + row label) the document structure allows.
- **Open question (resolve during edit):** Whether to state **“Go 1.24.0”** verbatim vs **“Go 1.24+”** in prose; either is acceptable if it matches `go.mod` and is consistent across `stack.md` and `admin-guide.md`.

---

## Decision log

| Decision | Alternatives | Why chosen | Risks / mitigations |
|----------|--------------|------------|---------------------|
| New `docs/prd/deferred.md` vs only a subsection in `backlog.md` | Single long `backlog.md` | **Separate `deferred.md`** keeps backlog readable and gives one URL for “what we explicitly deferred.” | Risk: orphan doc → **mitigation:** one-line link from top of `backlog.md`. |
| Spec ID column uses directory name under `specs/done/` | Full path only | Matches repo convention and greppable IDs. | Risk: ambiguity for `no spec yet` → **mitigation:** literal phrase in table. |
| Docs-only default | Also bump `go.mod` | User constraint + no requirement to change toolchain. | If grep finds doc/code mismatch beyond the known XML path, reassess in implementation. |

---

## Ownership boundaries

**In scope**

- `docs/prd/deferred.md` (new)
- `docs/prd/backlog.md`
- `docs/prd/stack.md`
- `docs/admin-guide.md`

**Out of scope**

- Application source (`cmd/`, `internal/`), tests, `go.mod`, CI, and specs under `specs/done/` (read-only references unless a contradiction forces a doc fix elsewhere)

---

## Changelog

- **2026-03-19:** Initial plan (docs alignment + deferred table + verification).

---

## Approval checklist

- [ ] Scope matches intent (no extra features)
- [ ] File paths and ownership are clear
- [ ] Data model is correct and safe (greenfield: no migration/backfill) — N/A for docs-only; table is narrative only
- [ ] Authn/authz + context scoping are correct — N/A; links only
- [ ] API contracts are specified — N/A
- [ ] Test plan + verification steps are included
- [ ] Rollout/ops notes are sufficient — N/A beyond “none”

---

## Recommended next steps

1. Developer: create `deferred.md`, populate table with exact anchors from `decision-log.md` after a quick read of that file’s structure.
2. Developer: patch `stack.md` (XML + Go), `admin-guide.md` (Go), `backlog.md` (link).
3. Run verification greps; fix any stray wrong module strings in `docs/`.
4. Verifier: confirm acceptance criteria and link integrity only (trust no code change unless scope expanded).
