/**
 * KUI SPA bootstrap.
 * Vanilla JS + Vite. Winbox.js compatible.
 * Bootstrap: fetch vms + preferences; 401 → login; else VM list or first-run checklist.
 */
import "./styles.css";
import { ApiError, apiFetch, login, Preferences, VMsResponse } from "./lib/api";
import {
  renderFirstRunChecklist,
  shouldShowChecklist,
} from "./components/FirstRunChecklist";

const appEl = document.getElementById("app");
if (!appEl) throw new Error("app element missing");
const app: HTMLElement = appEl;

async function bootstrap(): Promise<void> {
  let vmsResp: VMsResponse;
  let preferences: Preferences | null = null;

  try {
    const [vms, prefs] = await Promise.all([
      apiFetch<VMsResponse>("/vms"),
      apiFetch<Preferences>("/preferences"),
    ]);
    vmsResp = vms;
    preferences = prefs;
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      renderLoginPage(app, bootstrap);
      return;
    }
    throw e;
  }

  renderMain(app, vmsResp, preferences, bootstrap);
}

function renderMain(
  container: HTMLElement,
  vmsResp: VMsResponse,
  preferences: Preferences | null,
  onDataChange: () => void
): void {
  container.innerHTML = "";

  if (shouldShowChecklist(vmsResp.vms, preferences)) {
    const checklistContainer = document.createElement("div");
    container.appendChild(checklistContainer);
    renderFirstRunChecklist(checklistContainer, async () => {
      const prefs = await apiFetch<Preferences>("/preferences");
      renderMain(container, vmsResp, prefs, onDataChange);
    });
    return;
  }

  const vmList = document.createElement("div");
  vmList.className = "vm-list";
  if (vmsResp.vms.length === 0) {
    vmList.innerHTML = "<p>No VMs</p>";
  } else {
    const ul = document.createElement("ul");
    for (const vm of vmsResp.vms as { display_name?: string; host_id?: string }[]) {
      const li = document.createElement("li");
      li.textContent = vm.display_name ?? vm.host_id ?? "VM";
      ul.appendChild(li);
    }
    vmList.appendChild(ul);
  }
  container.appendChild(vmList);
}

function renderLoginPage(container: HTMLElement, onSuccess: () => void): void {
  container.innerHTML = "";
  const form = document.createElement("form");
  form.className = "login-form";
  form.innerHTML = `
    <h2>Log in</h2>
    <label>Username <input type="text" name="username" required autocomplete="username" /></label>
    <label>Password <input type="password" name="password" required autocomplete="current-password" /></label>
    <button type="submit">Log in</button>
    <p class="login-error" role="alert"></p>
  `;

  const errorEl = form.querySelector(".login-error") as HTMLParagraphElement;

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(form);
    const username = (fd.get("username") as string)?.trim() ?? "";
    const password = (fd.get("password") as string) ?? "";
    if (!username || !password) return;

    errorEl.textContent = "";
    try {
      await login(username, password);
      onSuccess();
    } catch (err) {
      errorEl.textContent =
        err instanceof ApiError ? err.message : "Login failed";
    }
  });

  container.appendChild(form);
}

bootstrap().catch((err) => {
  app.innerHTML = `<p class="error">Failed to load: ${err instanceof Error ? err.message : String(err)}</p>`;
});
