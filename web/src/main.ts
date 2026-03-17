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
  type VM,
  type VMsResponse,
} from "./lib/api";
import { addAlert } from "./lib/alerts";
import { subscribeToEvents } from "./lib/events";
import {
  renderFirstRunChecklist,
  shouldShowChecklist,
} from "./components/FirstRunChecklist";
import { renderCreateVMModal } from "./components/CreateVMModal";
import { renderCloneVMModal } from "./components/CloneVMModal";
import { renderDomainXMLEditor } from "./components/DomainXMLEditor";
import { renderAlertsPanel } from "./components/AlertsPanel";
import { renderHostSelector } from "./components/HostSelector";
import { renderVMList } from "./components/VMList";
import { renderShortcutHelp } from "./components/ShortcutHelp";
import { openConsoleForVM } from "./lib/console";
import { registerShortcuts } from "./lib/shortcuts";
import { closeTopmostWinBox } from "./lib/winbox-adapter";

const appEl = document.getElementById("app");
if (!appEl) throw new Error("app element missing");
const app: HTMLElement = appEl;

function injectSkipLink(): void {
  const skip = document.createElement("a");
  skip.href = "#main-content";
  skip.textContent = "Skip to main content";
  skip.className = "skip-link";
  document.body.insertBefore(skip, document.body.firstChild);
}
injectSkipLink();

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

let eventsUnsub: (() => void) | null = null;
let alertsPanelUnsub: (() => void) | null = null;
let shortcutsUnsub: (() => void) | null = null;

function renderMain(
  container: HTMLElement,
  vmsResp: VMsResponse,
  preferences: Preferences | null,
  hosts: Host[],
  onDataChange: () => void
): void {
  container.innerHTML = "";

  if (eventsUnsub) {
    eventsUnsub();
    eventsUnsub = null;
  }
  if (alertsPanelUnsub) {
    alertsPanelUnsub();
    alertsPanelUnsub = null;
  }
  if (shortcutsUnsub) {
    shortcutsUnsub();
    shortcutsUnsub = null;
  }

  const shortcutHelpRoot = document.getElementById("shortcut-help-root");
  const shortcutHelpVisibleRef = { current: false };

  const updateShortcutHelp = (): void => {
    if (shortcutHelpRoot) {
      renderShortcutHelp(shortcutHelpRoot, {
        visible: shortcutHelpVisibleRef.current,
        onClose: () => {
          shortcutHelpVisibleRef.current = false;
          updateShortcutHelp();
        },
      });
    }
  };

  eventsUnsub = subscribeToEvents();

  const layout = document.createElement("div");
  layout.className = "app-layout";

  const header = document.createElement("header");
  header.className = "app-header";
  const nav = document.createElement("nav");
  nav.className = "app-nav";
  nav.setAttribute("aria-label", "Main navigation");

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
      } catch (err) {
        const msg = err instanceof ApiError ? err.message : "Failed to save host preference";
        addAlert("api_error", msg, err instanceof ApiError ? String(err.status) : undefined);
        renderMain(container, vmsResp, preferences, hosts, onDataChange);
      }
    },
  });
  nav.appendChild(hostSelectorEl);

  const alertsToggle = document.createElement("button");
  alertsToggle.type = "button";
  alertsToggle.className = "alerts-toggle";
  alertsToggle.textContent = "Alerts";
  alertsToggle.setAttribute("aria-label", "Toggle alerts panel");
  let alertsPanelVisible = true;
  const alertsPanelWrapper = document.createElement("div");
  alertsPanelWrapper.className = "alerts-panel-wrapper";
  const alertsPanelEl = document.createElement("div");
  alertsPanelWrapper.appendChild(alertsPanelEl);
  alertsPanelUnsub = renderAlertsPanel(alertsPanelEl);

  alertsToggle.addEventListener("click", () => {
    alertsPanelVisible = !alertsPanelVisible;
    alertsPanelWrapper.classList.toggle("alerts-panel-wrapper--hidden", !alertsPanelVisible);
  });
  nav.appendChild(alertsToggle);
  header.appendChild(nav);
  layout.appendChild(header);

  layout.appendChild(alertsPanelWrapper);

  const content = document.createElement("main");
  content.id = "main-content";
  content.tabIndex = -1;
  content.className = "app-content";

  const modalContainer = document.createElement("div");
  modalContainer.id = "modal-root";

  const openCreateModal = (): void => {
    modalContainer.innerHTML = "";
    renderCreateVMModal(modalContainer, {
      hosts,
      defaultHostId: selectedHostId,
      onClose: () => {
        modalContainer.innerHTML = "";
      },
      onSuccess: onDataChange,
    });
  };

  const openCloneModal = (vm: VM): void => {
    modalContainer.innerHTML = "";
    renderCloneVMModal(modalContainer, {
      sourceVM: vm,
      hosts,
      defaultHostId: selectedHostId,
      onClose: () => {
        modalContainer.innerHTML = "";
      },
      onSuccess: onDataChange,
    });
  };

  const openDomainXMLEditor = (vm: VM): void => {
    modalContainer.innerHTML = "";
    renderDomainXMLEditor(modalContainer, {
      vm,
      onClose: () => {
        modalContainer.innerHTML = "";
      },
      onSuccess: onDataChange,
    });
  };

  const groupBy =
    preferences?.list_view_options?.list_view?.group_by === "created_at"
      ? "created_at"
      : "last_access";

  const selectionRef: { vm: VM | null; index: number } = { vm: null, index: 0 };

  if (shouldShowChecklist(vmsResp.vms, vmsResp.orphans, preferences)) {
    const checklistContainer = document.createElement("div");
    content.appendChild(checklistContainer);
    renderFirstRunChecklist(checklistContainer, {
      onDismissed: async () => {
        const [vms, prefs, hostsResp] = await Promise.all([
          apiFetch<VMsResponse>("/vms"),
          apiFetch<Preferences>("/preferences"),
          fetchHosts(),
        ]);
        renderMain(container, vms, prefs, hostsResp, onDataChange);
      },
      onOpenCreateModal: openCreateModal,
    });
  } else {
    const vmListContainer = document.createElement("div");
    vmListContainer.className = "vm-list-container";
    renderVMList(vmListContainer, {
      data: vmsResp,
      groupBy,
      onRefresh: onDataChange,
      onOpenConsole: openConsoleForVM,
      onOpenCreateModal: openCreateModal,
      onOpenCloneModal: openCloneModal,
      onOpenDomainXMLEditor: openDomainXMLEditor,
      onRowSelect: (vm, index) => {
        selectionRef.vm = vm;
        selectionRef.index = index;
      },
    });
    content.appendChild(vmListContainer);
  }

  layout.appendChild(content);
  layout.appendChild(modalContainer);
  container.appendChild(layout);

  shortcutsUnsub = registerShortcuts({
    getHasModalOpen: () => modalContainer.children.length > 0,
    getHasSelection: () => selectionRef.vm != null,
    getSelectedVM: () => selectionRef.vm,
    onShowHelp: () => {
      shortcutHelpVisibleRef.current = true;
      updateShortcutHelp();
    },
    onEscape: () => {
      if (shortcutHelpVisibleRef.current) {
        shortcutHelpVisibleRef.current = false;
        updateShortcutHelp();
        return;
      }
      if (modalContainer.children.length > 0) {
        modalContainer.innerHTML = "";
      } else {
        closeTopmostWinBox();
      }
    },
    onEnter: () => {
      if (selectionRef.vm) openConsoleForVM(selectionRef.vm);
    },
    onCreateVM: openCreateModal,
    onRefresh: onDataChange,
    onClone: () => {
      if (selectionRef.vm) openCloneModal(selectionRef.vm);
    },
  });

  updateShortcutHelp();
}

function renderLoginPage(container: HTMLElement, onSuccess: () => void): void {
  container.innerHTML = "";
  const main = document.createElement("main");
  main.id = "main-content";
  main.tabIndex = -1;
  main.className = "app-content";
  const form = document.createElement("form");
  form.className = "login-form";
  form.innerHTML = `
    <h1>Log in</h1>
    <label for="login-username">Username</label>
    <input id="login-username" type="text" name="username" required aria-required="true" autocomplete="username" aria-describedby="login-error" />
    <label for="login-password">Password</label>
    <input id="login-password" type="password" name="password" required aria-required="true" autocomplete="current-password" aria-describedby="login-error" />
    <button type="submit">Log in</button>
    <p id="login-error" class="login-error" role="alert" aria-live="polite"></p>
  `;

  const errorEl = form.querySelector("#login-error") as HTMLParagraphElement;
  const usernameInput = form.querySelector("#login-username") as HTMLInputElement;
  const passwordInput = form.querySelector("#login-password") as HTMLInputElement;

  const clearInvalid = (): void => {
    usernameInput.removeAttribute("aria-invalid");
    passwordInput.removeAttribute("aria-invalid");
  };

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(form);
    const username = (fd.get("username") as string)?.trim() ?? "";
    const password = (fd.get("password") as string) ?? "";
    if (!username || !password) return;

    errorEl.textContent = "";
    clearInvalid();
    try {
      await login(username, password);
      onSuccess();
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : "Login failed";
      errorEl.textContent = msg;
      usernameInput.setAttribute("aria-invalid", "true");
      passwordInput.setAttribute("aria-invalid", "true");
    }
  });

  main.appendChild(form);
  container.appendChild(main);
}

bootstrap().catch((err) => {
  app.innerHTML = `<p class="error">Failed to load: ${err instanceof Error ? err.message : String(err)}</p>`;
});
