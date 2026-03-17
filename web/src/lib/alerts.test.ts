import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  addAlert,
  dismissAlert,
  clearAllAlerts,
  getAlerts,
  subscribe,
  type AlertType,
} from "./alerts";

describe("alerts", () => {
  beforeEach(() => {
    clearAllAlerts();
  });

  it("getAlerts returns empty array initially", () => {
    expect(getAlerts()).toEqual([]);
  });

  it("addAlert adds alert and returns id", () => {
    const id = addAlert("host_offline", "Host foo is offline");
    expect(id).toMatch(/^alert-\d+-\d+$/);
    const alerts = getAlerts();
    expect(alerts).toHaveLength(1);
    expect(alerts[0]).toMatchObject({
      type: "host_offline",
      message: "Host foo is offline",
      timestamp: expect.any(Number),
    });
  });

  it("addAlert with details stores details", () => {
    const id = addAlert("api_error", "Request failed", "status: 500");
    const alerts = getAlerts();
    expect(alerts[0].details).toBe("status: 500");
  });

  const alertTypes: AlertType[] = [
    "host_offline",
    "host_online",
    "host_connection_error",
    "create_failure",
    "clone_failure",
    "console_failure",
    "vm_state_changed",
    "vm_state_failure",
    "api_error",
  ];

  it.each(alertTypes)("addAlert accepts type %s", (type) => {
    addAlert(type, "test message");
    expect(getAlerts()[0].type).toBe(type);
  });

  it("dismissAlert removes alert by id", () => {
    const id = addAlert("host_offline", "msg");
    expect(getAlerts()).toHaveLength(1);
    dismissAlert(id);
    expect(getAlerts()).toHaveLength(0);
  });

  it("dismissAlert with unknown id is no-op", () => {
    addAlert("host_offline", "msg");
    dismissAlert("alert-999-0");
    expect(getAlerts()).toHaveLength(1);
  });

  it("clearAllAlerts removes all alerts", () => {
    addAlert("host_offline", "msg1");
    addAlert("api_error", "msg2");
    clearAllAlerts();
    expect(getAlerts()).toEqual([]);
  });

  it("subscribe calls listener with current alerts and returns unsub", () => {
    const listener = vi.fn();
    const unsub = subscribe(listener);
    expect(listener).toHaveBeenCalledWith([]);

    addAlert("host_offline", "msg");
    expect(listener).toHaveBeenCalledTimes(2);
    expect(listener).toHaveBeenLastCalledWith(
      expect.arrayContaining([
        expect.objectContaining({ message: "msg" }),
      ])
    );

    unsub();
    addAlert("api_error", "msg2");
    expect(listener).toHaveBeenCalledTimes(2);
  });
});
