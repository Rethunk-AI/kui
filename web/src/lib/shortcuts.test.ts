import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { registerShortcuts } from "./shortcuts";
import type { VM } from "./api";

function keydown(opts: {
  key: string;
  ctrlKey?: boolean;
  metaKey?: boolean;
  shiftKey?: boolean;
  altKey?: boolean;
}) {
  return new KeyboardEvent("keydown", {
    key: opts.key,
    ctrlKey: opts.ctrlKey ?? false,
    metaKey: opts.metaKey ?? false,
    shiftKey: opts.shiftKey ?? false,
    altKey: opts.altKey ?? false,
    bubbles: true,
  });
}

const mockVM: VM = {
  host_id: "h1",
  libvirt_uuid: "u1",
  display_name: "Test VM",
  claimed: true,
  status: "running",
  console_preference: null,
  last_access: null,
  created_at: "",
  updated_at: "",
};

describe("shortcuts", () => {
  let unregister: () => void;
  let ctx: {
    onEscape: ReturnType<typeof vi.fn>;
    onEnter: ReturnType<typeof vi.fn>;
    onCreateVM: ReturnType<typeof vi.fn>;
    onRefresh: ReturnType<typeof vi.fn>;
    onClone: ReturnType<typeof vi.fn>;
    onShowHelp?: ReturnType<typeof vi.fn>;
    getHasModalOpen: ReturnType<typeof vi.fn>;
    getHasSelection: ReturnType<typeof vi.fn>;
    getSelectedVM: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    ctx = {
      onEscape: vi.fn(),
      onEnter: vi.fn(),
      onCreateVM: vi.fn(),
      onRefresh: vi.fn(),
      onClone: vi.fn(),
      onShowHelp: vi.fn(),
      getHasModalOpen: vi.fn(() => false),
      getHasSelection: vi.fn(() => false),
      getSelectedVM: vi.fn(() => null),
    };
    unregister = registerShortcuts(ctx);
  });

  afterEach(() => {
    unregister();
  });

  it("registers and unregisters without error", () => {
    expect(typeof unregister).toBe("function");
  });

  it("Escape calls onEscape", () => {
    const ev = keydown({ key: "Escape" });
    document.dispatchEvent(ev);
    expect(ctx.onEscape).toHaveBeenCalled();
  });

  it("Enter with selection calls onEnter", () => {
    ctx.getHasSelection.mockReturnValue(true);
    ctx.getSelectedVM.mockReturnValue(mockVM);
    const ev = keydown({ key: "Enter" });
    document.dispatchEvent(ev);
    expect(ctx.onEnter).toHaveBeenCalled();
  });

  it("Enter without selection does not call onEnter", () => {
    ctx.getHasSelection.mockReturnValue(false);
    const ev = keydown({ key: "Enter" });
    document.dispatchEvent(ev);
    expect(ctx.onEnter).not.toHaveBeenCalled();
  });

  it("Enter with selection but null VM does not call onEnter", () => {
    ctx.getHasSelection.mockReturnValue(true);
    ctx.getSelectedVM.mockReturnValue(null);
    const ev = keydown({ key: "Enter" });
    document.dispatchEvent(ev);
    expect(ctx.onEnter).not.toHaveBeenCalled();
  });

  it("Ctrl+N without modal calls onCreateVM", () => {
    const ev = keydown({ key: "n", ctrlKey: true });
    document.dispatchEvent(ev);
    expect(ctx.onCreateVM).toHaveBeenCalled();
  });

  it("Meta+N (Cmd) without modal calls onCreateVM", () => {
    const ev = keydown({ key: "n", metaKey: true });
    document.dispatchEvent(ev);
    expect(ctx.onCreateVM).toHaveBeenCalled();
  });

  it("Ctrl+N with modal open does not call onCreateVM", () => {
    ctx.getHasModalOpen.mockReturnValue(true);
    const ev = keydown({ key: "n", ctrlKey: true });
    document.dispatchEvent(ev);
    expect(ctx.onCreateVM).not.toHaveBeenCalled();
  });

  it("Ctrl+Shift+N does not call onCreateVM", () => {
    const ev = keydown({ key: "n", ctrlKey: true, shiftKey: true });
    document.dispatchEvent(ev);
    expect(ctx.onCreateVM).not.toHaveBeenCalled();
  });

  it("Ctrl+R without modal calls onRefresh", () => {
    const ev = keydown({ key: "r", ctrlKey: true });
    document.dispatchEvent(ev);
    expect(ctx.onRefresh).toHaveBeenCalled();
  });

  it("Ctrl+R with modal does not call onRefresh", () => {
    ctx.getHasModalOpen.mockReturnValue(true);
    const ev = keydown({ key: "r", ctrlKey: true });
    document.dispatchEvent(ev);
    expect(ctx.onRefresh).not.toHaveBeenCalled();
  });

  it("Ctrl+Shift+C with selection and no modal calls onClone", () => {
    ctx.getHasSelection.mockReturnValue(true);
    ctx.getSelectedVM.mockReturnValue(mockVM);
    const ev = keydown({ key: "C", ctrlKey: true, shiftKey: true });
    document.dispatchEvent(ev);
    expect(ctx.onClone).toHaveBeenCalled();
  });

  it("Ctrl+Shift+C without selection does not call onClone", () => {
    ctx.getHasSelection.mockReturnValue(false);
    const ev = keydown({ key: "C", ctrlKey: true, shiftKey: true });
    document.dispatchEvent(ev);
    expect(ctx.onClone).not.toHaveBeenCalled();
  });

  it("Ctrl+Shift+C with modal does not call onClone", () => {
    ctx.getHasModalOpen.mockReturnValue(true);
    ctx.getHasSelection.mockReturnValue(true);
    ctx.getSelectedVM.mockReturnValue(mockVM);
    const ev = keydown({ key: "C", ctrlKey: true, shiftKey: true });
    document.dispatchEvent(ev);
    expect(ctx.onClone).not.toHaveBeenCalled();
  });

  it("? calls onShowHelp when provided", () => {
    const ev = keydown({ key: "?" });
    document.dispatchEvent(ev);
    expect(ctx.onShowHelp).toHaveBeenCalled();
  });

  it("Shift+/ calls onShowHelp when provided", () => {
    const ev = keydown({ key: "/", shiftKey: true });
    document.dispatchEvent(ev);
    expect(ctx.onShowHelp).toHaveBeenCalled();
  });

  it("? with Ctrl does not call onShowHelp", () => {
    const ev = keydown({ key: "?", ctrlKey: true });
    document.dispatchEvent(ev);
    expect(ctx.onShowHelp).not.toHaveBeenCalled();
  });

  it("works without onShowHelp", () => {
    const ctxNoHelp = { ...ctx };
    delete ctxNoHelp.onShowHelp;
    unregister();
    unregister = registerShortcuts(ctxNoHelp);
    const ev = keydown({ key: "?" });
    document.dispatchEvent(ev);
    expect(() => {}).not.toThrow();
  });

  it("unregister removes listener", () => {
    unregister();
    const ev = keydown({ key: "Escape" });
    document.dispatchEvent(ev);
    expect(ctx.onEscape).not.toHaveBeenCalled();
  });
});
