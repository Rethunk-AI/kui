import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { subscribeToEvents } from "./events";
import { addAlert } from "./alerts";
import { showToast } from "./toast";

vi.mock("./alerts", () => ({
  addAlert: vi.fn(),
}));
vi.mock("./toast", () => ({
  showToast: vi.fn(),
}));

describe("events", () => {
  let mockEventSource: {
    addEventListener: ReturnType<typeof vi.fn>;
    onerror: (() => void) | null;
    close: ReturnType<typeof vi.fn>;
  };
  let EventSourceCtor: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    mockEventSource = {
      addEventListener: vi.fn(),
      onerror: null,
      close: vi.fn(),
    };
    EventSourceCtor = vi.fn(() => mockEventSource);
    vi.stubGlobal("EventSource", EventSourceCtor);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.mocked(addAlert).mockClear();
    vi.mocked(showToast).mockClear();
  });

  it("subscribeToEvents creates EventSource with correct URL", () => {
    const unsub = subscribeToEvents();
    expect(EventSourceCtor).toHaveBeenCalledWith(
      expect.stringContaining("/api/events"),
      { withCredentials: true }
    );
    unsub();
  });

  it("subscribeToEvents returns cleanup that closes EventSource", () => {
    const unsub = subscribeToEvents();
    unsub();
    expect(mockEventSource.close).toHaveBeenCalled();
  });

  it("host.offline event adds alert and no toast", () => {
    subscribeToEvents();
    const addListener = mockEventSource.addEventListener;
    expect(addListener).toHaveBeenCalledWith(
      "host.offline",
      expect.any(Function)
    );
    const handler = addListener.mock.calls.find(
      (c: unknown[]) => c[0] === "host.offline"
    )?.[1];
    handler?.({ data: JSON.stringify({ host_id: "h1", reason: "timeout" }) });
    expect(addAlert).toHaveBeenCalledWith(
      "host_offline",
      "Host h1 is offline: timeout",
      expect.any(String)
    );
    expect(showToast).not.toHaveBeenCalled();
  });

  it("host.online event adds alert and success toast", () => {
    subscribeToEvents();
    const addListener = mockEventSource.addEventListener;
    const handler = addListener.mock.calls.find(
      (c: unknown[]) => c[0] === "host.online"
    )?.[1];
    handler?.({ data: JSON.stringify({ host_id: "h1" }) });
    expect(addAlert).toHaveBeenCalledWith(
      "host_online",
      "Host h1 is online",
      expect.any(String)
    );
    expect(showToast).toHaveBeenCalledWith("Host h1 is online", "success");
  });

  it("vm.state_changed event adds alert and info toast", () => {
    subscribeToEvents();
    const addListener = mockEventSource.addEventListener;
    const handler = addListener.mock.calls.find(
      (c: unknown[]) => c[0] === "vm.state_changed"
    )?.[1];
    handler?.({ data: JSON.stringify({ vm_id: "v1", state: "running" }) });
    expect(addAlert).toHaveBeenCalledWith(
      "vm_state_changed",
      "VM v1 state: running",
      expect.any(String)
    );
    expect(showToast).toHaveBeenCalledWith(
      "VM v1 state: running",
      "info"
    );
  });

  it("handles invalid JSON in event data", () => {
    subscribeToEvents();
    const addListener = mockEventSource.addEventListener;
    const handler = addListener.mock.calls.find(
      (c: unknown[]) => c[0] === "host.offline"
    )?.[1];
    handler?.({ data: "not json" });
    expect(addAlert).toHaveBeenCalledWith(
      "host_offline",
      "Host unknown is offline",
      undefined
    );
  });

  it("onerror closes EventSource", () => {
    subscribeToEvents();
    expect(mockEventSource.onerror).toBeInstanceOf(Function);
    mockEventSource.onerror?.();
    expect(mockEventSource.close).toHaveBeenCalled();
  });
});
