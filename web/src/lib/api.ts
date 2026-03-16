/**
 * KUI API client.
 * Uses same-origin /api by default.
 */
const API_BASE = import.meta.env.VITE_API_BASE ?? "/api";

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
    throw new Error(body || `HTTP ${res.status}`);
  }
  return res.json() as Promise<T>;
}
