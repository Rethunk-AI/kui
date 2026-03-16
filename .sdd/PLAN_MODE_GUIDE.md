# PLAN Mode Guide

PLAN mode is a plan-approve-execute workflow: AI presents plans for approval **before** executing file changes.

## What is PLAN Mode?

```
User Command → Analysis (Readonly) → Create Plan → User Approval → Execute → Document
```

**Key benefit:** You review and approve before files are created or modified.

### Cursor 2.1+ Enhancements

- **Interactive questions** — Answer directly in the UI
- **Plan search (⌘+F)** — Search within plans
- **AI Code Reviews** — Automatic review after implementation

## The Four Phases

### Phase 1: Analysis (Readonly)

- AI reads your request, checks existing files, identifies gaps
- Asks clarifying questions
- **No files modified**

### Phase 2: Planning

- AI shows what will be created/modified, structure, reasoning
- You review, request changes, or approve
- **Still no files modified**

### Phase 3: Execution

- AI creates/modifies files as planned
- Follows approved approach

### Phase 4: Documentation

- Updates tracking files, records decisions, timestamps

## Quick Start

```bash
/brief hello-sdd Test PLAN mode with a simple hello world feature
```

1. AI analyzes and may ask questions
2. You answer
3. AI presents plan (structure, files, approach)
4. You approve
5. AI executes

## Command-Specific Examples

### `/brief` — Feature Brief Creation

**Command:** `/brief user-profile-custom Allow users to customize profile with avatar and bio`

**Analysis:** AI checks existing patterns, asks: avatar upload vs library? bio character limit? privacy settings?

**Plan:** Shows brief structure (Problem, Research, Requirements, Approach, Next Actions). Approve to create `specs/active/user-profile-custom/feature-brief.md`.

### `/evolve` — Living Documentation Update

**Command:** `/evolve checkout-flow Discovered we need international address formats`

**Analysis:** AI reads current brief, assesses impact.

**Plan:** Shows BEFORE/AFTER for affected sections, changelog entry. Approve to update.

### `/research` — Pattern Investigation

**Command:** `/research auth-system JWT authentication with session management`

**Plan:** Research strategy (internal codebase, external libraries, security analysis). Approve to create `research.md`.

### `/implement` — Systematic Implementation

**Command:** `/implement user-notifications`

**Plan:** Todo-list preview, phase breakdown, pattern reuse. Approve to create todo-list and begin execution.

### `/upgrade` — Escalating Complexity

**Command:** `/upgrade payment-integration Discovered PCI compliance and multi-provider requirements`

**Plan:** Content mapping from brief to full SDD (research, spec, plan, tasks). Approve to create full suite.

## Tips

- **Be specific** — More detail in commands yields better plans
- **Answer questions thoroughly** — Helps AI create better plans
- **Review plans** — Use ⌘+F to search; verify structure and paths
- **Request changes** — Don't approve if something is off
- **Provide context upfront** — Speeds approval

## Troubleshooting

| Issue | Action |
| ----- | ------ |
| AI didn't ask enough | Provide more context in command |
| Plan too detailed/vague | Request adjustment |
| AI keeps asking | Answer thoroughly; ensures understanding |
| Want to change after approval | Use `/evolve` to update |

## References

- [guidelines.md](.sdd/guidelines.md) — Methodology
- [sdd-system.mdc](.cursor/rules/sdd-system.mdc) — Philosophy and brief vs full SDD
