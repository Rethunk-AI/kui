import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  shouldShowChecklist,
  renderFirstRunChecklist,
} from "./FirstRunChecklist";

vi.mock("../lib/api", () => ({
  putPreferences: vi.fn().mockResolvedValue({}),
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

describe("FirstRunChecklist", () => {
  describe("shouldShowChecklist", () => {
    it("returns false when vms exist", () => {
      expect(
        shouldShowChecklist([{ id: "v1" }], [], { list_view_options: null })
      ).toBe(false);
    });

    it("returns false when orphans exist", () => {
      expect(
        shouldShowChecklist([], [{ name: "o1" }], { list_view_options: null })
      ).toBe(false);
    });

    it("returns false when onboarding dismissed", () => {
      expect(
        shouldShowChecklist([], [], {
          list_view_options: { onboarding_dismissed: true },
        })
      ).toBe(false);
    });

    it("returns true when empty and not dismissed", () => {
      expect(
        shouldShowChecklist([], [], { list_view_options: null })
      ).toBe(true);
    });

    it("returns true when preferences null", () => {
      expect(shouldShowChecklist([], [], null)).toBe(true);
    });
  });

  describe("renderFirstRunChecklist", () => {
    let container: HTMLElement;

    beforeEach(() => {
      container = document.createElement("div");
      document.body.appendChild(container);
    });

    it("renders title and list", () => {
      renderFirstRunChecklist(container, { onDismissed: () => {} });
      expect(container.querySelector("h1")?.textContent).toBe("Get started");
      expect(container.querySelector("ul")).toBeTruthy();
    });

    it("Create VM button when onOpenCreateModal provided", () => {
      const onOpenCreateModal = vi.fn();
      renderFirstRunChecklist(container, {
        onDismissed: () => {},
        onOpenCreateModal,
      });
      const btn = container.querySelector("button");
      expect(btn?.textContent).toBe("Create VM");
      btn?.click();
      expect(onOpenCreateModal).toHaveBeenCalled();
    });

    it("Dismiss button error re-enables button", async () => {
      const { putPreferences } = await import("../lib/api");
      vi.mocked(putPreferences).mockRejectedValueOnce(new Error("Save failed"));
      const onDismissed = vi.fn();
      renderFirstRunChecklist(container, { onDismissed });
      const buttons = container.querySelectorAll("button");
      const dismissBtn = Array.from(buttons).find((b) => b.textContent === "Dismiss");
      dismissBtn?.click();
      await vi.waitFor(() => {
        expect((dismissBtn as HTMLButtonElement).disabled).toBe(false);
      });
      expect(onDismissed).not.toHaveBeenCalled();
    });

    it("Dismiss button calls putPreferences and onDismissed", async () => {
      const { putPreferences } = await import("../lib/api");
      const onDismissed = vi.fn();
      renderFirstRunChecklist(container, { onDismissed });
      const buttons = container.querySelectorAll("button");
      const dismissBtn = Array.from(buttons).find((b) => b.textContent === "Dismiss");
      expect(dismissBtn).toBeTruthy();
      dismissBtn?.click();
      await vi.waitFor(() => {
        expect(putPreferences).toHaveBeenCalledWith({
          list_view_options: { onboarding_dismissed: true },
        });
        expect(onDismissed).toHaveBeenCalled();
      });
    });

    it("accepts function as props (legacy)", () => {
      const onDismissed = vi.fn();
      renderFirstRunChecklist(container, onDismissed);
      expect(container.querySelector(".first-run-checklist")).toBeTruthy();
    });
  });
});
