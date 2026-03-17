# feat-a11y — Manual Accessibility Testing

Automated checks (pa11y/axe-core) cover static analysis. These manual tests validate keyboard navigation and screen reader support.

## Prerequisites

- Built web app: `cd web && corepack yarn run build && corepack yarn run preview`
- Or dev server: `cd web && corepack yarn run dev` (requires backend for full functionality)

---

## 1. Keyboard-only navigation

**Goal:** All interactive elements reachable and operable via keyboard.

### Steps

1. **Tab order**
   - Use Tab to move forward through focusable elements.
   - Use Shift+Tab to move backward.
   - Verify logical order (skip link → header → main content → controls).

2. **Skip link**
   - Focus the page (e.g. Tab from address bar).
   - First Tab should focus "Skip to main content".
   - Activate with Enter; focus should move to main content.

3. **Modals**
   - Open Create VM, Clone VM, or Shortcut Help.
   - Tab cycles only within the modal (focus trap).
   - Escape closes the modal and returns focus to trigger.

4. **VM list**
   - Use arrow keys to move selection (if applicable).
   - Enter to open/select.

5. **Forms**
   - Tab to each control; labels and errors should be associated.
   - Required fields and validation errors should be announced.

### Pass criteria

- No keyboard traps.
- Focus visible on all interactive elements.
- Escape exits modals.

---

## 2. Screen reader (NVDA on Windows)

**Goal:** Content and structure announced correctly.

### Setup

- Install [NVDA](https://www.nvaccess.org/).
- Start NVDA (Ctrl+Alt+N).

### Steps

1. **Landmarks**
   - Use NVDA landmark shortcuts (e.g. D for landmarks).
   - Verify main, header, navigation are announced.

2. **Headings**
   - Use H to move by heading.
   - Verify h1 → h2 hierarchy.

3. **Forms**
   - Tab to form controls; labels and errors should be read.
   - Required fields announced as "required".

4. **Live regions**
   - Trigger a toast or alert.
   - Message should be announced without manual navigation.

### Pass criteria

- All interactive elements have accessible names.
- Form errors identified and announced.
- Toasts/alerts announced via live region.

---

## 3. Screen reader (VoiceOver on macOS)

**Goal:** Same as NVDA; verify on macOS.

### Setup

- VoiceOver: Cmd+F5 (or System Settings → Accessibility).

### Steps

1. **Rotor**
   - VO+U for Web rotor.
   - Check headings, landmarks, links.

2. **Navigation**
   - VO+Right/Left to move by element.
   - Verify structure and labels.

3. **Forms and live regions**
   - Same checks as NVDA (labels, errors, toasts).

### Pass criteria

- Same as NVDA section.

---

## Reference

- [WCAG 2.1 AAA](https://www.w3.org/WAI/WCAG21/quickref/?levels=aaa)
- [NVDA User Guide](https://www.nvaccess.org/files/nvda/documentation/userGuide.html)
- [VoiceOver Guide](https://support.apple.com/guide/voiceover/welcome/mac)
