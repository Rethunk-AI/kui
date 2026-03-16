# Canonical Configuration and Tool Enhancements

Recommendations for additional project files, with one canonical source per config type and symlinks for tool-specific names. LLMs must be instructed to edit only canonical files.

**Sources:** [Cursor Rules](https://cursor.com/docs/rules), [Cursor Subagents](https://cursor.com/docs/subagents), [Cursor Skills](https://cursor.com/docs/skills) (2026).

**Cursor built-in commands:** Use `/create-rule`, `/create-skill`, or ask Agent to create subagents. These write to `.cursor/rules/`, `.cursor/skills/`, `.cursor/agents/` respectively.

---

## 1. AI / LLM Instructions (Canonical: `.cursor/`)

**Current:** `.cursor/rules/`, `.cursor/agents/`, `.cursor/skills/` — Cursor 2026 native.

**Cursor 2026 hierarchy:** Team Rules → Project Rules (`.cursor/rules`) → User Rules. `AGENTS.md` is a "simple alternative" to `.cursor/rules` — plain markdown, no metadata. `.cursorrules` is legacy/deprecated; prefer `.cursor/rules/`.

**Claude / Codex compatibility:** Cursor loads from `.claude/agents/`, `.codex/agents/`, `.claude/skills/`, `.codex/skills/` for compatibility. Project `.cursor/` takes precedence when names conflict.

**Solution:**

| Item | Role |
|------|------|
| **`.cursor/rules/`** | Canonical rules. Edit only these. |
| **`.cursor/agents/`** | Canonical subagents. Edit only these. |
| **`.cursor/skills/`** | Canonical skills. Edit only these. |
| **`AGENTS.md`** | Optional: simple entry point for tools that read only one file. States `.cursor/` is canonical; do not edit symlinks. |
| **`.claude/agents/`** | Symlink → `../.cursor/agents/` (Claude Code compatibility) |
| **`.codex/agents/`** | Symlink → `../.cursor/agents/` (Codex compatibility) |
| **`.claude/skills/`** | Symlink → `../.cursor/skills/` (Claude Code compatibility) |
| **`.codex/skills/`** | Symlink → `../.cursor/skills/` (Codex compatibility) |

**Do not create:** `.cursorrules` (legacy). Do not duplicate rules in `AGENTS.md` — use it only as a pointer if needed.

**AGENTS.md content (if used):**

```markdown
# KUI — AI/LLM Instructions

**Canonical configuration:** `.cursor/rules/`, `.cursor/agents/`, `.cursor/skills/`. Edit only those.

**Do not edit:** `.claude/`, `.codex/` — they are symlinks to `.cursor/`.

## Summary
- Workflow: planner → developer → verifier. See `.cursor/rules/workflow.mdc`.
- Greenfield: no migrations. See `.cursor/rules/greenfield.mdc`.
- Go: see `.cursor/rules/go-standards.mdc`.
```

---

## 2. Editor Config (Canonical: `.editorconfig`)

**Purpose:** Shared formatting (indent, charset, trim) across VS Code, Cursor, JetBrains, etc.

**Add:** `.editorconfig` at repo root.

```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true

[*.go]
indent_style = tab
indent_size = 4

[*.{md,yml,yaml,json}]
indent_style = space
indent_size = 2
```

No symlinks needed — EditorConfig is the standard.

---

## 3. Git Ignore (Canonical: `.gitignore`)

**Current:** None.

**Add:** `.gitignore` with Go + IDE + OS patterns.

```
# Binaries
/bin/
*.exe

# Go
*.test
coverage.out
coverage.html

# IDE
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
```

---

## 4. Lint Config (Canonical: `.golangci.yml`)

**Current:** CI uses golangci-lint with defaults.

**Add:** `.golangci.yml` at repo root. CI (GitHub + GitLab) and local `golangci-lint run` both use it.

```yaml
run:
  timeout: 5m
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - ineffassign
```

---

## 5. VS Code / Cursor Tasks (Canonical: `.vscode/tasks.json`)

**Purpose:** `make all`-equivalent and test tasks for VS Code / Cursor.

**Add:** `.vscode/tasks.json` with `build`, `test`, `vet` tasks. Cursor inherits VS Code tasks.

No symlinks — `.vscode/` is shared.

---

## 6. GitLab Project README (Canonical: `README`)

**Current:** Root `README` exists.

**Option:** Symlink `.gitlab/README.md` → `../README` so GitLab project page shows the same content. GitLab renders `README`, `README.md`, or `.gitlab/README.md`; symlink avoids duplication.

---

## 7. Contributing (Canonical: `CONTRIBUTING.md`)

**Purpose:** GitLab and GitHub both surface this for new contributors.

**Add:** `CONTRIBUTING.md` with: how to build/test, PR/MR checklist, link to SECURITY.md. Can reference `.github/PULL_REQUEST_TEMPLATE.md` for the checklist.

---

## 8. Summary: Canonical vs Symlink

| Config Type | Canonical | Symlinks / Notes |
|-------------|-----------|------------------|
| Rules | `.cursor/rules/` | — |
| Subagents | `.cursor/agents/` | `.claude/agents/` → `../.cursor/agents/`, `.codex/agents/` → `../.cursor/agents/` |
| Skills | `.cursor/skills/` | `.claude/skills/` → `../.cursor/skills/`, `.codex/skills/` → `../.cursor/skills/` |
| AI entry point | `AGENTS.md` (optional pointer) | — |
| Editor | `.editorconfig` | — |
| Git ignore | `.gitignore` | — |
| Lint | `.golangci.yml` | — |
| IDE tasks | `.vscode/tasks.json` | — |
| Project README | `README` | `.gitlab/README.md` → `../README` (optional) |
| MR/PR template | `.github/PULL_REQUEST_TEMPLATE.md` | `.gitlab/merge_request_templates/default.md` (already symlinked) |

---

## 9. LLM Instruction (Add to Workflow Rule)

Add to `.cursor/rules/workflow.mdc` (or a new `rules/canonical-config.mdc`):

```markdown
## Canonical configuration

- **Rules, subagents, skills:** Edit only `.cursor/rules/`, `.cursor/agents/`, `.cursor/skills/`. Do not edit `.claude/` or `.codex/` — they are symlinks to `.cursor/`.
- **Lint:** Edit `.golangci.yml` only.
- **Editor:** Edit `.editorconfig` only.
- **Git:** Edit `.gitignore` only.
```

---

## 10. Files to Add (Checklist)

- [ ] `AGENTS.md` (optional pointer; states canonical is `.cursor/`)
- [ ] `.claude/agents/` → symlink to `.cursor/agents/`
- [ ] `.codex/agents/` → symlink to `.cursor/agents/`
- [ ] `.claude/skills/` → symlink to `.cursor/skills/`
- [ ] `.codex/skills/` → symlink to `.cursor/skills/`
- [ ] `.editorconfig`
- [ ] `.gitignore`
- [ ] `.golangci.yml`
- [ ] `.vscode/tasks.json`
- [ ] `CONTRIBUTING.md`
- [ ] `.gitlab/README.md` → symlink to `../README` (optional)
- [ ] Update `.cursor/rules/workflow.mdc` with canonical-config section
