import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderShortcutHelp } from "./ShortcutHelp";

describe("ShortcutHelp", () => {
  let container: HTMLElement;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
  });

  it("renders nothing when not visible", () => {
    renderShortcutHelp(container, { visible: false, onClose: () => {} });
    expect(container.querySelector(".shortcut-help-overlay")).toBeFalsy();
  });

  it("renders overlay when visible", () => {
    renderShortcutHelp(container, { visible: true, onClose: () => {} });
    expect(container.querySelector(".shortcut-help-overlay")).toBeTruthy();
    expect(container.querySelector(".shortcut-help-panel")).toBeTruthy();
    expect(container.querySelector("#shortcut-help-title")?.textContent).toBe("Keyboard shortcuts");
  });

  it("close button calls onClose", () => {
    const onClose = vi.fn();
    renderShortcutHelp(container, { visible: true, onClose });
    (container.querySelector(".shortcut-help__close") as HTMLButtonElement)?.click();
    expect(onClose).toHaveBeenCalled();
  });

  it("Escape calls onClose", () => {
    const onClose = vi.fn();
    renderShortcutHelp(container, { visible: true, onClose });
    const overlay = container.querySelector(".shortcut-help-overlay");
    overlay?.dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true })
    );
    expect(onClose).toHaveBeenCalled();
  });

  it("click outside calls onClose", () => {
    const onClose = vi.fn();
    renderShortcutHelp(container, { visible: true, onClose });
    const overlay = container.querySelector(".shortcut-help-overlay");
    overlay?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    expect(onClose).toHaveBeenCalled();
  });
});
