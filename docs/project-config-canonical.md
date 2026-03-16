# Canonical Configuration

One canonical source per config type. Edit only canonical files; symlinks exist for tool compatibility. See `.cursor/rules/canonical-config.mdc` for the workflow rule.

---

## AI / LLM Instructions

| Item | Canonical | Do not edit |
|------|-----------|-------------|
| Rules | `.cursor/rules/` | — |
| Subagents | `.cursor/agents/` | `.claude/`, `.codex/` (symlinks) |
| Skills | `.cursor/skills/` | `.claude/`, `.codex/` (symlinks) |
| Entry point | `AGENTS.md` | — |

---

## Other Config

| Config | Canonical | Notes |
|--------|-----------|-------|
| Lint | `.golangci.yml` | — |
| Editor | `.editorconfig` | — |
| Git ignore | `.gitignore` | — |
| MR/PR template | `.github/PULL_REQUEST_TEMPLATE.md` | `.gitlab/merge_request_templates/` symlink |
| README | `README.md` | — |
