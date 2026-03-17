/**
 * Setup wizard for first-run configuration.
 * Collects admin credentials and host config; validates before saving.
 */
import {
  ApiError,
  setupComplete,
  validateHost,
  type SetupCompleteRequest,
} from "../lib/api";

export function renderSetupWizard(
  container: HTMLElement,
  onSuccess: () => void
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

  let hostIndex = 0;

  function getHostInputs(row: Element): {
    id: HTMLInputElement;
    uri: HTMLInputElement;
    keyfile: HTMLInputElement;
  } | null {
    const inputs = row.querySelectorAll("input[type='text']");
    if (inputs.length < 3) return null;
    return {
      id: inputs[0] as HTMLInputElement,
      uri: inputs[1] as HTMLInputElement,
      keyfile: inputs[2] as HTMLInputElement,
    };
  }

  function updateDefaultHostSelect(): void {
    defaultHostSelect.innerHTML = "";
    const rows = hostsContainer.querySelectorAll(".setup-host-row");
    rows.forEach((row) => {
      const inps = getHostInputs(row);
      const id = inps ? inps.id.value.trim() || "host" : "host";
      const opt = document.createElement("option");
      opt.value = id;
      opt.textContent = id;
      defaultHostSelect.appendChild(opt);
    });
  }

  function addHostRow(initial?: { id: string; uri: string; keyfile: string }): void {
    const idx = hostIndex++;
    const row = document.createElement("div");
    row.className = "setup-host-row";
    row.innerHTML = `
      <label for="setup-host-id-${idx}">Host ID</label>
      <input id="setup-host-id-${idx}" type="text" placeholder="e.g. local" aria-describedby="setup-error" />
      <label for="setup-host-uri-${idx}">URI</label>
      <input id="setup-host-uri-${idx}" type="text" placeholder="e.g. qemu:///system or qemu+ssh://user@host/system" aria-describedby="setup-error" />
      <label for="setup-host-keyfile-${idx}">SSH key path (empty for local)</label>
      <input id="setup-host-keyfile-${idx}" type="text" placeholder="/path/to/key" aria-describedby="setup-error" />
      <button type="button" class="setup-validate-btn">Validate host</button>
      <p class="setup-validate-result" role="status" aria-live="polite"></p>
      <button type="button" class="setup-remove-host">Remove</button>
    `;

    const inps = getHostInputs(row)!;
    if (initial) {
      inps.id.value = initial.id;
      inps.uri.value = initial.uri;
      inps.keyfile.value = initial.keyfile;
    }

    const validateBtn = row.querySelector(".setup-validate-btn") as HTMLButtonElement;
    const validateResult = row.querySelector(".setup-validate-result") as HTMLParagraphElement;
    const removeBtn = row.querySelector(".setup-remove-host") as HTMLButtonElement;

    validateBtn.addEventListener("click", async () => {
      const hostId = inps.id.value.trim() || "host";
      const uri = inps.uri.value.trim();
      const keyfile = inps.keyfile.value.trim();
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
        }
      } catch (err) {
        const msg = err instanceof ApiError ? err.message : "Validation failed";
        validateResult.textContent = msg;
        validateResult.className = "setup-validate-result setup-validate-result--error";
      }
    });

    removeBtn.addEventListener("click", () => {
      const rows = hostsContainer.querySelectorAll(".setup-host-row");
      if (rows.length <= 1) return;
      row.remove();
      updateDefaultHostSelect();
    });

    inps.id.addEventListener("input", updateDefaultHostSelect);
    inps.id.addEventListener("change", updateDefaultHostSelect);

    hostsContainer.appendChild(row);
  }

  addHostRow({ id: "local", uri: "qemu:///system", keyfile: "" });

  const addHostBtn = document.createElement("button");
  addHostBtn.type = "button";
  addHostBtn.textContent = "Add host";
  addHostBtn.addEventListener("click", () => {
    addHostRow();
    updateDefaultHostSelect();
  });

  hostsSection.appendChild(hostsContainer);
  hostsSection.appendChild(addHostBtn);

  // Default host
  const defaultSection = document.createElement("div");
  defaultSection.className = "setup-default-host";
  const defaultHostLabel = document.createElement("label");
  defaultHostLabel.htmlFor = "setup-default-host";
  defaultHostLabel.textContent = "Default host";
  const defaultHostSelect = document.createElement("select");
  defaultHostSelect.id = "setup-default-host";
  defaultHostSelect.name = "default_host";
  defaultHostSelect.setAttribute("aria-describedby", "setup-error");
  defaultHostSelect.innerHTML = '<option value="local">local</option>';
  defaultSection.appendChild(defaultHostLabel);
  defaultSection.appendChild(defaultHostSelect);
  form.appendChild(hostsSection);
  form.appendChild(defaultSection);

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

    const hosts: Array<{ id: string; uri: string; keyfile: string }> = [];
    const rows = hostsContainer.querySelectorAll(".setup-host-row");
    for (const row of rows) {
      const inps = getHostInputs(row);
      if (!inps) continue;
      const id = inps.id.value.trim();
      if (id === "") {
        errorEl.textContent = "Host ID is required";
        return;
      }
      const uri = inps.uri.value.trim();
      if (!uri) {
        errorEl.textContent = "All hosts must have a URI";
        return;
      }
      hosts.push({
        id,
        uri,
        keyfile: inps.keyfile.value.trim(),
      });
    }

    if (hosts.length === 0) {
      errorEl.textContent = "At least one host is required";
      return;
    }

    const defaultHost = defaultHostSelect.value;
    if (!hosts.some((h) => h.id === defaultHost)) {
      errorEl.textContent = "Default host must be one of the configured hosts";
      return;
    }

    const body: SetupCompleteRequest = {
      admin: { username: adminUsername, password: adminPassword },
      hosts,
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
