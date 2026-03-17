/**
 * SSE subscription to GET /api/events.
 * On host.offline, host.online, vm.state_changed — add to alerts and optionally toast.
 */
import { apiFetch } from "./api";
import { addAlert } from "./alerts";
import { showToast } from "./toast";

const API_BASE = import.meta.env.VITE_API_BASE ?? "/api";

interface SSEEventData {
  host_id?: string;
  vm_id?: string;
  state?: string;
  reason?: string;
}

const EVENT_TYPES = ["host.offline", "host.online", "vm.state_changed"] as const;

function isTransient(type: string): boolean {
  return type === "host.online" || type === "vm.state_changed";
}

function toAlertType(type: string): "host_offline" | "host_online" | "vm_state_changed" {
  switch (type) {
    case "host.offline":
      return "host_offline";
    case "host.online":
      return "host_online";
    case "vm.state_changed":
      return "vm_state_changed";
    default:
      return "host_offline";
  }
}

function formatMessage(type: string, data: SSEEventData): string {
  switch (type) {
    case "host.offline":
      return `Host ${data.host_id ?? "unknown"} is offline${data.reason ? `: ${data.reason}` : ""}`;
    case "host.online":
      return `Host ${data.host_id ?? "unknown"} is online`;
    case "vm.state_changed":
      return `VM ${data.vm_id ?? "unknown"} state: ${data.state ?? "unknown"}`;
    default:
      return type;
  }
}

function formatDetails(data: SSEEventData): string {
  const parts: string[] = [];
  if (data.host_id) parts.push(`host_id: ${data.host_id}`);
  if (data.vm_id) parts.push(`vm_id: ${data.vm_id}`);
  if (data.state) parts.push(`state: ${data.state}`);
  if (data.reason) parts.push(`reason: ${data.reason}`);
  return parts.join("\n") || "";
}

function handleEvent(type: string, data: SSEEventData): void {
  const alertType = toAlertType(type);
  const message = formatMessage(type, data);
  const details = formatDetails(data);

  addAlert(alertType, message, details || undefined);

  if (isTransient(type)) {
    showToast(message, type === "host.online" ? "success" : "info");
  }
}

export function subscribeToEvents(): () => void {
  const url = `${API_BASE}/events`;
  const es = new EventSource(url, { withCredentials: true });

  for (const type of EVENT_TYPES) {
    es.addEventListener(type, (e: MessageEvent) => {
      let data: SSEEventData = {};
      try {
        data = JSON.parse(e.data) as SSEEventData;
      } catch {
        // ignore parse errors
      }
      handleEvent(type, data);
    });
  }

  es.onerror = () => {
    es.close();
    apiFetch("/auth/me").catch(() => {});
  };

  return () => es.close();
}
