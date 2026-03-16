/**
 * Inline host selector for create/clone/console modals.
 * Spec §3: contextual override; receives hosts + selection + onChange.
 * Reusable when create/clone/console modals are added.
 */
import type { Host } from "../lib/api";

export interface InlineHostSelectorProps {
  hosts: Host[];
  selectedHostId: string | null;
  onChange: (hostId: string) => void;
  label?: string;
  disabled?: boolean;
}

export function renderInlineHostSelector(
  container: HTMLElement,
  props: InlineHostSelectorProps
): void {
  container.innerHTML = "";
  const wrapper = document.createElement("div");
  wrapper.className = "inline-host-selector";

  const label = document.createElement("label");
  label.className = "inline-host-selector__label";
  label.textContent = props.label ?? "Host";

  const select = document.createElement("select");
  select.className = "inline-host-selector__select";
  select.disabled = props.disabled ?? false;
  select.setAttribute("aria-label", props.label ?? "Host");

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
