# KUI Backlog

Prioritized post-MVP items. Core PRD: [prd.md](../prd.md). Decisions: [decision-log.md](decision-log.md).

---

## Completed (see specs)

Former **v2 — Enhancements** items that are implemented; decision-log may still describe them historically as “v2.”

| Item | Spec (plan) |
|------|-------------|
| Keyboard shortcuts for power users | [feat-keyboard-shortcuts](../../specs/done/feat-keyboard-shortcuts/plan.md) |
| WCAG 2.1 AAA accessibility | [feat-a11y](../../specs/done/feat-a11y/plan.md) |
| Stuck VM detection and recovery | [feat-stuck-vm](../../specs/done/feat-stuck-vm/plan.md) |
| Orphan bulk claim + conflict resolution | [feat-orphan-bulk](../../specs/done/feat-orphan-bulk/plan.md) |
| Domain XML edit for VM repair | [feat-domain-xml-edit](../../specs/done/feat-domain-xml-edit/plan.md) |

**Gap closure** (cross-ref only; see [TODO.md](../../TODO.md) for full done-spec list):

- [gap-audit](../../specs/done/gap-audit/plan.md) — umbrella gap closure (incl. later gap IDs per gap-audit narrative)
- [gap-remediation](../../specs/done/gap-remediation/plan.md)
- [gap-401-session-audit](../../specs/done/gap-401-session-audit/plan.md)
- [gap-domain-xml-network](../../specs/done/gap-domain-xml-network/plan.md)
- [gap-template-network](../../specs/done/gap-template-network/plan.md)

---

## v3 — Operations (much after core)

| Priority | Item | Rationale |
|----------|------|------------|
| 1 | Backup/restore | User needs to recover KUI DB; defer until core system proven |
| 2 | Import/export | Config and templates portability; follows backup/restore |
