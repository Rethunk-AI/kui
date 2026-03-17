/**
 * KUI API client.
 * Uses same-origin /api by default.
 */
const API_BASE = import.meta.env.VITE_API_BASE ?? "/api";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export interface Host {
  id: string;
  uri: string;
}

export interface Preferences {
  default_host_id?: string | null;
  list_view_options?: {
    list_view?: { sort?: string; page_size?: number; group_by?: string };
    onboarding_dismissed?: boolean;
  } | null;
}

export interface VM {
  host_id: string;
  libvirt_uuid: string;
  display_name: string | null;
  claimed: boolean;
  status: string;
  console_preference: string | null;
  last_access: string | null;
  created_at: string;
  updated_at: string;
}

export interface Orphan {
  host_id: string;
  libvirt_uuid: string;
  name: string;
}

export interface VMsResponse {
  vms: VM[];
  hosts: Record<string, string>;
  orphans: Orphan[];
}

export async function apiFetch<T>(
  path: string,
  opts?: RequestInit
): Promise<T> {
  const url = path.startsWith("http") ? path : `${API_BASE}${path}`;
  const res = await fetch(url, {
    ...opts,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...opts?.headers,
    },
  });
  if (!res.ok) {
    const body = await res.text();
    throw new ApiError(res.status, body || `HTTP ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export async function login(username: string, password: string): Promise<void> {
  const res = await fetch(`${API_BASE}/auth/login`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) {
    const body = await res.text();
    throw new ApiError(res.status, body || `HTTP ${res.status}`);
  }
}

export async function putPreferences(body: {
  default_host_id?: string | null;
  list_view_options?: { onboarding_dismissed?: boolean };
}): Promise<Preferences> {
  return apiFetch<Preferences>("/preferences", {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export async function fetchHosts(): Promise<Host[]> {
  return apiFetch<Host[]>("/hosts");
}

export async function claimVM(
  hostId: string,
  libvirtUuid: string,
  displayName?: string
): Promise<VM> {
  return apiFetch<VM>(`/hosts/${encodeURIComponent(hostId)}/vms/${encodeURIComponent(libvirtUuid)}/claim`, {
    method: "POST",
    body: JSON.stringify(
      displayName != null && displayName.trim() !== ""
        ? { display_name: displayName.trim() }
        : {}
    ),
  });
}

export interface BulkClaimOrphanItem {
  host_id: string;
  libvirt_uuid: string;
  display_name?: string;
}

export interface BulkClaimResponse {
  claimed: Array<{ host_id: string; libvirt_uuid: string; display_name: string }>;
  conflicts: Array<{ host_id: string; libvirt_uuid: string; reason: string }>;
}

export async function bulkClaimOrphans(
  items: BulkClaimOrphanItem[]
): Promise<BulkClaimResponse> {
  return apiFetch<BulkClaimResponse>("/orphans/claim", {
    method: "POST",
    body: JSON.stringify({ items }),
  });
}

export interface BulkDestroyOrphanItem {
  host_id: string;
  libvirt_uuid: string;
}

export interface BulkDestroyResponse {
  destroyed: Array<{ host_id: string; libvirt_uuid: string }>;
  failed: Array<{ host_id: string; libvirt_uuid: string; reason: string }>;
}

export async function bulkDestroyOrphans(
  items: BulkDestroyOrphanItem[]
): Promise<BulkDestroyResponse> {
  return apiFetch<BulkDestroyResponse>("/orphans/destroy", {
    method: "POST",
    body: JSON.stringify({ items }),
  });
}

export async function recoverVM(
  hostId: string,
  libvirtUuid: string
): Promise<void> {
  await apiFetch<{ status: string }>(
    `/hosts/${encodeURIComponent(hostId)}/vms/${encodeURIComponent(libvirtUuid)}/recover`,
    { method: "POST" }
  );
}

export interface Pool {
  name: string;
  uuid: string;
  state: string;
}

export interface Volume {
  name: string;
  path: string;
  capacity: number;
}

export interface Network {
  name: string;
  uuid: string;
  active: boolean;
}

export interface CreateVMRequest {
  host_id: string;
  pool: string;
  disk: { name?: string; size_mb?: number };
  cpu?: number;
  ram_mb?: number;
  network?: string;
  display_name?: string;
}

export interface CreateVMResponse {
  host_id: string;
  libvirt_uuid: string;
  display_name: string;
  created_at: string;
  status: string;
}

export async function createVM(req: CreateVMRequest): Promise<CreateVMResponse> {
  return apiFetch<CreateVMResponse>("/vms", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export interface CloneVMRequest {
  target_host_id: string;
  target_pool: string;
  target_name?: string;
}

export async function cloneVM(
  sourceHostId: string,
  sourceLibvirtUuid: string,
  req: CloneVMRequest
): Promise<CreateVMResponse> {
  return apiFetch<CreateVMResponse>(
    `/hosts/${encodeURIComponent(sourceHostId)}/vms/${encodeURIComponent(sourceLibvirtUuid)}/clone`,
    {
      method: "POST",
      body: JSON.stringify(req),
    }
  );
}

export async function fetchHostPools(hostId: string): Promise<Pool[]> {
  return apiFetch<Pool[]>(`/hosts/${encodeURIComponent(hostId)}/pools`);
}

export async function fetchHostPoolVolumes(
  hostId: string,
  poolName: string
): Promise<Volume[]> {
  return apiFetch<Volume[]>(
    `/hosts/${encodeURIComponent(hostId)}/pools/${encodeURIComponent(poolName)}/volumes`
  );
}

export async function fetchHostNetworks(hostId: string): Promise<Network[]> {
  return apiFetch<Network[]>(`/hosts/${encodeURIComponent(hostId)}/networks`);
}

export interface VMDetail {
  host_id: string;
  libvirt_uuid: string;
  display_name: string | null;
  claimed: boolean;
  status: string;
  console_preference: string | null;
  last_access: string | null;
  created_at: string;
  updated_at: string;
}

export async function fetchDomainXML(
  hostId: string,
  libvirtUuid: string
): Promise<string> {
  const url = `${API_BASE}/hosts/${encodeURIComponent(hostId)}/vms/${encodeURIComponent(libvirtUuid)}/domain-xml`;
  const res = await fetch(url, {
    credentials: "include",
    headers: { Accept: "application/xml" },
  });
  if (!res.ok) {
    const body = await res.text();
    throw new ApiError(res.status, body || `HTTP ${res.status}`);
  }
  return res.text();
}

export async function putDomainXML(
  hostId: string,
  libvirtUuid: string,
  xml: string
): Promise<VMDetail> {
  const url = `${API_BASE}/hosts/${encodeURIComponent(hostId)}/vms/${encodeURIComponent(libvirtUuid)}/domain-xml`;
  const res = await fetch(url, {
    method: "PUT",
    credentials: "include",
    headers: { "Content-Type": "application/xml" },
    body: xml,
  });
  if (!res.ok) {
    const body = await res.text();
    throw new ApiError(res.status, body || `HTTP ${res.status}`);
  }
  return res.json() as Promise<VMDetail>;
}
