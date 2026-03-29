import { describe, it, expect, beforeEach, vi } from "vitest";
import { openWinBox, closeTopmostWinBox } from "./winbox-adapter";

const mockClose = vi.fn();
const mockInstance = {
  close: mockClose,
  window: null as HTMLElement | null,
};

function WinBoxMock(_title: string, _opts: unknown) {
  mockInstance.window = document.createElement("div");
  const closeBtn = document.createElement("button");
  closeBtn.className = "wb-close";
  mockInstance.window.appendChild(closeBtn);
  return mockInstance;
}

vi.mock("winbox/src/js/winbox.js", () => ({
  default: vi.fn(WinBoxMock),
}));

describe("winbox-adapter", () => {
  beforeEach(() => {
    mockClose.mockClear();
    mockInstance.window = null;
  });

  it("closeTopmostWinBox returns false when no WinBox open", () => {
    while (closeTopmostWinBox()) {
      /* drain */
    }
    expect(closeTopmostWinBox()).toBe(false);
  });

  it("openWinBox creates instance with mount and title", () => {
    const mount = document.createElement("div");
    mount.textContent = "content";
    const instance = openWinBox("Test Title", mount);
    expect(instance).toBe(mockInstance);
  });

  it("closeTopmostWinBox closes and returns true when one is open", () => {
    const mount = document.createElement("div");
    openWinBox("Test", mount);
    expect(closeTopmostWinBox()).toBe(true);
    expect(mockClose).toHaveBeenCalled();
  });

  it("setCloseButtonAriaLabel sets aria-label on close button", () => {
    const mount = document.createElement("div");
    openWinBox("Test", mount);
    expect(mockInstance.window?.querySelector(".wb-close")).toBeTruthy();
    const closeBtn = mockInstance.window?.querySelector(".wb-close");
    expect((closeBtn as HTMLElement)?.getAttribute("aria-label")).toBe("Close");
  });
});
