import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  ApiError,
  apiFetch,
  login,
  putPreferences,
  fetchHosts,
  claimVM,
  recoverVM,
  bulkClaimOrphans,
  bulkDestroyOrphans,
  createVM,
  cloneVM,
  fetchHostPools,
  fetchHostPoolVolumes,
  fetchHostNetworks,
} from "./api";

describe("api", () => {
  let mockFetch: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    mockFetch = vi.fn();
    vi.stubGlobal("fetch", mockFetch);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("ApiError has status and message", () => {
    const err = new ApiError(404, "Not found");
    expect(err.status).toBe(404);
    expect(err.message).toBe("Not found");
    expect(err.name).toBe("ApiError");
  });

  it("apiFetch success returns parsed JSON", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ foo: "bar" }),
    });
    const result = await apiFetch<{ foo: string }>("/test");
    expect(result).toEqual({ foo: "bar" });
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/test"),
      expect.objectContaining({
        credentials: "include",
        headers: expect.objectContaining({ "Content-Type": "application/json" }),
      })
    );
  });

  it("apiFetch with full URL does not prepend API_BASE", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    });
    await apiFetch("https://example.com/api/test");
    expect(mockFetch).toHaveBeenCalledWith(
      "https://example.com/api/test",
      expect.any(Object)
    );
  });

  it("apiFetch 4xx throws ApiError with body", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 401,
      text: () => Promise.resolve("Unauthorized"),
    });
    await expect(apiFetch("/test")).rejects.toThrow(ApiError);
    await expect(apiFetch("/test")).rejects.toMatchObject({
      status: 401,
      message: "Unauthorized",
    });
  });

  it("apiFetch 5xx throws ApiError", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 500,
      text: () => Promise.resolve(""),
    });
    await expect(apiFetch("/test")).rejects.toMatchObject({
      status: 500,
      message: "HTTP 500",
    });
  });

  it("login success", async () => {
    mockFetch.mockResolvedValue({ ok: true });
    await login("user", "pass");
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/auth/login"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ username: "user", password: "pass" }),
      })
    );
  });

  it("login failure throws ApiError", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 401,
      text: () => Promise.resolve("Invalid credentials"),
    });
    await expect(login("user", "pass")).rejects.toThrow(ApiError);
  });

  it("putPreferences returns preferences", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({ default_host_id: "h1", list_view_options: null }),
    });
    const result = await putPreferences({ default_host_id: "h1" });
    expect(result.default_host_id).toBe("h1");
  });

  it("fetchHosts returns hosts", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([{ id: "h1", uri: "qemu:///system" }]),
    });
    const result = await fetchHosts();
    expect(result).toHaveLength(1);
    expect(result[0]).toEqual({ id: "h1", uri: "qemu:///system" });
  });

  it("claimVM with display name", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          host_id: "h1",
          libvirt_uuid: "u1",
          display_name: "My VM",
          claimed: true,
          status: "running",
          console_preference: null,
          last_access: null,
          created_at: "",
          updated_at: "",
        }),
    });
    await claimVM("h1", "u1", "My VM");
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/hosts/h1/vms/u1/claim"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ display_name: "My VM" }),
      })
    );
  });

  it("claimVM without display name sends empty body", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    });
    await claimVM("h1", "u1");
    expect(JSON.parse(mockFetch.mock.calls[0][1].body)).toEqual({});
  });

  it("claimVM with whitespace-only display name sends empty body", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    });
    await claimVM("h1", "u1", "   ");
    expect(JSON.parse(mockFetch.mock.calls[0][1].body)).toEqual({});
  });

  it("recoverVM", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: "running" }),
    });
    await recoverVM("h1", "u1");
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/hosts/h1/vms/u1/recover"),
      expect.objectContaining({ method: "POST" })
    );
  });

  it("bulkClaimOrphans", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          claimed: [{ host_id: "h1", libvirt_uuid: "u1", display_name: "VM1" }],
          conflicts: [],
        }),
    });
    const result = await bulkClaimOrphans([
      { host_id: "h1", libvirt_uuid: "u1", display_name: "VM1" },
    ]);
    expect(result.claimed).toHaveLength(1);
    expect(result.claimed[0].display_name).toBe("VM1");
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/orphans/claim"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          items: [{ host_id: "h1", libvirt_uuid: "u1", display_name: "VM1" }],
        }),
      })
    );
  });

  it("bulkDestroyOrphans", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          destroyed: [{ host_id: "h1", libvirt_uuid: "u1" }],
          failed: [],
        }),
    });
    const result = await bulkDestroyOrphans([
      { host_id: "h1", libvirt_uuid: "u1" },
    ]);
    expect(result.destroyed).toHaveLength(1);
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/orphans/destroy"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          items: [{ host_id: "h1", libvirt_uuid: "u1" }],
        }),
      })
    );
  });

  it("createVM", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          host_id: "h1",
          libvirt_uuid: "u1",
          display_name: "My VM",
          created_at: "2024-01-01",
          status: "running",
        }),
    });
    await createVM({
      host_id: "h1",
      pool: "default",
      disk: { name: "vol1" },
    });
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/vms"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          host_id: "h1",
          pool: "default",
          disk: { name: "vol1" },
        }),
      })
    );
  });

  it("cloneVM", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          host_id: "h2",
          libvirt_uuid: "u2",
          display_name: "My VM",
          created_at: "2024-01-01",
          status: "running",
        }),
    });
    await cloneVM("h1", "u1", {
      target_host_id: "h2",
      target_pool: "default",
      target_name: "clone-vm",
    });
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/hosts/h1/vms/u1/clone"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          target_host_id: "h2",
          target_pool: "default",
          target_name: "clone-vm",
        }),
      })
    );
  });

  it("fetchHostPools", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve([{ name: "default", uuid: "u1", state: "running" }]),
    });
    const result = await fetchHostPools("h1");
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe("default");
  });

  it("fetchHostPoolVolumes", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve([
          { name: "vol1", path: "/path", capacity: 1024 },
        ]),
    });
    const result = await fetchHostPoolVolumes("h1", "default");
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe("vol1");
  });

  it("fetchHostNetworks", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve([{ name: "default", uuid: "u1", active: true }]),
    });
    const result = await fetchHostNetworks("h1");
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe("default");
  });
});
