# feat-a11y — Plan

## Overview

Achieve WCAG 2.1 AAA accessibility for the KUI web UI. Target: full a11y compliance once functional UI is stable. Greenfield: no migration paths, no backwards compatibility.

**References:** `docs/prd/backlog.md` (v2 item 2), W3C WCAG 2.1.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│ WCAG 2.1 AAA — POUR Principles                                            │
├─────────────────────────────────────────────────────────────────────────┤
│ Perceivable    │ Semantic HTML, alt text, 7:1 contrast, no info by color │
│ Operable       │ Full keyboard nav, skip links, focus order, no traps     │
│ Understandable │ Labels, errors, predictable layout, lang attribute      │
│ Robust         │ Valid HTML, ARIA where needed, screen reader tested     │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Scope

### In scope

- **Perceivable:** 7:1 contrast (AAA), text resize 200%, no information by color alone, alt text for icons
- **Operable:** Full keyboard access, skip link, focus visible, no keyboard traps, 44×44px min touch targets (AAA)
- **Understandable:** Form labels, error identification, page language, consistent nav
- **Robust:** Valid HTML5, ARIA roles/names where needed, tested with screen reader (NVDA/VoiceOver)

### Out of scope

- Video/audio captions (KUI has no video)
- Real-time captions for console (defer)
- Custom screen reader announcements beyond ARIA live regions

---

## WCAG 2.1 AAA Checklist (KUI-relevant)

| Criterion | Level | Requirement | KUI Action |
|-----------|-------|-------------|------------|
| 1.4.6 | AAA | Contrast 7:1 (normal text), 4.5:1 (large) | Update CSS variables |
| 2.1.3 | AAA | No keyboard trap | Modal focus trap with Escape exit |
| 2.2.3 | AAA | No time limits (or adjustable) | N/A or make configurable |
| 2.3.3 | AAA | No more than 3 flashes/sec | No animated flashes |
| 2.4.9 | AAA | Link purpose from link text alone | Descriptive link text |
| 2.4.10 | AAA | Section headings | Use h1–h6 hierarchy |
| 2.5.5 | AAA | Target size 44×44px min | Buttons, controls |
| 3.2.5 | AAA | Change on request | No auto-submit; user-initiated |
| 4.1.3 | AA | Status messages | aria-live for toasts/alerts |

---

## Components

| Component | Responsibility |
|-----------|----------------|
| `web/src/styles.css` | CSS variables for contrast; focus-visible; min touch target sizes |
| `web/index.html` | `lang="en"`, skip link target |
| `web/src/main.ts` | Skip link; main landmark; focus management |
| `web/src/components/*.ts` | Semantic HTML; ARIA; labels; roles |
| `web/src/lib/alerts.ts` | `aria-live="polite"` for toast container |
| `web/src/lib/toast.ts` | Toast announcements to screen readers |

---

## Implementation Tasks

### 1. Color and contrast

**File:** `web/src/styles.css`

- Define CSS variables for text/background with 7:1 contrast
- Large text (≥18pt or 14pt bold): 4.5:1 minimum
- Ensure focus outline has 3:1 contrast against background
- No information conveyed by color alone (e.g., status uses icon + text)

### 2. Focus and keyboard

**Files:** All interactive components

- All interactive elements focusable via Tab
- Visible focus ring (`:focus-visible`) — 2px outline, high contrast
- Modal: focus trap; Tab cycles within modal; Escape closes
- VM list: arrow-key navigation (from feat-keyboard-shortcuts)
- No focus trap without Escape exit

### 3. Skip link

**File:** `web/index.html` or `web/src/main.ts`

- Add "Skip to main content" link at top
- Link targets `#main-content` or `main` id
- Visible on focus; hidden otherwise (or always visible per preference)

### 4. Semantic structure

**Files:** `web/src/main.ts`, `web/src/components/*.ts`

- `<main id="main-content">` for primary content
- `<header>`, `<nav>` with `aria-label` where needed
- Heading hierarchy: one `h1` per view; `h2` for sections
- Lists: `<ul>`/`<ol>` for list content
- Buttons: `<button>` with `aria-label` if icon-only

### 5. Form labels and errors

**Files:** `CreateVMModal.ts`, `CloneVMModal.ts`, `FirstRunChecklist.ts`, login form

- Every form control has associated `<label>` or `aria-label`
- Errors: `aria-describedby` or `aria-invalid` + `role="alert"`
- Required fields: `aria-required="true"` or `required`

### 6. Touch targets

**File:** `web/src/styles.css`

- Buttons, links, controls: min 44×44px (AAA 2.5.5)
- Use padding to expand hit area if visual size smaller

### 7. Live regions for alerts

**File:** `web/src/lib/alerts.ts`, `web/src/components/AlertsPanel.ts`

- Alerts container: `aria-live="polite"` (or `assertive` for critical)
- Toast: announce to screen reader when shown
- Avoid `aria-live` on rapidly changing content

### 8. WinBox and console

**File:** `web/src/lib/winbox-adapter.ts`, `web/src/lib/console.ts`

- WinBox title bar: ensure close button has `aria-label="Close"`
- Console iframe/container: `aria-label` describing content
- noVNC/xterm: document that keyboard goes to guest; ensure focus management when opening/closing

### 9. Testing and validation

- Run `axe-core` or `pa11y` in CI
- Manual test: keyboard-only navigation
- Manual test: NVDA (Windows) or VoiceOver (macOS)

---

## File Paths Summary

| Path | Change |
|------|--------|
| `web/index.html` | `lang="en"`, skip link |
| `web/src/styles.css` | Contrast, focus, touch targets |
| `web/src/main.ts` | Skip link, landmarks |
| `web/src/components/CreateVMModal.ts` | Labels, errors, focus trap |
| `web/src/components/CloneVMModal.ts` | Labels, errors, focus trap |
| `web/src/components/FirstRunChecklist.ts` | Semantic structure |
| `web/src/components/VMList.ts` | Roles, labels, keyboard |
| `web/src/components/AlertsPanel.ts` | aria-live |
| `web/src/components/HostSelector.ts` | Labels |
| `web/src/lib/alerts.ts` | Live region |
| `web/src/lib/winbox-adapter.ts` | ARIA for close |

---

## Security

- No security impact; a11y improves usability only

---

## Testing

- **Automated:** `corepack yarn run build` + `npx pa11y http://localhost:...` or axe-core in Playwright
- **Manual:** Keyboard-only; screen reader (NVDA/VoiceOver)
- **Lint:** Consider `eslint-plugin-jsx-a11y` if migrating to JSX; for vanilla TS, manual review

---

## Verification Steps

```bash
cd web && corepack yarn run build
npx pa11y-ci  # or pa11y on built output
```

- [ ] 7:1 contrast for normal text
- [ ] All interactive elements keyboard accessible
- [ ] Skip link works
- [ ] Modal focus trap + Escape
- [ ] Form labels and errors announced
- [ ] 44×44px min touch targets
- [ ] No axe/pa11y violations (AAA ruleset)

---

## Decision Log

| Decision | Alternatives | Rationale |
|----------|--------------|-----------|
| WCAG AAA | AA only | Backlog explicitly targets AAA |
| axe/pa11y | Manual only | Automated baseline; manual for edge cases |
| Vanilla TS | Add a11y plugin | No JSX; manual review sufficient for current size |

---

## Ownership

- **In scope:** `web/` (all frontend)
- **Out of scope:** Backend, API, libvirt

---

## Changelog

- 2025-03-16: Initial plan
