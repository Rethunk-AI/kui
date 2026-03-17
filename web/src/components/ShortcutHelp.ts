/**
 * Shortcut help overlay. Lists keyboard shortcuts.
 * Spec feat-keyboard-shortcuts Task 6.
 */
import { setupFocusTrap } from "../lib/focus-trap";

export interface ShortcutHelpProps {
  visible: boolean;
  onClose: () => void;
}

const SHORTCUTS: { key: string; description: string }[] = [
  { key: "Enter", description: "Open console" },
  { key: "Escape", description: "Close modal/console" },
  { key: "Ctrl+N", description: "Create VM" },
  { key: "Ctrl+R", description: "Refresh" },
  { key: "Ctrl+Shift+C", description: "Clone" },
  { key: "?", description: "Show this help" },
];

/**
 * Renders the shortcut help overlay into the given container.
 * When visible, Escape or click outside calls onClose.
 */
export function renderShortcutHelp(
  container: HTMLElement,
  props: ShortcutHelpProps
): void {
  container.innerHTML = "";

  if (!props.visible) return;

  const overlay = document.createElement("div");
  overlay.className = "shortcut-help-overlay";
  overlay.setAttribute("role", "dialog");
  overlay.setAttribute("aria-modal", "true");
  overlay.setAttribute("aria-labelledby", "shortcut-help-title");

  const panel = document.createElement("div");
  panel.className = "shortcut-help-panel";

  const header = document.createElement("div");
  header.className = "shortcut-help__header";
  const title = document.createElement("h2");
  title.id = "shortcut-help-title";
  title.textContent = "Keyboard shortcuts";
  header.appendChild(title);
  const closeBtn = document.createElement("button");
  closeBtn.type = "button";
  closeBtn.className = "shortcut-help__close";
  closeBtn.textContent = "×";
  closeBtn.setAttribute("aria-label", "Close");
  header.appendChild(closeBtn);
  panel.appendChild(header);

  const list = document.createElement("dl");
  list.className = "shortcut-help__list";
  for (const { key, description } of SHORTCUTS) {
    const dt = document.createElement("dt");
    dt.className = "shortcut-help__key";
    dt.textContent = key;
    const dd = document.createElement("dd");
    dd.className = "shortcut-help__desc";
    dd.textContent = description;
    list.appendChild(dt);
    list.appendChild(dd);
  }
  panel.appendChild(list);

  overlay.appendChild(panel);

  const cleanupFocusTrap = setupFocusTrap(overlay);
  const wrappedOnClose = (): void => {
    cleanupFocusTrap();
    props.onClose();
  };

  overlay.addEventListener("click", (e) => {
    if (e.target === overlay) wrappedOnClose();
  });

  overlay.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      wrappedOnClose();
    }
  });

  closeBtn.addEventListener("click", wrappedOnClose);

  container.appendChild(overlay);
}
