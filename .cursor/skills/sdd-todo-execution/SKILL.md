---
name: sdd-todo-execution
description: Executes plan task lists sequentially with no skipping, explicit progress, and BLOCKED/MODIFIED notation. Use when implementing from plan.md, executing todo-lists, or following plan tasks. Ensures every item gets explicit action (complete, block, or modify).
---

# SDD Todo Execution

Execute task lists from `plan.md` or `tasks-compact.md` systematically. The todo-list is mandatory, not optional.

## Rules

1. **Read the entire todo-list** before starting.
2. **Execute in order** — respect dependencies; do not skip.
3. **Mark completion** — Update `- [ ]` to `- [x]` as you complete each item.
4. **Document blockers** — If you cannot complete an item: `- [ ] Task (BLOCKED: reason)`.
5. **Document modifications** — If approach changes: `- [x] Task (MODIFIED: new approach)`.
6. **Never skip silently** — Every item requires explicit action (complete, block, or modify).

## Execution Pattern

```
1. Read todo-list
2. Identify next uncompleted task
3. Execute the task completely
4. Update checkbox: - [ ] → - [x]
5. Document any issues or changes
6. Repeat until all tasks complete
```

## Progress Format

```markdown
## Current Status
**Phase**: 2/4
**Progress**: 12/35 tasks complete
**Blockers**: 2 items waiting for X
**Next**: Task description
```

## References

- `.sdd/templates/plan-compact.md` — Plan structure
- `.sdd/templates/tasks-compact.md` — Task template
- `.cursor/rules/workflow.mdc` — planner → developer → verifier
