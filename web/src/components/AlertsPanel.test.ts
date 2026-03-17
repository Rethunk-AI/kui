import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderAlertsPanel } from "./AlertsPanel";
import {
  addAlert,
  clearAllAlerts,
  dismissAlert,
  subscribe,
} from "../lib/alerts";

describe("AlertsPanel", () => {
  let container: HTMLElement;

  beforeEach(() => {
    clearAllAlerts();
    container = document.createElement("div");
    document.body.appendChild(container);
  });

  it("renders panel with header and clear button", () => {
    const unsub = renderAlertsPanel(container);
    expect(container.querySelector(".alerts-panel__title")?.textContent).toBe("Alerts");
    expect(container.querySelector(".alerts-panel__clear")?.textContent).toBe("Clear all");
    unsub();
  });

  it("clear all button clears alerts", () => {
    addAlert("host_offline", "Test alert");
    const unsub = renderAlertsPanel(container);
    expect(container.querySelectorAll(".alerts-panel__item")).toHaveLength(1);
    (container.querySelector(".alerts-panel__clear") as HTMLButtonElement)?.click();
    expect(container.querySelectorAll(".alerts-panel__item")).toHaveLength(0);
    unsub();
  });

  it("dismiss button removes single alert", () => {
    const id = addAlert("host_offline", "Test");
    const unsub = renderAlertsPanel(container);
    const dismissBtn = container.querySelector(
      `[data-alert-id="${id}"] .alerts-panel__dismiss`
    ) as HTMLButtonElement;
    dismissBtn?.click();
    expect(container.querySelectorAll(".alerts-panel__item")).toHaveLength(0);
    unsub();
  });

  it("renders alert with details toggle", () => {
    addAlert("api_error", "Error", "details here");
    const unsub = renderAlertsPanel(container);
    const toggle = container.querySelector(".alerts-panel__details-toggle");
    expect(toggle?.textContent).toBe("Show details");
    (toggle as HTMLButtonElement)?.click();
    expect(toggle?.textContent).toBe("Hide details");
    (toggle as HTMLButtonElement)?.click();
    expect(toggle?.textContent).toBe("Show details");
    unsub();
  });

  it("clear button disabled when no alerts", () => {
    const unsub = renderAlertsPanel(container);
    expect((container.querySelector(".alerts-panel__clear") as HTMLButtonElement)?.disabled).toBe(true);
    addAlert("host_offline", "Test");
    expect((container.querySelector(".alerts-panel__clear") as HTMLButtonElement)?.disabled).toBe(false);
    unsub();
  });

  it("unsub stops updates", () => {
    const unsub = renderAlertsPanel(container);
    addAlert("host_offline", "First");
    expect(container.querySelectorAll(".alerts-panel__item")).toHaveLength(1);
    unsub();
    addAlert("api_error", "Second");
    expect(container.querySelectorAll(".alerts-panel__item")).toHaveLength(1);
  });
});
