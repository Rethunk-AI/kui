/**
 * Alerts panel: list of alerts with optional details, dismiss per alert, clear all.
 * Spec §4: operator awareness; in-memory only; optional sanitized details.
 */
import {
  clearAllAlerts,
  dismissAlert,
  subscribe,
  type Alert,
  type AlertType,
} from "../lib/alerts";

const CRITICAL_TYPES: Set<AlertType> = new Set([
  "host_offline",
  "host_connection_error",
  "create_failure",
  "clone_failure",
  "console_failure",
  "vm_state_failure",
  "api_error",
]);

const TYPE_LABELS: Record<AlertType, string> = {
  host_offline: "Host offline",
  host_online: "Host online",
  host_connection_error: "Host connection error",
  create_failure: "Create failed",
  clone_failure: "Clone failed",
  console_failure: "Console failed",
  vm_state_changed: "VM state changed",
  vm_state_failure: "VM state failure",
  api_error: "API error",
  bulk_claim: "Bulk claim",
  bulk_destroy: "Bulk destroy",
};

function formatTime(ts: number): string {
  const d = new Date(ts);
  return d.toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function renderAlertsPanel(container: HTMLElement): () => void {
  container.innerHTML = "";
  container.className = "alerts-panel";
  container.setAttribute("aria-live", "polite");
  container.setAttribute("role", "status");

  const header = document.createElement("div");
  header.className = "alerts-panel__header";
  const title = document.createElement("h2");
  title.textContent = "Alerts";
  title.className = "alerts-panel__title";
  header.appendChild(title);

  const clearBtn = document.createElement("button");
  clearBtn.type = "button";
  clearBtn.className = "alerts-panel__clear";
  clearBtn.textContent = "Clear all";
  clearBtn.addEventListener("click", () => clearAllAlerts());
  header.appendChild(clearBtn);

  container.appendChild(header);

  const list = document.createElement("ul");
  list.className = "alerts-panel__list";
  container.appendChild(list);

  function renderList(alerts: Alert[]): void {
    const hasCritical = alerts.some((a) => CRITICAL_TYPES.has(a.type));
    container.setAttribute("aria-live", hasCritical ? "assertive" : "polite");
    list.innerHTML = "";
    for (const a of alerts) {
      const li = document.createElement("li");
      li.className = `alerts-panel__item alerts-panel__item--${a.type}`;
      li.dataset.alertId = a.id;

      const row = document.createElement("div");
      row.className = "alerts-panel__row";

      const label = document.createElement("span");
      label.className = "alerts-panel__type";
      label.textContent = TYPE_LABELS[a.type] ?? a.type;

      const msg = document.createElement("span");
      msg.className = "alerts-panel__message";
      msg.textContent = a.message;

      const time = document.createElement("span");
      time.className = "alerts-panel__time";
      time.textContent = formatTime(a.timestamp);

      const dismissBtn = document.createElement("button");
      dismissBtn.type = "button";
      dismissBtn.className = "alerts-panel__dismiss";
      dismissBtn.textContent = "Dismiss";
      dismissBtn.setAttribute("aria-label", "Dismiss alert");
      dismissBtn.addEventListener("click", () => dismissAlert(a.id));

      row.appendChild(label);
      row.appendChild(msg);
      row.appendChild(time);
      row.appendChild(dismissBtn);
      li.appendChild(row);

      if (a.details) {
        const detailsRow = document.createElement("div");
        detailsRow.className = "alerts-panel__details-row";
        const detailsToggle = document.createElement("button");
        detailsToggle.type = "button";
        detailsToggle.className = "alerts-panel__details-toggle";
        detailsToggle.textContent = "Show details";
        detailsToggle.addEventListener("click", () => {
          const block = li.querySelector(".alerts-panel__details-block");
          if (block) {
            const isHidden = block.classList.contains("alerts-panel__details-block--hidden");
            block.classList.toggle("alerts-panel__details-block--hidden", !isHidden);
            detailsToggle.textContent = isHidden ? "Hide details" : "Show details";
          }
        });
        detailsRow.appendChild(detailsToggle);

        const detailsBlock = document.createElement("pre");
        detailsBlock.className = "alerts-panel__details-block alerts-panel__details-block--hidden";
        detailsBlock.textContent = a.details;
        detailsRow.appendChild(detailsBlock);
        li.appendChild(detailsRow);
      }

      list.appendChild(li);
    }
    clearBtn.disabled = alerts.length === 0;
  }

  const unsub = subscribe(renderList);
  return unsub;
}
