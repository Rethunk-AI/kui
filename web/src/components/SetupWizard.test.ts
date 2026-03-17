import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderSetupWizard } from "./SetupWizard";

vi.mock("../lib/api", () => ({
  setupComplete: vi.fn().mockResolvedValue(undefined),
  validateHost: vi.fn().mockResolvedValue({ valid: true }),
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

  it("shows Host ID is required when host ID is empty", async () => {
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
    ).value = "secret";

    const hostIdInput = container.querySelector(
      ".setup-host-row input[type='text']"
    ) as HTMLInputElement;
    hostIdInput.value = "";

    form.requestSubmit();
    await vi.waitFor(() => {
      const errorEl = container.querySelector("#setup-error");
      expect(errorEl?.textContent).toBe("Host ID is required");
    });
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("updates default host select when host ID changes", () => {
    renderSetupWizard(container, () => {});

    const defaultHostSelect = container.querySelector(
      "#setup-default-host"
    ) as HTMLSelectElement;
    const hostIdInput = container.querySelector(
      ".setup-host-row input[type='text']"
    ) as HTMLInputElement;

    expect(defaultHostSelect.value).toBe("local");
    hostIdInput.value = "prod";
    hostIdInput.dispatchEvent(new Event("input", { bubbles: true }));
    expect(defaultHostSelect.value).toBe("prod");
    expect(defaultHostSelect.options.length).toBe(1);
    expect(defaultHostSelect.options[0].textContent).toBe("prod");
  });
});
