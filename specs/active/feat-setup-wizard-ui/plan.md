# Setup wizard UI fixes

## Scope

Fix three UI issues in the setup wizard (vanilla TS + Shadcn tokens):

1. **SSH key visibility** — Conditional: hide for local (`qemu:///` or `qemu+unix:`), show for `qemu+ssh://`
2. **Default host dropdown** — Replace native `<select>` with Shadcn-style Select (trigger button, dropdown panel, keyboard nav, aria)
3. **Host collection modal** — Replace inline host cards with list of chips + add/edit modal

## Implementation order

1. SSH key visibility (simplest)
2. Default host Shadcn Select (isolated)
3. Host collection modal (largest refactor)

## Tasks

### 1. SSH key visibility

- [ ] Add `isLocalUri(uri: string): boolean` — `qemu:///` or `qemu+unix:` prefix
- [ ] In host form (row or modal): show keyfile field only when `!isLocalUri(uri)`
- [ ] Wire URI input change to toggle keyfile visibility
- [ ] For local: never send keyfile to validate; backend ignores it anyway

### 2. Default host Shadcn Select

- [ ] Create Shadcn-style Select: trigger button (looks like input, chevron), dropdown panel, keyboard nav (ArrowUp/Down, Enter, Escape)
- [ ] Use design tokens: `--border`, `--input`, `--radius`, etc.
- [ ] ARIA: `role="combobox"`, `aria-expanded`, `aria-haspopup="listbox"`, `aria-controls`
- [ ] Scope: setup wizard only; no need to generalize HostSelector/InlineHostSelector yet
- [ ] Add CSS classes: `.setup-select`, `.setup-select__trigger`, `.setup-select__panel`, `.setup-select__option`

### 3. Host collection modal

- [ ] Hosts section: list of host chips/cards (host ID + URI summary)
- [ ] "Add host" and "Edit" open same modal with full form (Host ID, URI, keyfile when needed, Validate, Save)
- [ ] Add → modal empty; Edit → modal with values; Save → close, list updates
- [ ] Remove: on list item or in modal footer when editing
- [ ] Reuse modal pattern from CreateVMModal (overlay, `.modal`, focus trap)
- [ ] Setup wizard container: append modal overlay when open; no modal-root (setup runs before main layout)

## Files

- `web/src/components/SetupWizard.ts` — main changes
- `web/src/styles.css` — new classes for Shadcn Select, host chips, host modal
- `web/src/components/SetupWizard.test.ts` — update for new structure

## Verification

- `make all` passes
- `make web-a11y` passes
- Manual: setup flows; keyfile hidden for local, shown for qemu+ssh; default host dropdown Shadcn-like; add/edit host in modal

### Delegation

- Developer: implement all three; run `make all` at completion
- Verifier: trust developer; spot-check plan compliance
- Commit: conventional commits; one per theme (fix, feat, refactor)
