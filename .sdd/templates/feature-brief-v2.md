# {{TASK_ID}} Feature Brief

## 🎯 Context (2min)

**Problem**: {{PROBLEM}}
**Users**: {{USERS}}
**Success**: {{SUCCESS_METRIC}}

## 🔍 Quick Research (15min)

### Existing Patterns

{{#each PATTERNS}}

- {{this.name}} → {{this.usage}} | Reuse: {{this.reusable}}
{{/each}}

### Tech Decision

**Approach**: {{CHOSEN_APPROACH}}
**Why**: {{RATIONALE}}
**Avoid**: {{REJECTED_OPTIONS}}

## ✅ Requirements (10min)

{{#each REQUIREMENTS}}

- {{this.story}} → {{this.acceptance}}
{{/each}}

## 🏗️ Implementation (5min)

**Components**: {{COMPONENTS}}
**APIs**: {{API_ENDPOINTS}}
**Data**: {{DATA_CHANGES}}

## 📋 Next Actions (2min)

{{#each IMMEDIATE_TASKS}}

- [ ] {{this.task}} ({{this.effort}})
{{/each}}

**Start Coding In**: {{START_TIME}}

---
**Total Planning Time**: ~30min | **Owner**: {{OWNER}} | {{DATE}}

<!-- Living Document - Update as you code -->

## 🔄 Implementation Tracking

**CRITICAL**: Follow the todo-list systematically. Mark items as complete, document blockers, update progress.

### Progress

- [ ] Track completed items here
- [ ] Update daily

### Blockers

- [ ] Document any blockers

**See**: [.sdd/IMPLEMENTATION_GUIDE.md](mdc:.sdd/IMPLEMENTATION_GUIDE.md) for detailed execution rules.
