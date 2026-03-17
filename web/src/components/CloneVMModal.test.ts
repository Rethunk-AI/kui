import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderCloneVMModal } from "./CloneVMModal";
import type { Host, VM } from "../lib/api";

vi.mock("../lib/api", () => ({
  cloneVM: vi.fn().mockResolvedValue({}),
  fetchHostPools: vi.fn().mockResolvedValue([{ name: "default", uuid: "u1", state: "running" }]),
  ApiError: class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

vi.mock("../lib/alerts", () => ({
  addAlert: vi.fn(),
}));

describe("CloneVMModal", () => {
  let container: HTMLElement;
  const sourceVM: VM = {
    host_id: "h1",
    libvirt_uuid: "u1",
    display_name: "Source VM",
    claimed: true,
    status: "running",
    console_preference: null,
    last_access: null,
    created_at: "",
    updated_at: "",
  };
  const hosts: Host[] = [
    { id: "h1", uri: "qemu:///system" },
    { id: "h2", uri: "qemu:///session" },
  ];

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
  });

  it("renders modal with source info", () => {
    renderCloneVMModal(container, {
      sourceVM,
      hosts,
      defaultHostId: "h2",
      onClose: () => {},
      onSuccess: () => {},
    });
    expect(container.querySelector("#clone-vm-modal-title")?.textContent).toBe("Clone VM");
    expect(container.textContent).toContain("Source VM");
  });

  it("close button calls onClose", () => {
    const onClose = vi.fn();
    renderCloneVMModal(container, {
      sourceVM,
      hosts,
      defaultHostId: "h2",
      onClose,
      onSuccess: () => {},
    });
    (container.querySelector(".modal__close") as HTMLButtonElement)?.click();
    expect(onClose).toHaveBeenCalled();
  });

  it("cancel button calls onClose", () => {
    const onClose = vi.fn();
    renderCloneVMModal(container, {
      sourceVM,
      hosts,
      defaultHostId: "h2",
      onClose,
      onSuccess: () => {},
    });
    const cancelBtn = Array.from(container.querySelectorAll("button")).find(
      (b) => b.textContent === "Cancel"
    );
    (cancelBtn as HTMLButtonElement)?.click();
    expect(onClose).toHaveBeenCalled();
  });

  it("overlay click calls onClose", () => {
    const onClose = vi.fn();
    renderCloneVMModal(container, {
      sourceVM,
      hosts,
      defaultHostId: "h2",
      onClose,
      onSuccess: () => {},
    });
    const overlay = container.querySelector(".modal-overlay");
    if (overlay) {
      (overlay as HTMLElement).dispatchEvent(new MouseEvent("click", { bubbles: true }));
    }
    expect(onClose).toHaveBeenCalled();
  });

  it("escape key calls onClose", () => {
    const onClose = vi.fn();
    renderCloneVMModal(container, {
      sourceVM,
      hosts,
      defaultHostId: "h2",
      onClose,
      onSuccess: () => {},
    });
    const overlay = container.querySelector(".modal-overlay");
    overlay?.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
    expect(onClose).toHaveBeenCalled();
  });

  it("submit without target host shows error", async () => {
    renderCloneVMModal(container, {
      sourceVM,
      hosts: [],
      defaultHostId: null,
      onClose: () => {},
      onSuccess: () => {},
    });
    const form = container.querySelector("form");
    form?.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await vi.waitFor(() => {
      const err = container.querySelector("#clone-vm-form-error");
      return err && (err as HTMLElement).style.display !== "none";
    });
    expect(container.textContent).toContain("Select a target host");
  });

  it("submit without target pool shows error", async () => {
    renderCloneVMModal(container, {
      sourceVM,
      hosts,
      defaultHostId: "h2",
      onClose: () => {},
      onSuccess: () => {},
    });
    await vi.waitFor(() => {
      const poolSelect = container.querySelector('select[name="target_pool"]') as HTMLSelectElement;
      return poolSelect?.options.length > 1;
    });
    const poolSelect = container.querySelector('select[name="target_pool"]') as HTMLSelectElement;
    poolSelect.value = "";
    poolSelect.dispatchEvent(new Event("change"));
    const form = container.querySelector("form");
    form?.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await vi.waitFor(() => {
      const err = container.querySelector("#clone-vm-form-error");
      return err && (err as HTMLElement).textContent?.includes("target pool");
    });
  });

  it("submit error shows error message", async () => {
    const { cloneVM } = await import("../lib/api");
    vi.mocked(cloneVM).mockRejectedValueOnce(new Error("Clone failed"));
    const onClose = vi.fn();
    renderCloneVMModal(container, {
      sourceVM,
      hosts,
      defaultHostId: "h2",
      onClose,
      onSuccess: () => {},
    });
    await vi.waitFor(() => {
      const poolSelect = container.querySelector('select[name="target_pool"]') as HTMLSelectElement;
      return poolSelect?.options.length > 1;
    });
    const poolSelect = container.querySelector('select[name="target_pool"]') as HTMLSelectElement;
    poolSelect.value = "default";
    const form = container.querySelector("form");
    form?.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await vi.waitFor(() => {
      const err = container.querySelector("#clone-vm-form-error");
      return err && (err as HTMLElement).textContent?.includes("Clone failed");
    });
    expect(onClose).not.toHaveBeenCalled();
  });

  it("submit calls cloneVM and onSuccess", async () => {
    const { cloneVM } = await import("../lib/api");
    const onSuccess = vi.fn();
    const onClose = vi.fn();
    renderCloneVMModal(container, {
      sourceVM,
      hosts,
      defaultHostId: "h2",
      onClose,
      onSuccess,
    });
    await vi.waitFor(() => {
      const poolSelect = container.querySelector('select[name="target_pool"]') as HTMLSelectElement;
      return poolSelect?.options.length > 1;
    });
    const poolSelect = container.querySelector('select[name="target_pool"]') as HTMLSelectElement;
    poolSelect.value = "default";
    poolSelect.dispatchEvent(new Event("change"));
    const nameInput = container.querySelector('input[name="target_name"]') as HTMLInputElement;
    nameInput.value = "clone-vm";
    const form = container.querySelector("form");
    form?.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await vi.waitFor(() => {
      expect(cloneVM).toHaveBeenCalledWith("h1", "u1", {
        target_host_id: "h2",
        target_pool: "default",
        target_name: "clone-vm",
      });
      expect(onSuccess).toHaveBeenCalled();
      expect(onClose).toHaveBeenCalled();
    });
  });
});
