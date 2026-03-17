import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { showToast } from "./toast";

describe("toast", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("showToast creates toast element in container", () => {
    showToast("Test message");
    const container = document.getElementById("kui-toast-container");
    expect(container).toBeTruthy();
    expect(container?.querySelector(".toast")).toBeTruthy();
    expect(container?.querySelector(".toast")?.textContent).toBe("Test message");
  });

  it("showToast creates announcer with message", () => {
    showToast("Test message");
    const announcer = document.getElementById("kui-toast-announcer");
    expect(announcer?.textContent).toBe("Test message");
  });

  it("showToast defaults to info type", () => {
    showToast("Test");
    const toast = document.querySelector(".toast");
    expect(toast?.classList.contains("toast--info")).toBe(true);
  });

  it("showToast with success type", () => {
    showToast("Success", "success");
    const toast = document.querySelector(".toast");
    expect(toast?.classList.contains("toast--success")).toBe(true);
  });

  it("showToast with warn type", () => {
    showToast("Warning", "warn");
    const toast = document.querySelector(".toast");
    expect(toast?.classList.contains("toast--warn")).toBe(true);
  });

  it("toast auto-dismisses after duration", () => {
    showToast("Test");
    const container = document.getElementById("kui-toast-container");
    expect(container?.querySelector(".toast")).toBeTruthy();
    vi.advanceTimersByTime(4000);
    vi.advanceTimersByTime(200);
    expect(container?.querySelector(".toast")).toBeFalsy();
  });

  it("toast click dismisses immediately", () => {
    showToast("Test");
    const toast = document.querySelector(".toast");
    expect(toast).toBeTruthy();
    expect(toast).toBeInstanceOf(HTMLElement);
    (toast as HTMLElement).click();
    vi.advanceTimersByTime(200);
    expect(document.querySelector(".toast")).toBeFalsy();
  });
});
