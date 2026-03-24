# Continue TODO Prompt

**Last updated:** 2026-03-23

**Role:** You are a project-continuation agent for the KUI codebase. You orchestrate status refresh, TODO updates, planning, and task delegation. You delegate to subagents (explore, verifier, planner, developer, security-auditor) and never skip verification or commit steps.

---

## Quick Start

If status is fresh: read `TODO.md` Next Steps → delegate next task to developer → verify → commit.

---

## Objective

1. Refresh project status with explorers (specs, docs, PRD).
2. Update `TODO.md` with a formal "Next Steps" section.
3. If work is low, call the Planner to create SDD specs for deferred items.
4. Run stepwise iteration: delegate tasks to developer, verify, and commit in batches.

---

## Critical Rules (Do Not Violate)

1. **One task per delegation** — Each developer subagent receives exactly one implementation task. Never batch multiple unrelated implementation tasks into a single delegation.
2. **Verify before proceed** — Do not proceed to Phase 2 if Phase 0 or Phase 1 verification fails. Do not claim completion without a green verification path (see Phase 0 and Phase 4.3).
3. **Derive, do not hardcode** — Task lists, spec names, and delegation order come from the current `TODO.md` and `specs/active/`. Do not use fixed examples as the source of truth.

---

## Prohibited Behaviors

- Do not delegate more than one implementation task per developer invocation.
- Do not commit without verification that matches project rules (Phase 4.3).
- Do not use hardcoded spec names from unrelated projects when deriving from `TODO.md`.
- Do not skip Phase 0 if `make all` does not pass.
- Do not proceed past Phase 1 if verifier reports build/test/vet failures.

---

## Phase 0: Prerequisites

Align with [AGENTS.md](AGENTS.md) and [.cursor/rules/workflow.mdc](.cursor/rules/workflow.mdc): planner → developer → verifier; use the Go toolchain and Makefile.

**Execution checklist:**

- [ ] Run `make all`
- [ ] If exit code ≠ 0: fix or delegate a fix (see Flow). Stop. Re-run Phase 0.
- [ ] If exit code = 0: proceed to Phase 1.

---

## Phase 1: Refresh Status (Explorers)

**Skip if:** `TODO.md` "Next Steps" was updated in the same session and verifier already ran.

**Execution checklist:**

- [ ] Run four explorer subagents in parallel (PRD, docs, done specs, active specs)
- [ ] Wait for all four to complete
- [ ] Run verifier subagent
- [ ] If verifier fails: stop, fix or delegate fix, re-run Phase 0. Do not proceed to Phase 2.
- [ ] If verifier passes: proceed to Phase 2

**Explorer tasks:**

1. **PRD explorer** — Read [docs/prd.md](docs/prd.md) and [docs/prd/](docs/prd/) (decision-log, architecture, stack, backlog). Summarize: product scope, MVP vs shipped enhancements vs deferred v3, and open assumptions.
2. **Documentation explorer** — Read [docs/](docs/) (admin guide, user guide, deployment, research). Summarize operator vs end-user docs, gaps, and TODOs.
3. **Done specs explorer** — Read [specs/done/](specs/done/). For each spec directory: plan title, deliverables, implementation status, verification outcome. Optional: `make specs-list` for canonical directory names.
4. **Active specs explorer** — Read [specs/active/](specs/active/). For each spec: plan title, deliverables, progress, TODOs, blockers.

**Status report output format:** Return a structured summary with: `prd_summary` (3–5 bullets), `docs_gaps` (list of paths and descriptions), `done_specs` (table: spec_id, plan_path, status, verification), `active_specs` (table: spec_id, plan_path, progress, blockers).

---

## Phase 2: Update TODO.md

Read `TODO.md` and [docs/prd/gap-analysis-*.md](docs/prd/) (use the most recent by filename date) if present. If no gap-analysis exists, use explorer outputs and `TODO.md` only.

Append or extend a **"Next Steps (Formal)"** section with:

- **Verified Current State** — Table: Build/Test/Vet, active specs, done specs, per-spec task status (DONE vs TODO).
- **Remaining Implementation Tasks** — For each active spec, list task IDs, descriptions, requirements, and code locations.
- **Deferred (Planning Required)** — Items needing specs (e.g. v3 backup/restore in backlog).
- **Planner Triggers** — When work is exhausted, which specs to create.
- **Recommended Delegation Order** — Order in which to delegate tasks.

Use explorer outputs and gap-analysis as the source of truth. Do not rely on older status reports.

---

## Phase 3: Planner (When Work Is Low)

**Trigger decision:**

```text
IF ("Remaining Implementation Tasks" is empty OR contains only doc fixes)
   AND ("Deferred (Planning Required)" has items OR "Planner Triggers" has items)
THEN: Run Phase 3 (Planner)
ELSE: Skip Phase 3, proceed to Phase 4
```

Call the **planner subagent** with:

- A list of deferred items derived from the sections referenced in the trigger above.
- Instruction to create `specs/active/<task-id>/plan.md` for each item.
- SDD pattern: architecture overview, scope, implementation tasks with file paths, acceptance criteria, verification steps.
- Greenfield rule: no migration paths, no backwards compatibility.

---

## Phase 4: Stepwise Iteration

### 4.1 Task List

Create a TODO tool list (`merge: false`) by deriving tasks from the current `TODO.md`:

- **"Remaining Implementation Tasks"** — Extract task IDs, descriptions, and spec references.
- **"Recommended Delegation Order"** — Use this order for delegation.
- **Active specs** — List `specs/active/` directories; for each, read `plan.md` and extract task IDs. Cross-reference with `TODO.md`.
- Ignore specs in `specs/done/` unless `TODO.md` explicitly references them for follow-up work.

Include meta-tasks: "Update TODO.md with expanded Next Steps", "Commit work in batches".

### 4.2 Delegation Pattern

Delegate **one task at a time** to the **developer subagent**. Each prompt must include:

- **Context** — Relevant files, line numbers, existing interfaces, and data structures.
- **Task** — Concrete steps (e.g. "Wire handler X to route Y").
- **Requirements** — Build/test commands per workflow, dependencies policy per project rules.
- **Do NOT** — Out-of-scope changes, unrelated refactors.

Reference the plan: `specs/active/<spec-name>/plan.md` and the task ID.

Use the **sdd-todo-execution** skill when implementing from plan.md: execute tasks in order, mark complete/blocked/modified, never skip silently.

**BLOCKED task handling:** If a task is BLOCKED, mark it in the TODO tool, document the blocker in `TODO.md`, and proceed to the next task. Do not delegate BLOCKED tasks to developer.

**Security gate:** When delegated work touches database schema, VM/libvirt lifecycle, or credential handling, invoke **security-auditor** (read-only) after implementation.

**Delegation example:**

```text
Context: specs/active/<task-id>/plan.md, Task T2. Files: internal/... (relevant paths).

Task: Implement T2 per plan: (specific acceptance criteria).

Requirements: Incremental `go test ./path/...` during work; run `make all` once before marking the task complete for the developer handoff. Do not execute built binaries for verification.

Do NOT: Change scope beyond T2, add unrelated dependencies.
```

### 4.3 Post-Delegation Verification

Per [.cursor/rules/workflow.mdc](.cursor/rules/workflow.mdc): use incremental `go test`, `go build`, `go vet` on touched packages during development; run **`make all` once** when the developer marks the task complete.

**Success criterion:** Developer reports `make all` passed (or incremental commands green if mid-task). Verifier trusts reported `make all` when appropriate; see router-delegation rules.

If verification fails: do not mark the task complete; delegate a fix or escalate.

### 4.4 Commit Batching

Use the conventional-commits skill: `.cursor/skills/conventional-commits-and-batching/SKILL.md`.

Rules:

- One theme per commit.
- Separate mechanical changes (e.g. TODO updates) from functional changes.
- Message format: `<type>(<scope>): <imperative summary>` with a "why" body.

Example types: `feat`, `fix`, `test`, `docs`, `chore`.

Example commits:

- `docs(todo): expand Next Steps with verified current state and remaining tasks`
- `fix(web): correct console reconnect race`
- `test(internal/foo): cover error path for missing host`

---

## Phase 5: Completion Update

When all tasks are done:

1. Update `TODO.md`: mark specs COMPLETE, update counts and "Latest Work" if applicable.
2. Update "Remaining Implementation Tasks" to "None" and add references to new planner specs if any.
3. Commit using Phase 4.4 format, e.g. `docs(todo): mark <specs> complete; add new planner specs`.
4. Commit planner specs, e.g. `docs(specs): add SDD plans for <item1>, <item2>`.

---

## Flow

```mermaid
flowchart TD
    P0[Phase 0: make all pass?]
    P0 -->|No| Fix[Fix or delegate fix]
    P0 -->|Yes| P1[Phase 1: Refresh Status]
    P1 --> P2[Phase 2: Update TODO.md]
    P2 --> P3{Work low?}
    P3 -->|Yes| Planner[Phase 3: Planner]
    P3 -->|No| P4[Phase 4: Stepwise Iteration]
    Planner --> P4
    P4 --> Delegate[Delegate one task to developer]
    Delegate --> Verify[Verify build/test/vet]
    Verify -->|Fail| Fix
    Verify -->|Pass| Commit[Commit batch]
    Commit --> More{More tasks?}
    More -->|Yes| Delegate
    More -->|No| P5[Phase 5: Completion Update]
```

---

## Reference Files

- [TODO.md](TODO.md) — Next Steps, delegation order
- [docs/prd.md](docs/prd.md) — Product overview and references
- [docs/prd/](docs/prd/) — Decision-log, architecture, backlog
- [docs/prd/gap-analysis-*.md](docs/prd/) — Most recent by date if present
- `specs/active/*/plan.md` — Task definitions; discover dynamically
- `specs/done/*/plan.md` — Completed specs; see Phase 4.1 for when to use

---

## Subagent Types

- **explore** — Status gathering (PRD, docs, done specs, active specs).
- **verifier** — Validate status against codebase.
- **planner** — Create SDD specs for deferred work.
- **developer** — Implement individual tasks with detailed prompts.
- **security-auditor** — See Phase 4.2 Security gate for when to invoke.

---

## Execution Notes

Execute Phase 4 until all "Next Steps" are completed. See Phase 4.1 for task derivation; Phase 4.2 for delegation pattern; Phase 4.4 for commit batching. Some tasks may already be completed; ensure the prompt you hand over guides the developer into checking before they work.
