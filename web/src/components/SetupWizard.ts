/**
 * Setup wizard for first-run configuration.
 * Collects admin credentials and host config; validates before saving.
 */
import {
  ApiError,
  provisionHost,
  setupComplete,
  validateHost,
  type ProvisionHostAuditResponse,
  type ProvisionHostResultResponse,
  type SetupCompleteRequest,
} from "../lib/api";
import { setupFocusTrap } from "../lib/focus-trap";

function isLocalUri(uri: string): boolean {
  const trimmed = uri.trim();
  return (
    trimmed.startsWith("qemu:///") || trimmed.startsWith("qemu+unix:")
  );
}

export type HostEntry = { id: string; uri: string; keyfile: string };

export interface SetupWizardOptions {
  /** For testing: initial hosts to use instead of default. */
  initialHosts?: HostEntry[];
}

export function renderSetupWizard(
  container: HTMLElement,
  onSuccess: () => void,
  options?: SetupWizardOptions
): void {
  container.innerHTML = "";

  const form = document.createElement("form");
  form.className = "login-form";
  form.setAttribute("aria-labelledby", "setup-wizard-title");

  const title = document.createElement("h1");
  title.id = "setup-wizard-title";
  title.textContent = "Initial setup";
  form.appendChild(title);

  // Admin section
  const adminSection = document.createElement("fieldset");
  adminSection.innerHTML = `
    <legend>Admin account</legend>
    <label for="setup-admin-username">Username</label>
    <input id="setup-admin-username" type="text" name="admin_username" required aria-required="true" autocomplete="username" aria-describedby="setup-error" />
    <label for="setup-admin-password">Password</label>
    <input id="setup-admin-password" type="password" name="admin_password" required aria-required="true" autocomplete="new-password" aria-describedby="setup-error" />
    <label for="setup-admin-password-confirm">Confirm password</label>
    <input id="setup-admin-password-confirm" type="password" name="admin_password_confirm" required aria-required="true" autocomplete="new-password" aria-describedby="setup-error" />
  `;
  form.appendChild(adminSection);

  // Hosts section
  const hostsSection = document.createElement("fieldset");
  const hostsLegend = document.createElement("legend");
  hostsLegend.textContent = "Hosts";
  hostsSection.appendChild(hostsLegend);

  const hostsContainer = document.createElement("div");
  hostsContainer.className = "setup-hosts";

  const defaultHosts: HostEntry[] = [
    { id: "local", uri: "qemu:///system", keyfile: "" },
  ];
  const hosts: HostEntry[] = options?.initialHosts
    ? [...options.initialHosts]
    : defaultHosts;

  function uriSummary(uri: string): string {
    const u = uri.trim();
    if (u.length <= 40) return u;
    return u.slice(0, 37) + "…";
  }

  function renderHostChips(): void {
    hostsContainer.innerHTML = "";
    const chipsList = document.createElement("div");
    chipsList.className = "setup-host-chips";
    hosts.forEach((h, idx) => {
      const chip = document.createElement("div");
      chip.className = "setup-host-chip";
      chip.innerHTML = `
        <span class="setup-host-chip__label">${h.id} — ${uriSummary(h.uri)}</span>
        <button type="button" class="setup-host-chip__edit" aria-label="Edit ${h.id}">Edit</button>
        <button type="button" class="setup-host-chip__remove" aria-label="Remove ${h.id}">Remove</button>
      `;
      chip.querySelector(".setup-host-chip__edit")?.addEventListener("click", () => {
        openHostModal(idx);
      });
      chip.querySelector(".setup-host-chip__remove")?.addEventListener("click", () => {
        if (hosts.length <= 1) return;
        hosts.splice(idx, 1);
        renderHostChips();
        updateDefaultHostSelect();
      });
      chipsList.appendChild(chip);
    });
    hostsContainer.appendChild(chipsList);
  }

  function openHostModal(editingIndex?: number): void {
    const isEdit = editingIndex !== undefined;
    const initial = isEdit ? hosts[editingIndex] : { id: "", uri: "", keyfile: "" };

    const overlay = document.createElement("div");
    overlay.className = "modal-overlay";
    overlay.setAttribute("role", "dialog");
    overlay.setAttribute("aria-modal", "true");
    overlay.setAttribute("aria-labelledby", "setup-host-modal-title");

    const modal = document.createElement("div");
    modal.className = "modal";

    const header = document.createElement("div");
    header.className = "modal__header";
    const title = document.createElement("h2");
    title.id = "setup-host-modal-title";
    title.textContent = isEdit ? "Edit host" : "Add host";
    header.appendChild(title);
    const closeBtn = document.createElement("button");
    closeBtn.type = "button";
    closeBtn.className = "modal__close";
    closeBtn.textContent = "×";
    closeBtn.setAttribute("aria-label", "Close");
    header.appendChild(closeBtn);
    modal.appendChild(header);

    const form = document.createElement("form");
    form.className = "modal__form";

    const idField = document.createElement("div");
    idField.className = "modal__field";
    idField.innerHTML = `
      <label class="modal__label">Host ID
        <input id="setup-host-modal-id" type="text" placeholder="e.g. local" aria-describedby="setup-host-modal-error" />
      </label>
    `;
    const uriField = document.createElement("div");
    uriField.className = "modal__field";
    uriField.innerHTML = `
      <label class="modal__label">URI
        <input id="setup-host-modal-uri" type="text" placeholder="e.g. qemu:///system or qemu+ssh://user@host/system" aria-describedby="setup-host-modal-error" />
      </label>
    `;
    const keyfileWrap = document.createElement("div");
    keyfileWrap.className = "modal__field setup-host-keyfile-wrap";
    keyfileWrap.style.display = isLocalUri(initial.uri) ? "none" : "";
    keyfileWrap.innerHTML = `
      <label class="modal__label">SSH key path
        <input id="setup-host-modal-keyfile" type="text" placeholder="/path/to/key" aria-describedby="setup-host-modal-error" />
      </label>
    `;
    const validateResult = document.createElement("p");
    validateResult.className = "setup-validate-result";
    validateResult.setAttribute("role", "status");
    validateResult.setAttribute("aria-live", "polite");
    const provisionPanel = document.createElement("div");
    provisionPanel.className = "setup-provision-panel";
    provisionPanel.style.display = "none";
    const modalError = document.createElement("p");
    modalError.id = "setup-host-modal-error";
    modalError.className = "modal__error";
    modalError.setAttribute("role", "alert");

    const idInput = idField.querySelector("input") as HTMLInputElement;
    const uriInput = uriField.querySelector("input") as HTMLInputElement;
    const keyfileInput = keyfileWrap.querySelector("input") as HTMLInputElement;
    idInput.value = initial.id;
    uriInput.value = initial.uri;
    keyfileInput.value = initial.keyfile;

    const toggleKeyfile = (): void => {
      keyfileWrap.style.display = isLocalUri(uriInput.value) ? "none" : "";
    };
    uriInput.addEventListener("input", toggleKeyfile);
    uriInput.addEventListener("change", toggleKeyfile);

    form.appendChild(idField);
    form.appendChild(uriField);
    form.appendChild(keyfileWrap);
    form.appendChild(modalError);

    const validateBtn = document.createElement("button");
    validateBtn.type = "button";
    validateBtn.className = "setup-validate-btn";
    validateBtn.textContent = "Validate host";
    form.appendChild(validateBtn);
    form.appendChild(validateResult);
    form.appendChild(provisionPanel);

    function needsProvision(error: string | undefined): boolean {
      if (!error) return false;
      return error.includes("no storage pools") || error.includes("no networks");
    }

    function renderProvisionPanel(
      audit: ProvisionHostAuditResponse["audit"],
      onProvision: (btn: HTMLButtonElement) => void
    ): void {
      provisionPanel.innerHTML = "";
      if (!audit || (!audit.pool && !audit.network)) return;
      provisionPanel.style.display = "";
      const details = document.createElement("div");
      details.className = "setup-provision-details";
      if (audit.pool) {
        const p = document.createElement("p");
        p.textContent = `Pool: ${audit.pool.path} (${audit.pool.type}, ${audit.pool.name})`;
        details.appendChild(p);
      }
      if (audit.network) {
        const p = document.createElement("p");
        p.textContent = `Network: ${audit.network.name} (${audit.network.subnet}, ${audit.network.type})`;
        details.appendChild(p);
      }
      provisionPanel.appendChild(details);
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "setup-provision-btn";
      btn.textContent = "Provision";
      btn.addEventListener("click", () => onProvision(btn));
      provisionPanel.appendChild(btn);
    }

    validateBtn.addEventListener("click", async () => {
      const hostId = idInput.value.trim() || "host";
      const uri = uriInput.value.trim();
      const keyfile = isLocalUri(uri) ? "" : keyfileInput.value.trim();
      provisionPanel.style.display = "none";
      provisionPanel.innerHTML = "";
      modalError.textContent = "";
      if (!uri) {
        validateResult.textContent = "URI is required";
        validateResult.className = "setup-validate-result setup-validate-result--error";
        return;
      }
      validateResult.textContent = "Validating…";
      validateResult.className = "setup-validate-result";
      try {
        const res = await validateHost({ host_id: hostId, uri, keyfile });
        if (res.valid) {
          validateResult.textContent = "Valid";
          validateResult.className = "setup-validate-result setup-validate-result--success";
        } else {
          validateResult.textContent = res.error ?? "Validation failed";
          validateResult.className = "setup-validate-result setup-validate-result--error";
          if (needsProvision(res.error)) {
            try {
              const auditRes = await provisionHost({
                host_id: hostId,
                uri,
                keyfile,
                dry_run: true,
              });
              if (
                "audit" in auditRes &&
                auditRes.audit
              ) {
                const audit = auditRes.audit;
                renderProvisionPanel(audit, async (btn) => {
                  btn.disabled = true;
                  btn.textContent = "Provisioning…";
                  try {
                    const execRes = await provisionHost({
                      host_id: hostId,
                      uri,
                      keyfile,
                      dry_run: false,
                    });
                    if ("pool" in execRes || "network" in execRes) {
                      const res = execRes as ProvisionHostResultResponse;
                      const needPool = !!audit.pool;
                      const needNetwork = !!audit.network;
                      const poolOk = !needPool || res.pool?.created;
                      const netOk = !needNetwork || res.network?.created;
                      if (poolOk && netOk) {
                        provisionPanel.innerHTML = "";
                        provisionPanel.style.display = "none";
                        validateResult.textContent = "Validating…";
                        const revalidate = await validateHost({
                          host_id: hostId,
                          uri,
                          keyfile,
                        });
                        if (revalidate.valid) {
                          validateResult.textContent = "Valid";
                          validateResult.className =
                            "setup-validate-result setup-validate-result--success";
                        } else {
                          validateResult.textContent =
                            revalidate.error ?? "Validation failed";
                        }
                      } else {
                        const parts: string[] = [];
                        if (needPool && !res.pool?.created)
                          parts.push(`Pool: ${res.pool?.error ?? "failed"}`);
                        if (needNetwork && !res.network?.created)
                          parts.push(`Network: ${res.network?.error ?? "failed"}`);
                        validateResult.textContent = "Partial failure: " + parts.join("; ");
                        btn.disabled = false;
                        btn.textContent = "Retry";
                      }
                    }
                  } catch (err) {
                    btn.disabled = false;
                    btn.textContent = "Provision";
                    validateResult.textContent =
                      err instanceof ApiError ? err.message : "Provision failed";
                    validateResult.className =
                      "setup-validate-result setup-validate-result--error";
                  }
                });
              }
            } catch {
              /* ignore */
            }
          }
        }
      } catch (err) {
        const msg = err instanceof ApiError ? err.message : "Validation failed";
        validateResult.textContent = msg;
        validateResult.className = "setup-validate-result setup-validate-result--error";
      }
    });

    const closeModal = (): void => {
      overlay.remove();
    };

    const footer = document.createElement("div");
    footer.className = "modal__footer";
    const saveBtn = document.createElement("button");
    saveBtn.type = "button";
    saveBtn.textContent = "Save";
    saveBtn.addEventListener("click", () => {
      const id = idInput.value.trim();
      const uri = uriInput.value.trim();
      if (!id) {
        modalError.textContent = "Host ID is required";
        return;
      }
      if (!uri) {
        modalError.textContent = "URI is required";
        return;
      }
      const keyfile = isLocalUri(uri) ? "" : keyfileInput.value.trim();
      if (isEdit) {
        hosts[editingIndex!] = { id, uri, keyfile };
      } else {
        hosts.push({ id, uri, keyfile });
      }
      renderHostChips();
      updateDefaultHostSelect();
      wrappedCloseModal();
    });
    const removeBtn = document.createElement("button");
    removeBtn.type = "button";
    removeBtn.textContent = "Remove";
    removeBtn.style.display = isEdit ? "" : "none";
    removeBtn.addEventListener("click", () => {
      if (hosts.length <= 1) return;
      hosts.splice(editingIndex!, 1);
      renderHostChips();
      updateDefaultHostSelect();
      wrappedCloseModal();
    });
    const cancelBtn = document.createElement("button");
    cancelBtn.type = "button";
    cancelBtn.textContent = "Cancel";
    cancelBtn.addEventListener("click", closeModal);
    footer.appendChild(removeBtn);
    footer.appendChild(cancelBtn);
    footer.appendChild(saveBtn);
    form.appendChild(footer);
    modal.appendChild(form);
    overlay.appendChild(modal);

    const cleanupFocusTrap = setupFocusTrap(overlay);
    const wrappedCloseModal = (): void => {
      cleanupFocusTrap();
      closeModal();
    };

    overlay.addEventListener("click", (e) => {
      if (e.target === overlay) wrappedCloseModal();
    });
    overlay.addEventListener("keydown", (e) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation();
        wrappedCloseModal();
      }
    });
    closeBtn.addEventListener("click", wrappedCloseModal);
    cancelBtn.addEventListener("click", wrappedCloseModal);

    container.appendChild(overlay);
  }

  renderHostChips();

  const addHostBtn = document.createElement("button");
  addHostBtn.type = "button";
  addHostBtn.textContent = "Add host";
  addHostBtn.addEventListener("click", () => openHostModal());

  hostsSection.appendChild(hostsContainer);
  hostsSection.appendChild(addHostBtn);

  // Default host — Shadcn-style Select
  const defaultSection = document.createElement("div");
  defaultSection.className = "setup-default-host";
  const defaultHostLabel = document.createElement("label");
  defaultHostLabel.id = "setup-default-host-label";
  defaultHostLabel.textContent = "Default host";
  const defaultHostSelectWrapper = document.createElement("div");
  defaultHostSelectWrapper.className = "setup-select";
  const panelId = "setup-default-host-listbox";
  const defaultHostTrigger = document.createElement("button");
  defaultHostTrigger.type = "button";
  defaultHostTrigger.className = "setup-select__trigger";
  defaultHostTrigger.id = "setup-default-host";
  defaultHostTrigger.setAttribute("role", "combobox");
  defaultHostTrigger.setAttribute("aria-describedby", "setup-error");
  defaultHostTrigger.setAttribute("aria-labelledby", "setup-default-host-label");
  defaultHostTrigger.setAttribute("aria-haspopup", "listbox");
  defaultHostTrigger.setAttribute("aria-controls", panelId);
  defaultHostTrigger.setAttribute("aria-expanded", "false");
  defaultHostTrigger.innerHTML = `<span class="setup-select__value">local</span><span class="setup-select__chevron" aria-hidden="true">▾</span>`;
  const defaultHostHidden = document.createElement("input");
  defaultHostHidden.type = "hidden";
  defaultHostHidden.name = "default_host";
  defaultHostHidden.value = "local";
  const defaultHostPanel = document.createElement("div");
  defaultHostPanel.className = "setup-select__panel";
  defaultHostPanel.id = panelId;
  defaultHostPanel.setAttribute("role", "listbox");
  defaultHostPanel.hidden = true;
  defaultHostSelectWrapper.appendChild(defaultHostTrigger);
  defaultHostSelectWrapper.appendChild(defaultHostHidden);
  defaultHostSelectWrapper.appendChild(defaultHostPanel);
  defaultSection.appendChild(defaultHostLabel);
  defaultSection.appendChild(defaultHostSelectWrapper);
  form.appendChild(hostsSection);
  form.appendChild(defaultSection);

  let selectedDefaultHost = "local";
  function setDefaultHostValue(value: string): void {
    selectedDefaultHost = value;
    defaultHostHidden.value = value;
    const display = defaultHostTrigger.querySelector(
      ".setup-select__value"
    ) as HTMLSpanElement | null;
    if (display) display.textContent = value;
  }
  function openDefaultHostPanel(): void {
    defaultHostPanel.hidden = false;
    defaultHostTrigger.setAttribute("aria-expanded", "true");
    clickOutsideCleanup?.();
    const handleClickOutside = (e: MouseEvent): void => {
      if (!defaultHostSelectWrapper.contains(e.target as Node)) {
        closeDefaultHostPanel();
      }
    };
    clickOutsideCleanup = () => {
      document.removeEventListener("click", handleClickOutside);
    };
    setTimeout(() => document.addEventListener("click", handleClickOutside), 0);
    const opts = defaultHostPanel.querySelectorAll(".setup-select__option");
    const idx = Array.from(opts).findIndex(
      (o) => (o as HTMLElement).dataset.value === selectedDefaultHost
    );
    opts.forEach((o, i) => {
      (o as HTMLElement).setAttribute(
        "aria-selected",
        i === idx ? "true" : "false"
      );
    });
    if (opts[idx]) (opts[idx] as HTMLElement).focus();
  }
  let clickOutsideCleanup: (() => void) | null = null;
  function closeDefaultHostPanel(): void {
    defaultHostPanel.hidden = true;
    defaultHostTrigger.setAttribute("aria-expanded", "false");
    defaultHostTrigger.focus();
    clickOutsideCleanup?.();
    clickOutsideCleanup = null;
  }
  defaultHostTrigger.addEventListener("click", () => {
    if (defaultHostPanel.hidden) openDefaultHostPanel();
    else closeDefaultHostPanel();
  });
  defaultHostPanel.addEventListener("keydown", (e) => {
    const opts = Array.from(
      defaultHostPanel.querySelectorAll<HTMLElement>(".setup-select__option")
    );
    const idx = opts.indexOf(document.activeElement as HTMLElement);
    if (e.key === "Escape") {
      e.preventDefault();
      closeDefaultHostPanel();
    } else if (e.key === "ArrowDown" && idx < opts.length - 1) {
      e.preventDefault();
      opts[idx + 1].focus();
      opts.forEach((o, i) =>
        o.setAttribute("aria-selected", i === idx + 1 ? "true" : "false")
      );
    } else if (e.key === "ArrowUp" && idx > 0) {
      e.preventDefault();
      opts[idx - 1].focus();
      opts.forEach((o, i) =>
        o.setAttribute("aria-selected", i === idx - 1 ? "true" : "false")
      );
    } else if (e.key === "Enter" && idx >= 0) {
      e.preventDefault();
      const val = opts[idx].dataset.value;
      if (val) {
        setDefaultHostValue(val);
        closeDefaultHostPanel();
      }
    }
  });
  defaultHostTrigger.addEventListener("keydown", (e) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      if (defaultHostPanel.hidden) openDefaultHostPanel();
      else closeDefaultHostPanel();
    } else if (e.key === "ArrowDown" && defaultHostPanel.hidden) {
      e.preventDefault();
      openDefaultHostPanel();
    }
  });

  function updateDefaultHostSelect(): void {
    defaultHostPanel.innerHTML = "";
    const ids = hosts.map((h) => h.id.trim() || "host");
    ids.forEach((id) => {
      const opt = document.createElement("div");
      opt.className = "setup-select__option";
      opt.setAttribute("role", "option");
      opt.dataset.value = id;
      opt.textContent = id;
      opt.tabIndex = -1;
      opt.addEventListener("click", () => {
        setDefaultHostValue(id);
        closeDefaultHostPanel();
      });
      defaultHostPanel.appendChild(opt);
    });
    if (ids.length > 0 && !ids.includes(selectedDefaultHost)) {
      setDefaultHostValue(ids[0]);
    }
  }
  updateDefaultHostSelect();

  const errorEl = document.createElement("p");
  errorEl.id = "setup-error";
  errorEl.className = "login-error";
  errorEl.setAttribute("role", "alert");
  errorEl.setAttribute("aria-live", "polite");
  form.appendChild(errorEl);

  const submitBtn = document.createElement("button");
  submitBtn.type = "submit";
  submitBtn.textContent = "Complete setup";
  form.appendChild(submitBtn);

  form.addEventListener("submit", async (e) => {
    e.preventDefault();

    const adminUsername = (
      form.querySelector("#setup-admin-username") as HTMLInputElement
    )?.value?.trim();
    const adminPassword = (
      form.querySelector("#setup-admin-password") as HTMLInputElement
    )?.value;
    const adminPasswordConfirm = (
      form.querySelector("#setup-admin-password-confirm") as HTMLInputElement
    )?.value;

    if (!adminUsername || !adminPassword) {
      errorEl.textContent = "Admin username and password are required";
      return;
    }
    if (adminPassword !== adminPasswordConfirm) {
      errorEl.textContent = "Passwords do not match";
      return;
    }

    const hostsToSubmit = hosts.map((h) => ({
      id: h.id.trim(),
      uri: h.uri.trim(),
      keyfile: isLocalUri(h.uri) ? "" : h.keyfile.trim(),
    }));
    for (const h of hostsToSubmit) {
      if (h.id === "") {
        errorEl.textContent = "Host ID is required";
        return;
      }
      if (!h.uri) {
        errorEl.textContent = "All hosts must have a URI";
        return;
      }
    }

    if (hostsToSubmit.length === 0) {
      errorEl.textContent = "At least one host is required";
      return;
    }

    const defaultHost = defaultHostHidden.value;
    if (!hostsToSubmit.some((h) => h.id === defaultHost)) {
      errorEl.textContent = "Default host must be one of the configured hosts";
      return;
    }

    const body: SetupCompleteRequest = {
      admin: { username: adminUsername, password: adminPassword },
      hosts: hostsToSubmit,
      default_host: defaultHost,
    };

    errorEl.textContent = "";
    submitBtn.disabled = true;
    try {
      await setupComplete(body);
      onSuccess();
    } catch (err) {
      submitBtn.disabled = false;
      const msg =
        err instanceof ApiError ? err.message : "Setup failed. Please try again.";
      errorEl.textContent = msg;
    }
  });

  container.appendChild(form);
}
