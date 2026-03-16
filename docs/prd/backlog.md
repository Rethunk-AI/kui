# KUI Backlog

Prioritized post-MVP items. Core PRD: [prd.md](../prd.md). Decisions: [decision-log.md](decision-log.md).

---

## v2 — Enhancements (after core works)

| Priority | Item | Rationale |
|----------|------|------------|
| 1 | Keyboard shortcuts for power users | Polish after core flow works; Enter to open, Escape to close, shortcuts for common actions |
| 2 | WCAG 2.1 AAA accessibility | Target full a11y compliance once functional UI is stable |
| 3 | Stuck VM detection and recovery | Libvirt state only; escalating actions: force stop → force destroy → undefine |
| 4 | Orphan bulk claim + conflict resolution | Bulk claim; resolve display name collision, UUID on multiple hosts, claimed-but-host_id-mismatch |
| 5 | Domain XML edit for VM repair | Edit domain XML in UI to fix broken config or missing disk; full vs guided defer to spec |

---

## v3 — Operations (much after core)

| Priority | Item | Rationale |
|----------|------|------------|
| 1 | Backup/restore | User needs to recover KUI DB; defer until core system proven |
| 2 | Import/export | Config and templates portability; follows backup/restore |
