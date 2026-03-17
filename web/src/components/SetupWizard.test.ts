import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderSetupWizard } from "./SetupWizard";

vi.mock("../lib/api", () => ({
  setupComplete: vi.fn().mockResolvedValue(undefined),
  validateHost: vi.fn().mockResolvedValue({ valid: true }),
  provisionHost: vi.fn().mockResolvedValue({}),
  ApiError: class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

describe("SetupWizard", () => {
  let container: HTMLElement;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
  });

  afterEach(() => {
    container.remove();
  });

  it("renders title and form", () => {
    renderSetupWizard(container, () => {});
    expect(container.querySelector("h1")?.textContent).toBe("Initial setup");
    expect(container.querySelector("#setup-admin-password-confirm")).toBeTruthy();
  });

  it("shows Passwords do not match when passwords mismatch", async () => {
    const onSuccess = vi.fn();
    renderSetupWizard(container, onSuccess);

    const form = container.querySelector("form")!;
    (
      form.querySelector("#setup-admin-username") as HTMLInputElement
    ).value = "admin";
    (form.querySelector("#setup-admin-password") as HTMLInputElement).value =
      "secret";
    (
      form.querySelector("#setup-admin-password-confirm") as HTMLInputElement
    ).value = "different";

    form.requestSubmit();
    await vi.waitFor(() => {
      const errorEl = container.querySelector("#setup-error");
      expect(errorEl?.textContent).toBe("Passwords do not match");
    });
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("shows host chips and default host select", () => {
    renderSetupWizard(container, () => {});

    const chip = container.querySelector(".setup-host-chip");
    expect(chip).toBeTruthy();
    expect(chip?.querySelector(".setup-host-chip__label")?.textContent).toContain(
      "local"
    );

    const trigger = container.querySelector("#setup-default-host");
    expect(trigger).toBeTruthy();
    const hidden = container.querySelector('input[name="default_host"]') as HTMLInputElement;
    expect(hidden?.value).toBe("local");
  });

  it("shows Host ID is required on form submit when host has empty ID", async () => {
    const onSuccess = vi.fn();
    renderSetupWizard(container, onSuccess, {
      initialHosts: [{ id: "", uri: "qemu:///system", keyfile: "" }],
    });

    const form = container.querySelector("form")!;
    (
      form.querySelector("#setup-admin-username") as HTMLInputElement
    ).value = "admin";
    (form.querySelector("#setup-admin-password") as HTMLInputElement).value =
      "secret";
    (
      form.querySelector("#setup-admin-password-confirm") as HTMLInputElement
    ).value = "secret";

    form.requestSubmit();
    await vi.waitFor(() => {
      const errorEl = container.querySelector("#setup-error");
      expect(errorEl?.textContent).toBe("Host ID is required");
    });
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("shows Host ID is required when saving host with empty ID in modal", async () => {
    const onSuccess = vi.fn();
    renderSetupWizard(container, onSuccess);

    const addBtn = Array.from(container.querySelectorAll("button")).find(
      (b) => b.textContent === "Add host"
    ) as HTMLButtonElement;
    addBtn.click();

    await vi.waitFor(() => {
      expect(container.querySelector(".modal-overlay")).toBeTruthy();
    });

    const uriInput = container.querySelector(
      "#setup-host-modal-uri"
    ) as HTMLInputElement;
    uriInput.value = "qemu:///system";
    uriInput.dispatchEvent(new Event("input", { bubbles: true }));

    const saveBtn = Array.from(
      container.querySelectorAll(".modal__footer button")
    ).find((b) => b.textContent === "Save") as HTMLButtonElement;
    saveBtn.click();

    await vi.waitFor(() => {
      const modalError = container.querySelector("#setup-host-modal-error");
      expect(modalError?.textContent).toBe("Host ID is required");
    });
    expect(container.querySelector(".modal-overlay")).toBeTruthy();
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("updates default host select when host is edited via modal", async () => {
    renderSetupWizard(container, () => {});

    const editBtn = container.querySelector(
      ".setup-host-chip__edit"
    ) as HTMLButtonElement;
    editBtn.click();

    await vi.waitFor(() => {
      const modal = container.querySelector(".modal-overlay");
      expect(modal).toBeTruthy();
    });

    const idInput = container.querySelector(
      "#setup-host-modal-id"
    ) as HTMLInputElement;
    idInput.value = "prod";
    idInput.dispatchEvent(new Event("input", { bubbles: true }));

    const saveBtn = Array.from(
      container.querySelectorAll(".modal__footer button")
    ).find((b) => b.textContent === "Save") as HTMLButtonElement;
    saveBtn.click();

    await vi.waitFor(() => {
      const hidden = container.querySelector(
        'input[name="default_host"]'
      ) as HTMLInputElement;
      expect(hidden?.value).toBe("prod");
    });
  });
});
