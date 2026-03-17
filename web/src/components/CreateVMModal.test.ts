import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderCreateVMModal } from "./CreateVMModal";
import type { Host } from "../lib/api";

vi.mock("../lib/api", () => ({
  createVM: vi.fn().mockResolvedValue({}),
  fetchHostPools: vi.fn().mockResolvedValue([{ name: "default", uuid: "u1", state: "running" }]),
  fetchHostNetworks: vi.fn().mockResolvedValue([{ name: "default", uuid: "u1", active: true }]),
  fetchHostPoolVolumes: vi.fn().mockResolvedValue([{ name: "vol1", path: "/path", capacity: 1024 }]),
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

describe("CreateVMModal", () => {
  let container: HTMLElement;
  const hosts: Host[] = [
    { id: "h1", uri: "qemu:///system" },
    { id: "h2", uri: "qemu:///session" },
  ];

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
  });

  it("renders modal", () => {
    renderCreateVMModal(container, {
      hosts,
      defaultHostId: "h1",
      onClose: () => {},
      onSuccess: () => {},
    });
    expect(container.querySelector("#create-vm-modal-title")?.textContent).toBe("Create VM");
  });

  it("close button calls onClose", () => {
    const onClose = vi.fn();
    renderCreateVMModal(container, {
      hosts,
      defaultHostId: "h1",
      onClose,
      onSuccess: () => {},
    });
    (container.querySelector(".modal__close") as HTMLButtonElement)?.click();
    expect(onClose).toHaveBeenCalled();
  });

  it("cancel button calls onClose", () => {
    const onClose = vi.fn();
    renderCreateVMModal(container, {
      hosts,
      defaultHostId: "h1",
      onClose,
      onSuccess: () => {},
    });
    const cancelBtn = Array.from(container.querySelectorAll("button")).find(
      (b) => b.textContent === "Cancel"
    );
    (cancelBtn as HTMLButtonElement)?.click();
    expect(onClose).toHaveBeenCalled();
  });

  it("toggle disk mode shows/hides volume vs size field", async () => {
    renderCreateVMModal(container, {
      hosts,
      defaultHostId: "h1",
      onClose: () => {},
      onSuccess: () => {},
    });
    await vi.waitFor(() => {
      const poolSelect = container.querySelector('select[name="pool"]') as HTMLSelectElement;
      return poolSelect?.options.length > 1;
    });
    const sizeField = container.querySelector(".modal__field--size") as HTMLElement;
    expect(sizeField?.style.display).toBe("none");
    const diskNew = container.querySelector('#disk-new') as HTMLInputElement;
    diskNew.checked = true;
    diskNew.dispatchEvent(new Event("change"));
    expect(sizeField?.style.display).not.toBe("none");
  });

  it("overlay click calls onClose", () => {
    const onClose = vi.fn();
    renderCreateVMModal(container, {
      hosts,
      defaultHostId: "h1",
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
    renderCreateVMModal(container, {
      hosts,
      defaultHostId: "h1",
      onClose,
      onSuccess: () => {},
    });
    const overlay = container.querySelector(".modal-overlay");
    overlay?.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
    expect(onClose).toHaveBeenCalled();
  });

  it("submit without host shows error", async () => {
    renderCreateVMModal(container, {
      hosts: [],
      defaultHostId: null,
      onClose: () => {},
      onSuccess: () => {},
    });
    const form = container.querySelector("form");
    form?.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await vi.waitFor(() => {
      const err = container.querySelector("#create-vm-form-error");
      return err && (err as HTMLElement).textContent?.includes("host");
    });
  });

  it("submit error shows error message", async () => {
    const { createVM } = await import("../lib/api");
    vi.mocked(createVM).mockRejectedValueOnce(new Error("Create failed"));
    renderCreateVMModal(container, {
      hosts,
      defaultHostId: "h1",
      onClose: () => {},
      onSuccess: () => {},
    });
    await vi.waitFor(() => {
      const poolSelect = container.querySelector('select[name="pool"]') as HTMLSelectElement;
      return poolSelect?.options.length > 1;
    });
    const poolSelect = container.querySelector('select[name="pool"]') as HTMLSelectElement;
    poolSelect.value = "default";
    const volumeSelect = container.querySelector('select[name="volume"]') as HTMLSelectElement;
    volumeSelect.value = "vol1";
    const form = container.querySelector("form");
    form?.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await vi.waitFor(() => {
      const err = container.querySelector("#create-vm-form-error");
      return err && (err as HTMLElement).textContent?.includes("Create failed");
    });
  });

  it("renders disk mode toggle and pool/volume/network selects", async () => {
    renderCreateVMModal(container, {
      hosts,
      defaultHostId: "h1",
      onClose: () => {},
      onSuccess: () => {},
    });
    await vi.waitFor(() => {
      const poolSelect = container.querySelector('select[name="pool"]') as HTMLSelectElement;
      return poolSelect?.options.length > 1;
    });
    expect(container.querySelector('#disk-existing')).toBeTruthy();
    expect(container.querySelector('#disk-new')).toBeTruthy();
    expect(container.querySelector('select[name="pool"]')).toBeTruthy();
    expect(container.querySelector('select[name="volume"]')).toBeTruthy();
    expect(container.querySelector('select[name="network"]')).toBeTruthy();
  });
});
