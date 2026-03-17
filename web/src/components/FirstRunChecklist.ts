/**
 * First-run checklist shown when VM list is empty and onboarding not dismissed.
 * Spec §5: create VM (pool/path), clone VM; Dismiss persists via PUT /api/preferences.
 */
import { ApiError, putPreferences, type Host } from "../lib/api";
import { addAlert } from "../lib/alerts";

export function shouldShowChecklist(
  vms: unknown[] | null | undefined,
  orphans: unknown[] | null | undefined,
  preferences: { list_view_options?: { onboarding_dismissed?: boolean } | null } | null
): boolean {
  if ((vms?.length ?? 0) > 0 || (orphans?.length ?? 0) > 0) return false;
  const dismissed = preferences?.list_view_options?.onboarding_dismissed;
  return dismissed !== true;
}

export interface FirstRunChecklistProps {
  hosts: Host[];
  onDismissed: () => void;
  onOpenCreateModal?: () => void;
}

export function renderFirstRunChecklist(
  container: HTMLElement,
  props: FirstRunChecklistProps | (() => void)
): void {
  const onDismissed = typeof props === "function" ? props : props.onDismissed;
  const hosts = typeof props === "function" ? [] : props.hosts;
  const onOpenCreateModal =
    typeof props === "function" ? undefined : props.onOpenCreateModal;
  container.innerHTML = "";
  const section = document.createElement("section");
  section.className = "first-run-checklist";

  const title = document.createElement("h1");
  title.textContent = "Get started";
  section.appendChild(title);

  const list = document.createElement("ul");
  list.innerHTML = `
    <li>Create VM from pool or disk path. Ensure your host has at least one storage pool (create in virt-manager or virsh if needed).</li>
    <li>Clone an existing VM</li>
  `;
  section.appendChild(list);

  const btnGroup = document.createElement("div");
  btnGroup.className = "first-run-checklist__actions";
  if (onOpenCreateModal) {
    const createBtn = document.createElement("button");
    createBtn.type = "button";
    createBtn.textContent = "Create VM";
    createBtn.disabled = hosts.length === 0;
    if (hosts.length === 0) {
      createBtn.title = "Add hosts in setup first";
    }
    createBtn.addEventListener("click", onOpenCreateModal);
    btnGroup.appendChild(createBtn);
  }
  const dismissBtn = document.createElement("button");
  dismissBtn.type = "button";
  dismissBtn.textContent = "Dismiss";
  dismissBtn.addEventListener("click", async () => {
    dismissBtn.disabled = true;
    try {
      await putPreferences({ list_view_options: { onboarding_dismissed: true } });
      onDismissed();
    } catch (err) {
      dismissBtn.disabled = false;
      if (err instanceof ApiError && err.status === 401) return;
      const msg = err instanceof ApiError ? err.message : "Failed to save preferences";
      addAlert("api_error", msg, err instanceof ApiError ? String(err.status) : undefined);
    }
  });
  btnGroup.appendChild(dismissBtn);
  section.appendChild(btnGroup);

  container.appendChild(section);
}
