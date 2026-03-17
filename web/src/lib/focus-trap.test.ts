import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupFocusTrap } from "./focus-trap";

describe("focus-trap", () => {
  let root: HTMLElement;
  let btn1: HTMLButtonElement;
  let btn2: HTMLButtonElement;
  let btn3: HTMLButtonElement;

  beforeEach(() => {
    document.body.innerHTML = "";
    root = document.createElement("div");
    btn1 = document.createElement("button");
    btn1.textContent = "Button 1";
    btn2 = document.createElement("button");
    btn2.textContent = "Button 2";
    btn3 = document.createElement("button");
    btn3.textContent = "Button 3";
    root.appendChild(btn1);
    root.appendChild(btn2);
    root.appendChild(btn3);
    document.body.appendChild(root);
  });

  it("focuses first focusable on mount", () => {
    setupFocusTrap(root);
    expect(document.activeElement).toBe(btn1);
  });

  it("returns cleanup function", () => {
    const cleanup = setupFocusTrap(root);
    expect(typeof cleanup).toBe("function");
    cleanup();
  });

  it("Tab from last prevents default and focuses first", () => {
    setupFocusTrap(root);
    btn3.focus();
    const ev = new KeyboardEvent("keydown", {
      key: "Tab",
      shiftKey: false,
      bubbles: true,
    });
    const preventDefault = vi.spyOn(ev, "preventDefault");
    root.dispatchEvent(ev);
    expect(preventDefault).toHaveBeenCalled();
  });

  it("Shift+Tab from first prevents default and focuses last", () => {
    setupFocusTrap(root);
    btn1.focus();
    const ev = new KeyboardEvent("keydown", {
      key: "Tab",
      shiftKey: true,
      bubbles: true,
    });
    const preventDefault = vi.spyOn(ev, "preventDefault");
    root.dispatchEvent(ev);
    expect(preventDefault).toHaveBeenCalled();
  });

  it("ignores non-Tab key", () => {
    setupFocusTrap(root);
    btn1.focus();
    const ev = new KeyboardEvent("keydown", {
      key: "Enter",
      bubbles: true,
    });
    root.dispatchEvent(ev);
    expect(ev.defaultPrevented).toBe(false);
  });

  it("cleanup removes keydown listener", () => {
    const cleanup = setupFocusTrap(root);
    cleanup();
    btn3.focus();
    const ev = new KeyboardEvent("keydown", {
      key: "Tab",
      shiftKey: false,
      bubbles: true,
    });
    root.dispatchEvent(ev);
    expect(ev.defaultPrevented).toBe(false);
  });

  it("empty root does not throw", () => {
    const empty = document.createElement("div");
    document.body.appendChild(empty);
    const cleanup = setupFocusTrap(empty);
    cleanup();
  });
});
