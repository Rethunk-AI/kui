/**
 * KUI SPA bootstrap.
 * Vanilla JS + Vite. Winbox.js compatible.
 * Bootstrap: fetch vms + preferences + hosts; 401 → login; else VM list or first-run checklist.
 */
import "./styles.css";
import {
  ApiError,
  apiFetch,
  fetchHosts,
  login,
  putPreferences,
  type Host,
  type Preferences,
  type VMsResponse,
} from "./lib/api";
import {
  renderFirstRunChecklist,
  shouldShowChecklist,
} from "./components/FirstRunChecklist";
import { renderHostSelector } from "./components/HostSelector";

const appEl = document.getElementById("app");
if (!appEl) throw new Error("app element missing");
const app: HTMLElement = appEl;

async function bootstrap(): Promise<void> {
  let vmsResp: VMsResponse;
  let preferences: Preferences | null = null;
  let hosts: Host[] = [];

  try {
    const [vms, prefs, hostsResp] = await Promise.all([
      apiFetch<VMsResponse>("/vms"),
      apiFetch<Preferences>("/preferences"),
      fetchHosts(),
    ]);
    vmsResp = vms;
    preferences = prefs;
    hosts = hostsResp;
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      renderLoginPage(app, bootstrap);
      return;
    }
    throw e;
  }

  renderMain(app, vmsResp, preferences, hosts, bootstrap);
}

function renderMain(
  container: HTMLElement,
  vmsResp: VMsResponse,
  preferences: Preferences | null,
  hosts: Host[],
  onDataChange: () => void
): void {
  container.innerHTML = "";

  const layout = document.createElement("div");
  layout.className = "app-layout";

  const header = document.createElement("header");
  header.className = "app-header";
  const nav = document.createElement("nav");
  nav.className = "app-nav";

  const hostSelectorEl = document.createElement("div");
  const selectedHostId =
    preferences?.default_host_id ?? hosts[0]?.id ?? null;
  renderHostSelector(hostSelectorEl, {
    hosts,
    selectedHostId,
    onChange: async (hostId: string) => {
      try {
        await putPreferences({ default_host_id: hostId });
        const prefs = await apiFetch<Preferences>("/preferences");
        renderMain(container, vmsResp, prefs, hosts, onDataChange);
      } catch {
        // Re-render to restore previous state
        renderMain(container, vmsResp, preferences, hosts, onDataChange);
      }
    },
  });
  nav.appendChild(hostSelectorEl);
  header.appendChild(nav);
  layout.appendChild(header);

  const content = document.createElement("main");
  content.className = "app-content";

  if (shouldShowChecklist(vmsResp.vms, preferences)) {
    const checklistContainer = document.createElement("div");
    content.appendChild(checklistContainer);
    renderFirstRunChecklist(checklistContainer, async () => {
      const [prefs, hostsResp] = await Promise.all([
        apiFetch<Preferences>("/preferences"),
        fetchHosts(),
      ]);
      renderMain(container, vmsResp, prefs, hostsResp, onDataChange);
    });
  } else {
    const vmList = document.createElement("div");
    vmList.className = "vm-list";
    if (vmsResp.vms.length === 0) {
      vmList.innerHTML = "<p>No VMs</p>";
    } else {
      const ul = document.createElement("ul");
      for (const vm of vmsResp.vms as {
        display_name?: string;
        host_id?: string;
      }[]) {
        const li = document.createElement("li");
        li.textContent = vm.display_name ?? vm.host_id ?? "VM";
        ul.appendChild(li);
      }
      vmList.appendChild(ul);
    }
    content.appendChild(vmList);
  }

  layout.appendChild(content);
  container.appendChild(layout);
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
