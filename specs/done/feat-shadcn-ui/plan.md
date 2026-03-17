# feat-shadcn-ui — Plan

## Overview

Introduce Shadcn-like design to the KUI web UI while preserving vanilla TypeScript, Vite, and WCAG 2.1 AAA accessibility. Visit every component and page throughout the entire UI.

**Approach:** Tailwind + Shadcn design tokens — add Tailwind CSS, adopt Shadcn CSS variables (colors, radii, shadows), restyle existing vanilla components. No React migration.

---

## Options Evaluated

| Option | Pros | Cons | Verdict |
|--------|------|-----|---------|
| **1. React + Shadcn** | Full Shadcn components; ecosystem | Full rewrite; 10+ components; Winbox integration risk | Rejected — largest scope, highest risk |
| **2. Tailwind + Shadcn tokens** | No framework change; incremental; design fidelity | Manual restyle of each component | **Recommended** |
| **3. Basecoat UI** | Shadcn-like; vanilla; Tailwind; 3.7k stars | Requires Tailwind; class-name mapping; JS for Select/Dropdown | Viable alternative; higher integration effort |
| **4. shadcn-vanilla-js** | Vanilla port of Shadcn | 5 stars, 0 forks; immature (Aug 2024) | Rejected — not production-ready |

---

## Recommendation: Tailwind + Shadcn Design Tokens

**Rationale:**

1. **Minimal risk** — No migration from vanilla TS to React. Existing render functions and imperative DOM APIs remain.
2. **Design fidelity** — Shadcn’s visual language (colors, radii, shadows, typography) is well-defined. Adopt via CSS variables.
3. **WCAG preservation** — We control tokens; map Shadcn variables to AAA-contrast values (7:1 normal text, 3:1 focus).
4. **Phased delivery** — Tokens → layout → components. Each phase ships working UI.
5. **Greenfield** — No migration paths. Define canonical design system and implement it.

**Basecoat** remains a viable alternative if pre-built component markup is preferred; it would require Tailwind + class-name adoption across all components.

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ KUI Web — Vanilla TS + Vite                                                  │
├──────────────────────────────────────────────────────────────────────────────┤
│ styles.css (or Tailwind + base layer)                                        │
│   :root { --background, --foreground, --radius, --ring, ... }  ← Shadcn vars │
│   WCAG AAA overrides: --foreground 7:1 on --background; --ring 3:1           │
├──────────────────────────────────────────────────────────────────────────────┤
│ main.ts → bootstrap, layout shell, login, VM list                            │
│ components/*.ts → render functions, imperative DOM                           │
│ lib/toast.ts, lib/winbox-adapter.ts                                          │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## Component and Page Inventory

### Pages / Views

| Page | File(s) | Current Classes | Changes |
|------|---------|-----------------|---------|
| Setup wizard | `SetupWizard.ts` | `login-form`, `setup-hosts`, `setup-host-row`, `setup-validate-btn`, etc. | Map to Shadcn form/input/button tokens; preserve fieldset/legend |
| Login form | `main.ts` | `login-form`, `login-error` | Same as above |
| VM list (main) | `main.ts`, `VMList.ts` | `vm-list-container`, `vm-list__*`, `vm-list-orphans__*` | Shadcn card/list styling; preserve structure |
| First-run checklist | `FirstRunChecklist.ts` | `first-run-checklist`, `first-run-checklist__actions` | Card + button styling |
| Shortcut help | `ShortcutHelp.ts` | `shortcut-help-overlay`, `shortcut-help-panel`, `shortcut-help__*` | Dialog styling (Shadcn-like) |

### Components

| Component | File | Current Classes | Changes |
|-----------|------|-----------------|---------|
| Layout shell | `main.ts` | `app-layout`, `app-header`, `app-nav`, `app-content` | Shadcn header/nav tokens; border, spacing |
| HostSelector | `HostSelector.ts` | `host-selector`, `host-selector__label`, `host-selector__select` | Select styling (Shadcn input/select) |
| InlineHostSelector | `InlineHostSelector.ts` | `inline-host-selector`, `inline-host-selector__label`, `inline-host-selector__select` | Same |
| CreateVMModal | `CreateVMModal.ts` | `modal-overlay`, `modal`, `modal__header`, `modal__form`, `modal__field`, `modal__label`, `modal__footer`, etc. | Dialog styling; form fields; buttons |
| CloneVMModal | `CloneVMModal.ts` | Same modal pattern | Same |
| DomainXMLEditor | `DomainXMLEditor.ts` | `modal-overlay`, `modal modal--wide`, `modal__textarea` | Dialog + textarea (code-style) |
| AlertsPanel | `AlertsPanel.ts` | `alerts-panel`, `alerts-panel__header`, `alerts-panel__item`, etc. | Sidebar/panel styling; status borders |
| VMList | `VMList.ts` | `vm-list__*`, `vm-list__btn`, `vm-list__status`, `vm-list-orphans__*` | List, badge, button variants |
| Toast | `lib/toast.ts` | `toast-container`, `toast`, `toast--info`, `toast--success`, `toast--warn` | Shadcn toast styling |
| ShortcutHelp | `ShortcutHelp.ts` | `shortcut-help-overlay`, `shortcut-help-panel`, `shortcut-help__*` | Dialog styling |

### Out of Scope

- **Winbox.js internals** — Third-party; no restyling of Winbox.js UI
- **noVNC / xterm skins** — Separate concern; console windows use their own styling

---

## Design Token Mapping (Shadcn → KUI, WCAG AAA)

Shadcn uses `--background`, `--foreground`, `--muted`, `--muted-foreground`, `--border`, `--input`, `--ring`, `--radius`, etc. KUI must preserve 7:1 contrast for normal text and 3:1 for focus.

| Shadcn Variable | KUI Mapping (dark theme) | WCAG Note |
|-----------------|-------------------------|-----------|
| `--background` | `#1a1a1a` (--kui-bg) | Base |
| `--foreground` | `#e0e0e0` (13:1 on bg) | 7:1 AAA |
| `--muted` | `rgba(255,255,255,0.08)` | Secondary surfaces |
| `--muted-foreground` | `#b0b0b0` (8:1) | 7:1 AAA secondary text |
| `--border` | `rgba(255,255,255,0.15)` | 3:1 min |
| `--input` | `rgba(255,255,255,0.08)` | Input bg |
| `--ring` | `#e0e0e0` | 3:1 focus |
| `--radius` | `0.5rem` (8px) | Shadcn default |
| `--destructive` | `#ec8a8a` (7.1:1) | Error/destructive |
| `--success` | `#81c784` (8.6:1) | Success |
| `--warning` | `#ffb74d` (10:1) | Warning |

---

## Class Name Mapping (Before → After)

Preserve BEM-like structure; swap visual tokens. Example mappings:

| Before | After | Notes |
|--------|-------|-------|
| `--kui-bg`, `--kui-fg` | `--background`, `--foreground` | Alias or replace |
| `vm-list__btn` | `vm-list__btn` + Tailwind `btn`-like utilities or token-based styles | Same class; new styles |
| `modal` | `modal` + Shadcn dialog tokens | Border, radius, shadow |
| `modal__header` | `modal__header` | Use `--border`, `--muted` |
| `vm-list__status--running` | Same | Use `--success` token |
| `vm-list__status--blocked` | Same | Use `--destructive` token |
| `toast--info` | Same | Use Shadcn primary/info |
| `login-form` | `login-form` | Use `input`, `btn` tokens |

Structure (BEM names) stays; only the CSS values change. Optionally introduce Tailwind utility classes where it simplifies (e.g. `rounded-lg`, `border`).

---

## Phased Approach

### Phase 1: Design Tokens and Tailwind Setup

1. Add Tailwind CSS to `web/`.
2. Add Shadcn-compatible CSS variables to `:root`, with WCAG AAA overrides.
3. Keep existing `--kui-*` as aliases or migrate to `--background` etc.
4. Verify: `pa11y` passes; contrast unchanged.

### Phase 2: Layout Shell

1. Restyle `app-layout`, `app-header`, `app-nav`, `app-content` using Shadcn tokens.
2. Restyle `app-header` border, nav spacing.
3. Verify: layout renders correctly; a11y unchanged.

### Phase 3: Core Components (VM List, Modals)

1. **VMList** — `vm-list-container`, `vm-list__*`, `vm-list-orphans__*`; badges, buttons, rows.
2. **CreateVMModal**, **CloneVMModal** — `modal__*`, form fields, buttons.
3. **DomainXMLEditor** — modal, textarea.
4. Verify: `go test`, `make web-a11y`.

### Phase 4: Remaining Components

1. **SetupWizard** — form, fieldsets, host rows.
2. **Login form** — `main.ts` login block.
3. **FirstRunChecklist** — card, buttons.
4. **ShortcutHelp** — dialog overlay.
5. **AlertsPanel** — sidebar, items, dismiss.
6. **HostSelector**, **InlineHostSelector** — select styling.
7. **Toast** — `lib/toast.ts`; container and variants.
8. Verify: full `make all`; `make web-a11y`.

### Phase 5: Polish and Cleanup

1. Remove redundant `--kui-*` if fully migrated.
2. Consolidate duplicate styles (e.g. modal vs shortcut-help).
3. Final a11y pass.

---

## Data Models

No schema changes. UI-only.

---

## API Contracts

No API changes. UI-only.

---

## Security

- No new auth surface; UI-only.
- Ensure no secrets in CSS or tokens.

---

## Testing Strategy

| Test | Command | Notes |
|------|---------|-------|
| Unit | `go test ./...` | Backend unchanged |
| Web unit | `cd web && yarn test` | Component render tests |
| Build | `make all` | Build + test + vet |
| A11y | `make web-a11y` | pa11y + axe |
| Visual | Manual | Compare before/after screenshots |

---

## Verification Commands

```bash
# Full verification
make all
make web-a11y

# Incremental
cd web && yarn run build
cd web && yarn run test
```

---

## Rollout

- Config: None. Design tokens are compile-time.
- Deployment: Rebuild `web/`; deploy as usual.
- No migrations, backfill, or backwards-compatibility.

---

## Decision Log

| Decision | Alternatives | Rationale |
|----------|--------------|-----------|
| Tailwind + Shadcn tokens | React+Shadcn, Basecoat, shadcn-vanilla-js | Lowest risk; no framework change; preserves vanilla TS |
| WCAG AAA overrides | Use Shadcn defaults | Shadcn defaults may not meet AAA; KUI has explicit contrast requirements |
| Preserve BEM structure | Adopt Basecoat class names | Less refactor; incremental; easier to trace |
| Phased by layout → components | Big-bang | Reduces risk; each phase shippable |

---

## Ownership Boundaries

### In Scope

- `web/package.json` — add Tailwind
- `web/src/styles.css` — tokens, restyle
- `web/src/main.ts` — layout classes
- `web/src/components/*.ts` — class names where styling changes
- `web/src/lib/toast.ts` — toast classes
- `web/vite.config.ts` — Tailwind plugin if needed
- `web/postcss.config.js` — if new

### Out of Scope

- `web/src/lib/winbox-adapter.ts` — Winbox.js internals
- noVNC / xterm skins
- Backend (`internal/`, `cmd/`)

---

## Tasks (Dependency Order)

1. **T1** Add Tailwind + PostCSS to `web/`; minimal config.
2. **T2** Add Shadcn design tokens to `:root` with WCAG AAA overrides; migrate `--kui-*` or alias.
3. **T3** Restyle layout shell (`app-layout`, `app-header`, `app-nav`, `app-content`).
4. **T4** Restyle VMList (`vm-list__*`, `vm-list-orphans__*`).
5. **T5** Restyle CreateVMModal, CloneVMModal, DomainXMLEditor (modals).
6. **T6** Restyle SetupWizard, login form.
7. **T7** Restyle FirstRunChecklist.
8. **T8** Restyle ShortcutHelp.
9. **T9** Restyle AlertsPanel.
10. **T10** Restyle HostSelector, InlineHostSelector.
11. **T11** Restyle Toast.
12. **T12** Polish: remove redundant tokens; consolidate; final a11y check.

**Dependencies:** T1 → T2 → T3; T3 → T4; T3 → T5; T3 → T6–T11; T4–T11 → T12.

---

## Assumptions

- WCAG 2.1 AAA remains required (feat-a11y).
- Vanilla TS + Vite stack is fixed; no React migration.
- Winbox.js, noVNC, xterm are out of scope.

---

## Open Questions

- Prefer Tailwind utility classes in component TS (e.g. `classList.add('rounded-lg')`) vs pure CSS? Recommendation: prefer CSS tokens in `styles.css`; add Tailwind utilities only where they simplify.
- Light theme? Out of scope for initial plan; tokens can support light mode later.

---

## Changelog

- 2025-03-17: Initial plan
