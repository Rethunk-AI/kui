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
