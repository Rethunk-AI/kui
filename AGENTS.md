# KUI — AI/LLM Instructions

**Canonical configuration:** `.cursor/rules/`, `.cursor/agents/`, `.cursor/skills/`. Edit only those.

**Do not edit:** `.claude/`, `.codex/` — they are symlinks to `.cursor/`. See [docs/project-config-canonical.md](docs/project-config-canonical.md) for the full mapping.

## Summary

- Workflow: planner → developer → verifier. See `.cursor/rules/workflow.mdc`.
- Greenfield: no migrations. See `.cursor/rules/greenfield.mdc`.
- Go: see `.cursor/rules/go-standards.mdc`.
- Canonical config: see `.cursor/rules/canonical-config.mdc`.
