/**
 * Focus trap for modals. Ensures Tab cycles within the trap root.
 * WCAG 2.1.3: No keyboard trap — modal must have Escape exit (handled by caller).
 */

const FOCUSABLE_SELECTOR =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

function getFocusableElements(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(
    (el) => el.offsetParent !== null && !el.hasAttribute("aria-hidden")
  );
}

/**
 * Sets up focus trap on root. Tab/Shift+Tab cycle within focusable elements.
 * Focuses first focusable on mount.
 * Returns cleanup function.
 */
export function setupFocusTrap(root: HTMLElement): () => void {
  const focusables = (): HTMLElement[] => getFocusableElements(root);
  const first = focusables()[0];

  if (first) {
    first.focus();
  }

  const handler = (ev: KeyboardEvent): void => {
    if (ev.key !== "Tab") return;
    const els = focusables();
    if (els.length === 0) return;

    const current = document.activeElement as HTMLElement | null;
    if (!current || !root.contains(current)) return;

    const idx = els.indexOf(current);
    if (idx === -1) return;

    if (ev.shiftKey) {
      if (idx === 0) {
        ev.preventDefault();
        els[els.length - 1].focus();
      }
    } else {
      if (idx === els.length - 1) {
        ev.preventDefault();
        els[0].focus();
      }
    }
  };

  root.addEventListener("keydown", handler);
  return () => root.removeEventListener("keydown", handler);
}
