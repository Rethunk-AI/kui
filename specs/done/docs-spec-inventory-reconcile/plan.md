# docs-spec-inventory-reconcile — Plan

## Overview

Reconcile **documentation and inventory** with repository truth: `specs/done/` currently contains **25** completed spec directories (each with `plan.md`). `TODO.md` and parts of the PRD backlog still reflect an older inventory (e.g. “18 specs”, v2 table listing shipped work as future). This work is **docs-only**; no product runtime behavior changes unless a **low-cost** Makefile helper (e.g. `make specs-list`) is added to make the canonical list auditable.

**Canonical done-spec IDs** (alphabetical; verify with `ls specs/done` before editing):

`api-auth`, `coverage-100`, `feat-a11y`, `feat-domain-xml-edit`, `feat-host-provisioning`, `feat-keyboard-shortcuts`, `feat-orphan-bulk`, `feat-setup-wizard-ui`, `feat-shadcn-ui`, `feat-stuck-vm`, `gap-401-session-audit`, `gap-audit`, `gap-domain-xml-network`, `gap-remediation`, `gap-template-network`, `schema-storage`, `setup-host-validation`, `spec-application-bootstrap`, `spec-audit-integration`, `spec-console-realtime`, `spec-frontend-build`, `spec-libvirt-connector`, `spec-template-management`, `spec-ui-deployment`, `spec-vm-lifecycle-create`

**Gap-audit theme (for cross-ref accuracy in backlog / decision-log pointers):**

| Spec ID | Role (for doc cross-refs) |
|---------|---------------------------|
| `gap-audit` | Umbrella gap closure (incl. later gap IDs 9–14 per TODO narrative) |
| `gap-remediation` | Remediation-related gaps |
| `gap-401-session-audit` | 401 / session audit behavior |
| `gap-domain-xml-network` | Domain XML / network gaps |
| `gap-template-network` | Template / network gaps |

Do **not** invent features; every doc change must match shipped specs and `specs/done/*/plan.md` (or `spec.md` where that is the normative link target for readers).

---

## Decision log

| Decision | Alternatives | Why chosen | Risks / mitigation |
|----------|--------------|------------|---------------------|
| Treat `ls specs/done` (or `make specs-list`) as source of truth for counts/lists | Hardcode “25” in prose | Count drifts again on next spec; automation stays honest | If Makefile target added, document that CI optional check could fail if dirs removed—acceptable greenfield |
| Link shipped v2 backlog rows to `specs/done/<id>/plan.md` | Link only `spec.md` | Plans summarize outcomes; team may prefer `spec.md` for API detail—**pick one convention per row** and stay consistent | If both linked, keep table readable (single primary link + optional secondary) |
| Optional `make specs-list` | Shell script in `scripts/` | Makefile is already canonical for verification; one-liner fits “low-cost” | Keep target pure (list dirs only); no Go code required |
| Update `docs/prd.md` §2 “Resolved” sentence | Leave as-is | Contradicts shipped state; PRD says “docs = truth” | Edit minimally: mark those items shipped + link specs |
| `Continue_TODO_Prompt.md` | Delete or full rewrite | File describes wrong product (game/checker); confuses continuation agents | Either **retarget** to KUI + TODO.md workflow or **archive** with pointer to `AGENTS.md` / workflow rules—do not leave misleading text |

---

## Ownership boundaries

**In scope**

- `TODO.md`
- `docs/prd/backlog.md`
- `docs/prd.md` (§2 and §4 reference table if backlog section changes)
- `Continue_TODO_Prompt.md` (stale / wrong-project guidance)
- `Makefile` (optional: `specs-list` or similar)
- `docs/prd/decision-log.md` (only if backlog/PRD cross-refs require pointer fixes—no decision rewrites)
- Repo sweep: `README.md`, `AGENTS.md`, `.github/**/*.md`, `.github/workflows/*`, other `docs/**/*.md` for stale counts

**Out of scope**

- Application code, schema, APIs, tests (unless adding Makefile)
- Moving specs between `active` / `done`
- CI product behavior beyond documenting optional future “spec list drift” check (not required unless desired)

---

## Ordered tasks (with ownership)

1. **Developer — Canonical inventory**  
   - [x] Enumerate `specs/done` directory names; confirm count **25** and set matches plan list above.  
   - [x] Update `TODO.md`: replace “All 18 specs” / “Done specs | 18” with **25**; append the **seven** missing IDs to the comma list: `gap-remediation`, `feat-host-provisioning`, `gap-401-session-audit`, `feat-setup-wizard-ui`, `gap-domain-xml-network`, `gap-template-network`, `setup-host-validation`.  
   - [x] Re-read “Gap audit” paragraph: ensure references to gaps 1–8 still align with the **five** gap-theme specs (no new claims).

2. **Developer — Backlog v2 table** (`docs/prd/backlog.md`)  
   - [x] For rows that are **shipped**, change narrative from “future v2” to **Done** (or move rows to a “Completed (see specs)” subsection).  
   - [x] Add markdown links to canonical spec artifacts, e.g.  
     - Keyboard shortcuts → `specs/done/feat-keyboard-shortcuts/plan.md` (or `spec.md` if present and preferred)  
     - WCAG a11y → `feat-a11y`  
     - Stuck VM → `feat-stuck-vm`  
     - Orphan bulk → `feat-orphan-bulk`  
     - Domain XML edit → `feat-domain-xml-edit`  
   - [x] Keep **v3** (backup/restore, import/export) clearly future.  
   - [x] Optionally add short “Gap closure” bullets linking `gap-audit`, `gap-remediation`, `gap-401-session-audit`, `gap-domain-xml-network`, `gap-template-network` if decision-log/backlog readers expect them—only if it reduces confusion without duplicating decision-log.

3. **Developer — PRD §2 reconcile** (`docs/prd.md`)  
   - [x] Line ~20: split **still open** vs **shipped**. Stuck VM, orphan bulk, domain XML edit are implemented per done specs—describe as **shipped** with links; do not label them “all v2” in a way that implies backlog-only.  
   - [x] Ensure §4 references row for `backlog.md` still accurate after backlog edit.

4. **Developer — Continue prompt** (`Continue_TODO_Prompt.md`)  
   - [x] Remove or replace game/checker-specific explorer bullets (PRD path, checker, scoring) with **KUI**-accurate paths (`docs/prd.md`, `docs/`, `specs/done/`, `specs/active/`).  
   - [x] Align Phase 0 with repo rules (`make all` where appropriate) without contradicting `AGENTS.md` / `.cursor/rules/workflow.mdc`.

5. **Developer — Optional Makefile**  
   - [x] If team wants a single command: add e.g. `specs-list` that prints sorted `specs/done` basenames (e.g. `ls -1 specs/done \| sort`).  
   - [x] Document in plan verification below; **no** requirement to wire CI unless explicitly chosen later.

6. **Developer — Dashboard sweep (second pass)**  
   - [x] Grep (case-insensitive as needed) for stale patterns: `All 18`, `Done specs | 18`, `18 specs`, `done specs`, standalone `\b18\b` near “spec” context, `v2 — Enhancements` prose that implies unimplemented shipped features.  
   - [x] Files to include: `README.md`, `TODO.md`, `docs/**/*.md`, `AGENTS.md`, `.github/**/*.md`, `.github/workflows/*`, root `*.md`.  
   - [x] Fix or ticket any drift found (same PR as above if trivial).

7. **Verifier**  
   - [x] Confirm no invented features; links resolve.  
   - [x] `make specs-list` run and matched `ls -1 specs/done | sort`. `make all` not green on verifier/developer hosts (missing libvirt pkg-config); unrelated to `specs-list` addition — confirm on a host with libvirt dev packages.

---

## File-by-file edit checklist

| File | Action |
|------|--------|
| `TODO.md` | Fix count (25); full ID list; table “Done specs”; “Last updated” date |
| `docs/prd/backlog.md` | v2 table → shipped + links; preserve v3 future |
| `docs/prd.md` | §2 resolved line: shipped vs deferred; §4 table if needed |
| `docs/prd/decision-log.md` | Only adjust cross-references if backlog/PRD point to wrong sections—no scope creep |
| `Continue_TODO_Prompt.md` | KUI-accurate explorer tasks; remove wrong-game content |
| `Makefile` | Optional `specs-list` target + `.PHONY` |
| `README.md` / `AGENTS.md` / `.github/**` | Sweep only if grep hits |
| This plan | Append changelog entry if scope changes during implementation |

---

## Acceptance criteria

- [x] `TODO.md`: stated **done** count equals **number of directories** in `specs/done/` (25 at time of verification); comma-separated list is a **permutation** of that directory set (no omissions, no extras).  
- [x] `docs/prd/backlog.md`: items confirmed shipped in repo are **not** presented as unfinished v2-only work; each has a **working** relative link to `specs/done/<id>/plan.md` or team-chosen `spec.md`.  
- [x] `docs/prd.md` §2: no contradiction where stuck VM, orphan bulk, domain XML are simultaneously “v2 backlog-only” and implemented.  
- [x] `Continue_TODO_Prompt.md`: no instructions that reference a non-KUI PRD or game mechanics.  
- [x] Dashboard sweep: documented patterns re-grepped; fixes applied or explicitly out-of-scope with reason.  
- [x] If `make specs-list` exists: its output matches `ls -1 specs/done \| sort`.

---

## Verification

- **Default:** `make all` — expected **unchanged** outcome if only markdown edits (and optional Makefile one-liner).  
- **`specs-list`:** `make specs-list` prints sorted basenames of `specs/done` (same as `ls -1 specs/done | sort`); use `diff <(ls -1 specs/done | sort) <(make -s specs-list)` to confirm.  
- **Do not** execute built binaries per project workflow; doc/link checks are manual or script-assisted.

---

## Rollout / ops

None beyond merging doc changes. Optional: mention `make specs-list` in `TODO.md` or contributor docs **only if** the target is added.

---

## Assumptions and open questions

- **Assumption:** All 25 `specs/done/*/` folders are intentional “completed” specs (each has `plan.md`).  
- **Question:** For backlog links, is **`plan.md`** or **`spec.md`** the preferred reader entry point? (Pick one for the v2 shipped table.)  
- **Question:** Should `Continue_TODO_Prompt.md` remain as a first-class doc or be trimmed to “see AGENTS.md” after retargeting?

---

## Changelog

| Date | Change |
|------|--------|
| 2026-03-19 | Initial plan from repo verification (25 done specs; TODO 18; backlog/PRD stale v2; Continue_TODO_Prompt wrong project). |
| 2026-03-23 | Implementation: `make specs-list` added to Makefile (sorted `specs/done` basenames); TODO.md, backlog, PRD §2/§4, Continue_TODO_Prompt reconciled to 25 done specs and shipped v2 items. |
| 2026-03-23 | Task 7 verification: acceptance criteria met; `make specs-list` verified; `make all` blocked on agents without libvirt `.pc` (environmental). TODO.md last ID line: no period glued to basename. |

---

## Approval checklist

- [ ] Scope matches intent (no extra features)
- [ ] File paths and ownership are clear
- [ ] Data model is correct and safe (greenfield: no migration/backfill)
- [ ] Authn/authz + context scoping are correct (N/A — docs only)
- [ ] API contracts are specified (N/A — docs only)
- [ ] Test plan + verification steps are included
- [ ] Rollout/ops notes are sufficient (if needed)

---

## Recommended next steps

1. Run **Task 1** (`TODO.md`) and **Task 2** (`backlog.md`) in one doc-focused PR.  
2. Run **Task 3** (`docs/prd.md` §2) in the same PR if small.  
3. **Task 4** (`Continue_TODO_Prompt.md`) — decide retarget vs archive, then edit.  
4. **Task 6** grep sweep; fix stragglers.  
5. Delegate **verifier** with explicit checklist against acceptance criteria.
