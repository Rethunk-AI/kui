/**
 * Global keyboard shortcuts for KUI.
 * Listens on document (capture phase). Handles Ctrl/Cmd for cross-platform.
 */
import type { VM } from "./api";

export interface ShortcutContext {
  onEscape: () => void;
  onEnter: () => void;
  onCreateVM: () => void;
  onRefresh: () => void;
  onClone: () => void;
  /** Called when ? or Shift+/ is pressed to show shortcut help. */
  onShowHelp?: () => void;
  /** Returns whether a modal is open (Create/Clone). */
  getHasModalOpen: () => boolean;
  /** Returns whether a VM row is selected. */
  getHasSelection: () => boolean;
  /** Returns the selected VM, or null. */
  getSelectedVM: () => VM | null;
}

function isModKey(ev: KeyboardEvent): boolean {
  return ev.metaKey || ev.ctrlKey;
}

/**
 * Registers global keyboard shortcuts. Returns an unregister function.
 * Use capture phase so Escape is handled before modal handlers.
 */
export function registerShortcuts(ctx: ShortcutContext): () => void {
  const handler = (ev: KeyboardEvent): void => {
    const hasModal = ctx.getHasModalOpen();
    const hasSelection = ctx.getHasSelection();
    const selectedVM = ctx.getSelectedVM();

    if (ev.key === "Escape") {
      ctx.onEscape();
      ev.preventDefault();
      ev.stopPropagation();
      return;
    }

    if (ev.key === "Enter" && hasSelection && selectedVM) {
      ctx.onEnter();
      ev.preventDefault();
      ev.stopPropagation();
      return;
    }

    if (ev.key === "n" && isModKey(ev) && !ev.shiftKey && !hasModal) {
      ctx.onCreateVM();
      ev.preventDefault();
      ev.stopPropagation();
      return;
    }

    if (ev.key === "r" && isModKey(ev) && !ev.shiftKey && !hasModal) {
      ctx.onRefresh();
      ev.preventDefault();
      ev.stopPropagation();
      return;
    }

    if (ev.key === "C" && isModKey(ev) && ev.shiftKey && hasSelection && selectedVM && !hasModal) {
      ctx.onClone();
      ev.preventDefault();
      ev.stopPropagation();
      return;
    }

    if ((ev.key === "?" || (ev.key === "/" && ev.shiftKey)) && !ev.ctrlKey && !ev.metaKey && !ev.altKey) {
      ctx.onShowHelp?.();
      ev.preventDefault();
      ev.stopPropagation();
      return;
    }
  };

  document.addEventListener("keydown", handler, { capture: true });
  return () => document.removeEventListener("keydown", handler, { capture: true });
}
