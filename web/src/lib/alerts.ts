/**
 * In-memory alerts store. Cleared on refresh.
 * Spec §4: host offline, create/clone/console failure, VM state, host connection errors.
 */

export type AlertType =
  | "host_offline"
  | "host_online"
  | "host_connection_error"
  | "create_failure"
  | "clone_failure"
  | "console_failure"
  | "vm_state_changed"
  | "vm_state_failure"
  | "api_error";

export interface Alert {
  id: string;
  type: AlertType;
  message: string;
  details?: string;
  timestamp: number;
}

type Listener = (alerts: Alert[]) => void;

let alerts: Alert[] = [];
let listeners: Listener[] = [];
let idCounter = 0;

function notify(): void {
  const copy = [...alerts];
  listeners.forEach((fn) => fn(copy));
}

export function getAlerts(): Alert[] {
  return [...alerts];
}

export function addAlert(
  type: AlertType,
  message: string,
  details?: string
): string {
  const id = `alert-${++idCounter}-${Date.now()}`;
  const alert: Alert = {
    id,
    type,
    message,
    details,
    timestamp: Date.now(),
  };
  alerts = [...alerts, alert];
  notify();
  return id;
}

export function dismissAlert(id: string): void {
  alerts = alerts.filter((a) => a.id !== id);
  notify();
}

export function clearAllAlerts(): void {
  alerts = [];
  notify();
}

export function subscribe(listener: Listener): () => void {
  listeners.push(listener);
  listener(getAlerts());
  return () => {
    listeners = listeners.filter((l) => l !== listener);
  };
}
