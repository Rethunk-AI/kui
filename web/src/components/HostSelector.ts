/**
 * Host selector for primary nav/header.
 * Spec §3: global default host; persists via PUT /api/preferences.
 */
import type { Host } from "../lib/api";

export interface HostSelectorProps {
  hosts: Host[];
  selectedHostId: string | null;
  onChange: (hostId: string) => void;
  disabled?: boolean;
}

export function renderHostSelector(
  container: HTMLElement,
  props: HostSelectorProps
): void {
  container.innerHTML = "";
  const wrapper = document.createElement("div");
  wrapper.className = "host-selector";

  const label = document.createElement("label");
  label.className = "host-selector__label";
  label.textContent = "Host";

  const select = document.createElement("select");
  select.className = "host-selector__select";
  select.disabled = props.disabled ?? false;
  select.setAttribute("aria-label", "Default host");

  if (props.hosts.length === 0) {
    const opt = document.createElement("option");
    opt.value = "";
    opt.textContent = "No hosts";
    select.appendChild(opt);
  } else {
    for (const h of props.hosts) {
      const opt = document.createElement("option");
      opt.value = h.id;
      opt.textContent = h.id;
      if (h.id === props.selectedHostId) {
        opt.selected = true;
      }
      select.appendChild(opt);
    }
  }

  select.addEventListener("change", () => {
    const value = select.value;
    if (!value) return;
    const exists = props.hosts.some((h) => h.id === value);
    if (exists) {
      props.onChange(value);
    }
  });

  label.appendChild(select);
  wrapper.appendChild(label);
  container.appendChild(wrapper);
}
