# feat-keyboard-shortcuts — Plan

## Overview

Add keyboard shortcuts for power users: Enter to open (console), Escape to close (modals/console windows), and shortcuts for common actions (Create VM, Clone, Refresh). Greenfield: no migration paths, no backwards compatibility.

**References:** `docs/prd/backlog.md` (v2 item 1), `docs/prd/decision-log.md` §2 (Frontend).

---

## Architecture

```
┌───────────────────────────────────────────────────────────────────┐
│ Browser (document)                                                │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │ keydown listener (global, capture phase)                     │ │
│  │  - Escape → close focused modal / WinBox                     │ │
│  │  - Enter → open console (when VM row focused)                │ │
│  │  - Ctrl+N / Cmd+N → Create VM                                │ │
│  │  - Ctrl+R / Cmd+R → Refresh                                  │ │
│  │  - Ctrl+Shift+C / Cmd+Shift+C → Clone (when VM row focused)  │ │
│  └──────────────────────────────────────────────────────────────┘ │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐ │
│  │ VMList       │  │ Modals       │  │ WinBox (console)         │ │
│  │ - row focus  │  │ - Escape     │  │ - Escape → close         │ │
│  │ - Enter      │  │   closes     │  │   (via WinBox API)       │ │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘ │
└───────────────────────────────────────────────────────────────────┘
```

---

## Scope

### In scope

- **Enter** — Open console for focused VM row (when VM list has focus and a row is selected)
- **Escape** — Close modal (Create/Clone) or close focused WinBox console window
- **Ctrl+N / Cmd+N** — Open Create VM modal (when main content visible)
- **Ctrl+R / Cmd+R** — Refresh VM list (when main content visible)
- **Ctrl+Shift+C / Cmd+Shift+C** — Open Clone modal for focused VM (when VM row selected)
- Focus management: VM list rows focusable; arrow keys navigate rows
- Shortcut help: `?` or `Shift+?` opens a help overlay listing shortcuts

### Out of scope

- Customizable shortcuts (fixed bindings only)
- Shortcuts inside noVNC/xterm (those capture keyboard)
- Winbox.js internal shortcuts (unchanged)

---

## Components

| Component | Responsibility |
|-----------|----------------|
| `web/src/lib/shortcuts.ts` | Central keydown handler; dispatches to callbacks; registers/unregisters on mount/unmount |
| `web/src/main.ts` | Registers shortcuts with callbacks (openCreateModal, onRefresh, etc.); passes focus/selection state |
| `web/src/components/VMList.ts` | Row focus/selection; `data-selected` or `aria-selected`; arrow-key navigation; exposes `onRowSelect`, `selectedIndex` |
| `web/src/lib/winbox-adapter.ts` | Optional: pass `onclose` or keyboard option to WinBox; Escape handled by WinBox if supported, else by global handler |
| Modals (`CreateVMModal`, `CloneVMModal`) | Trap focus; Escape closes (already has close button; ensure Escape triggers `onClose`) |

---

## Implementation Tasks

### 1. Create shortcuts module

**File:** `web/src/lib/shortcuts.ts`

- Export `registerShortcuts(ctx: ShortcutContext): () => void`
- `ShortcutContext`: `{ onEscape, onEnter, onCreateVM, onRefresh, onClone, hasModalOpen, hasSelection, selectedVM }`
- Listen on `document` keydown (capture phase)
- Prevent default for handled shortcuts
- Return unregister function
- Handle `Ctrl`/`Cmd` for cross-platform (Meta on Mac, Control on Windows/Linux)

### 2. VM list row focus and selection

**File:** `web/src/components/VMList.ts`

- Add `tabindex="0"` to each VM row (or container) for focus
- Add `role="listbox"` and `role="option"` (or `role="grid"` / `role="row"`) per ARIA listbox pattern
- Track `selectedIndex` state; update on ArrowUp/ArrowDown (when list has focus)
- Emit `onRowSelect(vm, index)` when selection changes
- Ensure Enter is handled by parent (shortcuts) when row focused

### 3. Modal Escape handling

**Files:** `web/src/components/CreateVMModal.ts`, `web/src/components/CloneVMModal.ts`

- Add `keydown` listener on overlay: if `key === "Escape"`, call `onClose()`
- Ensure focus is trapped in modal (optional for v2; Escape must work regardless)

### 4. WinBox Escape handling

**File:** `web/src/lib/winbox-adapter.ts`

- Check WinBox API: if it supports Escape natively, no change
- If not: document that global Escape handler will close topmost WinBox; may need to track open WinBox instances and call `.close()` on Escape

### 5. Integrate shortcuts in main.ts

**File:** `web/src/main.ts`

- After `renderMain`, call `registerShortcuts` with context from current state
- Pass `hasModalOpen`: true when `modalContainer` has child
- Pass `hasSelection`, `selectedVM` from VMList
- On unmount (re-render), unregister previous shortcuts

### 6. Shortcut help overlay

**File:** `web/src/components/ShortcutHelp.ts` (new)

- `?` or `Shift+?` toggles overlay
- Overlay lists: Enter, Escape, Ctrl+N, Ctrl+R, Ctrl+Shift+C with descriptions
- Escape or click outside closes overlay

---

## API / Data

No API changes. Frontend-only.

---

## Security

- Shortcuts do not bypass auth; they trigger same actions as buttons
- No new endpoints or credentials

---

## Testing

- **Unit:** `shortcuts.ts` — mock `ShortcutContext`, simulate keydown, assert callbacks invoked
- **Integration:** VMList — arrow keys change selection; Enter triggers callback
- **Manual:** Verify all shortcuts work; no conflicts with browser/OS

---

## Verification Steps

```bash
go build -o bin/ ./cmd/...
cd web && npm run build
```

- [ ] Enter opens console when VM row focused
- [ ] Escape closes Create/Clone modal
- [ ] Escape closes WinBox console window
- [ ] Ctrl+N opens Create VM modal
- [ ] Ctrl+R refreshes VM list
- [ ] Ctrl+Shift+C opens Clone modal when VM row selected
- [ ] `?` opens shortcut help overlay
- [ ] Arrow keys navigate VM rows

---

## Decision Log

| Decision | Alternatives | Rationale |
|----------|--------------|----------|
| Global document listener | Per-component listeners | Single place; easier to avoid conflicts; capture phase for Escape before modals |
| Fixed shortcuts | User-configurable | Simpler; matches backlog "shortcuts for common actions"; configurable in future |
| Cmd on Mac, Ctrl on Win/Linux | Ctrl only | Standard cross-platform pattern |

---

## Ownership

- **In scope:** `web/src/lib/shortcuts.ts`, `web/src/components/VMList.ts`, `web/src/components/ShortcutHelp.ts`, `web/src/main.ts`, `web/src/components/CreateVMModal.ts`, `web/src/components/CloneVMModal.ts`, `web/src/lib/winbox-adapter.ts`
- **Out of scope:** Backend, libvirt, API routes

---

## Changelog

- 2025-03-16: Initial plan
