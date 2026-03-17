import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderVMList } from "./VMList";
import type { VMsResponse, VM } from "../lib/api";

vi.mock("../lib/api", () => ({
  claimVM: vi.fn().mockResolvedValue({}),
  recoverVM: vi.fn().mockResolvedValue({}),
  bulkClaimOrphans: vi.fn().mockResolvedValue({ claimed: [], conflicts: [] }),
  bulkDestroyOrphans: vi.fn().mockResolvedValue({ destroyed: [], failed: [] }),
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

vi.mock("../lib/toast", () => ({
  showToast: vi.fn(),
}));

const makeVM = (overrides: Partial<VM> = {}): VM => ({
  host_id: "h1",
  libvirt_uuid: "u1",
  display_name: "Test VM",
  claimed: true,
  status: "running",
  console_preference: null,
  last_access: "2024-01-15T10:00:00Z",
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-15T10:00:00Z",
  ...overrides,
});

describe("VMList", () => {
  let container: HTMLElement;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
  });

  it("renders empty state", () => {
    const data: VMsResponse = {
      vms: [],
      hosts: {},
      orphans: [],
    };
    renderVMList(container, { data, onRefresh: () => {} });
    expect(container.querySelector(".vm-list__empty")?.textContent).toBe("No VMs");
  });

  it("renders Create VM button when onOpenCreateModal provided", () => {
    const data: VMsResponse = {
      vms: [makeVM()],
      hosts: { h1: "running" },
      orphans: [],
    };
    const onOpenCreateModal = vi.fn();
    renderVMList(container, {
      data,
      onRefresh: () => {},
      onOpenCreateModal,
    });
    const btn = container.querySelector(".vm-list__btn--create");
    expect(btn).toBeTruthy();
    (btn as HTMLButtonElement)?.click();
    expect(onOpenCreateModal).toHaveBeenCalled();
  });

  it("renders VM rows with status", () => {
    const data: VMsResponse = {
      vms: [makeVM({ display_name: "My VM", status: "running" })],
      hosts: { h1: "running" },
      orphans: [],
    };
    renderVMList(container, { data, onRefresh: () => {} });
    expect(container.querySelector(".vm-list__name")?.textContent).toBe("My VM");
    expect(container.querySelector(".vm-list__status")?.textContent).toBe("running");
  });

  it("onRowSelect called with first VM", () => {
    const data: VMsResponse = {
      vms: [makeVM({ display_name: "First" })],
      hosts: { h1: "running" },
      orphans: [],
    };
    const onRowSelect = vi.fn();
    renderVMList(container, {
      data,
      onRefresh: () => {},
      onRowSelect,
    });
    expect(onRowSelect).toHaveBeenCalledWith(
      expect.objectContaining({ display_name: "First" }),
      0
    );
  });

  it("Console button calls onOpenConsole", () => {
    const vm = makeVM();
    const data: VMsResponse = {
      vms: [vm],
      hosts: { h1: "running" },
      orphans: [],
    };
    const onOpenConsole = vi.fn();
    renderVMList(container, {
      data,
      onRefresh: () => {},
      onOpenConsole,
    });
    const consoleBtn = container.querySelector(".vm-list__btn--console");
    (consoleBtn as HTMLButtonElement)?.click();
    expect(onOpenConsole).toHaveBeenCalledWith(vm);
  });

  it("Clone button calls onOpenCloneModal", () => {
    const vm = makeVM();
    const data: VMsResponse = {
      vms: [vm],
      hosts: { h1: "running" },
      orphans: [],
    };
    const onOpenCloneModal = vi.fn();
    renderVMList(container, {
      data,
      onRefresh: () => {},
      onOpenCloneModal,
    });
    const cloneBtn = container.querySelector(".vm-list__btn--clone");
    (cloneBtn as HTMLButtonElement)?.click();
    expect(onOpenCloneModal).toHaveBeenCalledWith(vm);
  });

  it("renders orphans section", () => {
    const data: VMsResponse = {
      vms: [],
      hosts: {},
      orphans: [
        { host_id: "h1", libvirt_uuid: "u1", name: "Orphan VM" },
      ],
    };
    renderVMList(container, { data, onRefresh: () => {} });
    expect(container.querySelector(".vm-list-orphans")).toBeTruthy();
    expect(container.querySelector(".vm-list-orphans__name")?.textContent).toBe("Orphan VM");
  });

  it("Claim button error re-enables button", async () => {
    const { claimVM } = await import("../lib/api");
    vi.mocked(claimVM).mockRejectedValueOnce(new Error("Claim failed"));
    const onRefresh = vi.fn();
    const data: VMsResponse = {
      vms: [],
      hosts: {},
      orphans: [{ host_id: "h1", libvirt_uuid: "u1", name: "Orphan" }],
    };
    renderVMList(container, { data, onRefresh });
    const claimBtn = container.querySelector(".vm-list__btn--claim") as HTMLButtonElement;
    claimBtn?.click();
    await vi.waitFor(() => {
      expect(claimBtn.disabled).toBe(false);
    });
    expect(onRefresh).not.toHaveBeenCalled();
  });

  it("Claim button calls claimVM and onRefresh", async () => {
    const { claimVM } = await import("../lib/api");
    const onRefresh = vi.fn();
    const data: VMsResponse = {
      vms: [],
      hosts: {},
      orphans: [
        { host_id: "h1", libvirt_uuid: "u1", name: "Orphan" },
      ],
    };
    renderVMList(container, { data, onRefresh });
    const claimBtn = container.querySelector(".vm-list__btn--claim");
    (claimBtn as HTMLButtonElement)?.click();
    await vi.waitFor(() => {
      expect(claimVM).toHaveBeenCalledWith("h1", "u1", "Orphan");
      expect(onRefresh).toHaveBeenCalled();
    });
  });

  it("Recover button error re-enables button", async () => {
    const { recoverVM } = await import("../lib/api");
    vi.mocked(recoverVM).mockRejectedValueOnce(new Error("Recover failed"));
    const onRefresh = vi.fn();
    const data: VMsResponse = {
      vms: [makeVM({ status: "crashed" })],
      hosts: { h1: "running" },
      orphans: [],
    };
    renderVMList(container, { data, onRefresh });
    const recoverBtn = container.querySelector(".vm-list__btn--recover") as HTMLButtonElement;
    recoverBtn?.click();
    await vi.waitFor(() => {
      expect(recoverBtn.disabled).toBe(false);
    });
    expect(onRefresh).not.toHaveBeenCalled();
  });

  it("Recover button for stuck VM", async () => {
    const { recoverVM } = await import("../lib/api");
    const onRefresh = vi.fn();
    const data: VMsResponse = {
      vms: [makeVM({ status: "crashed" })],
      hosts: { h1: "running" },
      orphans: [],
    };
    renderVMList(container, { data, onRefresh });
    const recoverBtn = container.querySelector(".vm-list__btn--recover");
    expect(recoverBtn).toBeTruthy();
    (recoverBtn as HTMLButtonElement)?.click();
    await vi.waitFor(() => {
      expect(recoverVM).toHaveBeenCalledWith("h1", "u1");
      expect(onRefresh).toHaveBeenCalled();
    });
  });

  it("orphans toggle expands/collapses", () => {
    const data: VMsResponse = {
      vms: [],
      hosts: {},
      orphans: [{ host_id: "h1", libvirt_uuid: "u1", name: "O" }],
    };
    renderVMList(container, { data, onRefresh: () => {} });
    const toggle = container.querySelector(".vm-list-orphans__toggle");
    expect(toggle?.getAttribute("aria-expanded")).toBe("true");
    (toggle as HTMLButtonElement)?.click();
    expect(toggle?.getAttribute("aria-expanded")).toBe("false");
  });
});
