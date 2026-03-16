/**
 * First-run checklist shown when VM list is empty and onboarding not dismissed.
 * Spec §5: create VM (pool/path), clone VM; Dismiss persists via PUT /api/preferences.
 */
import { apiFetch, putPreferences } from "../lib/api";

export function shouldShowChecklist(
  vms: unknown[],
  preferences: { list_view_options?: { onboarding_dismissed?: boolean } | null } | null
): boolean {
  if (vms.length > 0) return false;
  const dismissed = preferences?.list_view_options?.onboarding_dismissed;
  return dismissed !== true;
}

export function renderFirstRunChecklist(
  container: HTMLElement,
  onDismissed: () => void
): void {
  container.innerHTML = "";
  const section = document.createElement("section");
  section.className = "first-run-checklist";

  const title = document.createElement("h2");
  title.textContent = "Get started";
  section.appendChild(title);

  const list = document.createElement("ul");
  list.innerHTML = `
    <li>Create VM from pool or disk path</li>
    <li>Clone an existing VM</li>
  `;
  section.appendChild(list);

  const dismissBtn = document.createElement("button");
  dismissBtn.type = "button";
  dismissBtn.textContent = "Dismiss";
  dismissBtn.addEventListener("click", async () => {
    dismissBtn.disabled = true;
    try {
      await putPreferences({ list_view_options: { onboarding_dismissed: true } });
      onDismissed();
    } catch {
      dismissBtn.disabled = false;
    }
  });
  section.appendChild(dismissBtn);

  container.appendChild(section);
}
