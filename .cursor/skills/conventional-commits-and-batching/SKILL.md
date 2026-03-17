---
name: conventional-commits-and-batching
description: Create clean conventional commits by batching related diffs and writing "why-focused" messages. Use when the user mentions commits, git status/diff, conventional commits, splitting changes, or preparing PR history.
---

# Conventional Commits + Batching

Standardizes commit grouping and message quality.

## Batching workflow

1. **Review** `git status` and diff; identify logical themes.
2. **Group** by reason (one theme per commit).
3. **Stage** in dependency order when applicable (e.g. Go: schema/proto → internal → cmd).

## Batching rules

- One reason, one theme per commit.
- Prefer 5–15 files unless inseparable (renames, lockfile pairs, generated outputs).
- Separate mechanical from functional when feasible:
  - formatting/lint-only
  - renames/moves
  - dependency bumps

## Pairing rules (do not split)

- `package.json` + lockfile
- delete+add when renaming a file
- migrations + the code that depends on them

## Conventional commit template

```text
<type>(<scope>): <imperative summary>

<why this change exists; user value or risk reduction>
<notes: migrations, flags, rollback idea, follow-ups>

BREAKING CHANGE: <description>   # only when API/contract changes
```

- **Scope**: omit when obvious from diff; use package/area when it clarifies.
- **Body**: explain why, not what files changed.

### Types

- `feat`: user-facing feature
- `fix`: bug fix
- `refactor`: behavior-preserving restructure
- `perf`: performance improvement
- `test`: tests only
- `docs`: documentation only
- `chore`: tooling/maintenance

## Examples

```text
feat(files): add directory-aware navigation

Enable deep links into nested directories and keep breadcrumbs consistent with access context.
```

```text
fix(auth): prevent cross-tenant lookup in server action

Add scope filters to the update query so users cannot mutate records outside their org.
```

```text
refactor(config): extract validation into separate package

Reduces coupling and enables reuse in CLI and server. No behavior change.
```

```text
chore(deps): bump go to 1.23

Required for new stdlib APIs used in upcoming auth changes.
```

```text
feat(api)!: require tenant header on all endpoints

BREAKING CHANGE: Clients must send X-Tenant-ID. Migration guide in docs/migration.md.
```

## Quick checklist

- [ ] One coherent theme per commit
- [ ] Message explains "why", not a file list
- [ ] Risky change has a note (migration, flag, rollback idea)
