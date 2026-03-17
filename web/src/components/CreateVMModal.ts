/**
 * Create VM modal with InlineHostSelector.
 * Spec §3: contextual host override; POST /api/vms.
 */
import {
  ApiError,
  createVM,
  fetchHostNetworks,
  fetchHostPools,
  fetchHostPoolVolumes,
  type Host,
  type Network,
  type Pool,
  type Volume,
} from "../lib/api";
import { addAlert } from "../lib/alerts";
import { setupFocusTrap } from "../lib/focus-trap";
import { renderInlineHostSelector } from "./InlineHostSelector";

export interface CreateVMModalProps {
  hosts: Host[];
  defaultHostId: string | null;
  onClose: () => void;
  onSuccess: () => void;
}

export function renderCreateVMModal(
  container: HTMLElement,
  props: CreateVMModalProps
): void {
  const { hosts, defaultHostId, onClose, onSuccess } = props;
  const selectedHostId = defaultHostId ?? hosts[0]?.id ?? null;

  container.innerHTML = "";
  const overlay = document.createElement("div");
  overlay.className = "modal-overlay";
  overlay.setAttribute("role", "dialog");
  overlay.setAttribute("aria-modal", "true");
  overlay.setAttribute("aria-labelledby", "create-vm-modal-title");

  const modal = document.createElement("div");
  modal.className = "modal";

  const header = document.createElement("div");
  header.className = "modal__header";
  const title = document.createElement("h2");
  title.id = "create-vm-modal-title";
  title.textContent = "Create VM";
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

  const formError = document.createElement("p");
  formError.className = "modal__error";
  formError.setAttribute("role", "alert");
  formError.id = "create-vm-form-error";
  formError.style.display = "none";
  form.appendChild(formError);

  const hostSelectorContainer = document.createElement("div");
  hostSelectorContainer.className = "modal__field";
  renderInlineHostSelector(hostSelectorContainer, {
    hosts,
    selectedHostId,
    onChange: (id) => {
      selectedHostIdRef.current = id;
      loadPoolsAndNetworks(id);
    },
    label: "Host",
    required: true,
    ariaDescribedBy: "create-vm-form-error",
  });
  form.appendChild(hostSelectorContainer);

  const selectedHostIdRef = { current: selectedHostId };

  const poolSelect = document.createElement("select");
  poolSelect.name = "pool";
  poolSelect.required = true;
  poolSelect.setAttribute("aria-required", "true");
  poolSelect.setAttribute("aria-label", "Storage pool");
  poolSelect.setAttribute("aria-describedby", "create-vm-form-error");
  const poolLabel = document.createElement("label");
  poolLabel.className = "modal__label";
  poolLabel.textContent = "Storage pool";
  poolLabel.appendChild(poolSelect);
  const poolField = document.createElement("div");
  poolField.className = "modal__field";
  poolField.appendChild(poolLabel);
  const poolHint = document.createElement("p");
  poolHint.className = "modal__hint";
  poolHint.id = "create-vm-pool-hint";
  poolHint.style.display = "none";
  poolHint.textContent = "No storage pools on this host. Create one in virt-manager or virsh.";
  poolField.appendChild(poolHint);
  form.appendChild(poolField);

  const diskModeGroup = document.createElement("div");
  diskModeGroup.className = "modal__field";
  diskModeGroup.setAttribute("role", "radiogroup");
  const diskModeLabel = document.createElement("span");
  diskModeLabel.className = "modal__label";
  diskModeLabel.id = "create-vm-disk-label";
  diskModeLabel.textContent = "Disk";
  diskModeGroup.setAttribute("aria-labelledby", "create-vm-disk-label");
  diskModeGroup.appendChild(diskModeLabel);
  const diskExisting = document.createElement("input");
  diskExisting.type = "radio";
  diskExisting.name = "disk_mode";
  diskExisting.id = "disk-existing";
  diskExisting.value = "existing";
  diskExisting.checked = true;
  const diskNew = document.createElement("input");
  diskNew.type = "radio";
  diskNew.name = "disk_mode";
  diskNew.id = "disk-new";
  diskNew.value = "new";
  const diskExistingLabel = document.createElement("label");
  diskExistingLabel.htmlFor = "disk-existing";
  diskExistingLabel.textContent = "Use existing volume";
  diskExistingLabel.prepend(diskExisting);
  const diskNewLabel = document.createElement("label");
  diskNewLabel.htmlFor = "disk-new";
  diskNewLabel.textContent = "Create new disk";
  diskNewLabel.prepend(diskNew);
  diskModeGroup.appendChild(diskExistingLabel);
  diskModeGroup.appendChild(diskNewLabel);
  form.appendChild(diskModeGroup);

  const volumeSelect = document.createElement("select");
  volumeSelect.name = "volume";
  volumeSelect.setAttribute("aria-label", "Volume");
  volumeSelect.setAttribute("aria-describedby", "create-vm-form-error");
  const volumeLabel = document.createElement("label");
  volumeLabel.className = "modal__label";
  volumeLabel.textContent = "Volume";
  volumeLabel.appendChild(volumeSelect);
  const volumeField = document.createElement("div");
  volumeField.className = "modal__field modal__field--volume";
  volumeField.appendChild(volumeLabel);
  form.appendChild(volumeField);

  const sizeField = document.createElement("div");
  sizeField.className = "modal__field modal__field--size";
  sizeField.style.display = "none";
  const sizeLabel = document.createElement("label");
  sizeLabel.className = "modal__label";
  sizeLabel.textContent = "Size (MB)";
  const sizeInput = document.createElement("input");
  sizeInput.setAttribute("aria-describedby", "create-vm-form-error");
  sizeInput.type = "number";
  sizeInput.name = "size_mb";
  sizeInput.min = "1";
  sizeInput.placeholder = "2048";
  sizeLabel.appendChild(sizeInput);
  sizeField.appendChild(sizeLabel);
  form.appendChild(sizeField);

  const networkSelect = document.createElement("select");
  networkSelect.name = "network";
  networkSelect.setAttribute("aria-label", "Network");
  const networkLabel = document.createElement("label");
  networkLabel.className = "modal__label";
  networkLabel.textContent = "Network";
  networkLabel.appendChild(networkSelect);
  const networkField = document.createElement("div");
  networkField.className = "modal__field";
  networkField.appendChild(networkLabel);
  form.appendChild(networkField);

  const displayNameInput = document.createElement("input");
  displayNameInput.type = "text";
  displayNameInput.name = "display_name";
  displayNameInput.placeholder = "Display name (optional)";
  const displayNameLabel = document.createElement("label");
  displayNameLabel.className = "modal__label";
  displayNameLabel.textContent = "Display name";
  displayNameLabel.appendChild(displayNameInput);
  const displayNameField = document.createElement("div");
  displayNameField.className = "modal__field";
  displayNameField.appendChild(displayNameLabel);
  form.appendChild(displayNameField);

  let pools: Pool[] = [];
  let volumes: Volume[] = [];
  let networks: Network[] = [];

  function toggleDiskFields(): void {
    const isExisting = diskExisting.checked;
    volumeField.style.display = isExisting ? "" : "none";
    sizeField.style.display = isExisting ? "none" : "";
    volumeSelect.required = isExisting;
    sizeInput.required = !isExisting;
  }

  diskExisting.addEventListener("change", toggleDiskFields);
  diskNew.addEventListener("change", toggleDiskFields);

  async function loadPoolsAndNetworks(hostId: string): Promise<void> {
    poolSelect.innerHTML = "";
    volumeSelect.innerHTML = "";
    networkSelect.innerHTML = "";
    poolSelect.appendChild(createOption("", "Loading…"));
    try {
      const [poolsResp, networksResp] = await Promise.all([
        fetchHostPools(hostId),
        fetchHostNetworks(hostId),
      ]);
      pools = poolsResp;
      networks = networksResp;
      poolSelect.innerHTML = "";
      poolSelect.appendChild(createOption("", "Select pool"));
      for (const p of pools) {
        poolSelect.appendChild(createOption(p.name, p.name));
      }
      networkSelect.innerHTML = "";
      networkSelect.appendChild(createOption("", "Select network"));
      for (const n of networks) {
        networkSelect.appendChild(createOption(n.name, n.name));
      }
      if (networks.length > 0) {
        const defaultNet = networks.find((n) => n.name === "default") ?? networks[0];
        networkSelect.value = defaultNet.name;
      }
      if (pools.length === 0) {
        poolHint.style.display = "";
        submitBtn.disabled = true;
      } else {
        poolHint.style.display = "none";
        submitBtn.disabled = false;
      }
    } catch (err) {
      poolSelect.innerHTML = "";
      poolSelect.appendChild(createOption("", "Failed to load"));
      poolHint.style.display = "none";
      submitBtn.disabled = true;
      if (err instanceof ApiError && err.status === 401) return;
      addAlert(
        "api_error",
        err instanceof ApiError ? err.message : "Failed to load pools/networks",
        err instanceof ApiError ? String(err.status) : undefined
      );
    }
  }

  async function loadVolumes(hostId: string, poolName: string): Promise<void> {
    volumeSelect.innerHTML = "";
    volumeSelect.appendChild(createOption("", "Loading…"));
    try {
      volumes = await fetchHostPoolVolumes(hostId, poolName);
      volumeSelect.innerHTML = "";
      volumeSelect.appendChild(createOption("", "Select volume"));
      for (const v of volumes) {
        volumeSelect.appendChild(createOption(v.name, v.name));
      }
    } catch (err) {
      volumeSelect.innerHTML = "";
      volumeSelect.appendChild(createOption("", "Failed to load"));
      if (err instanceof ApiError && err.status === 401) return;
      addAlert(
        "api_error",
        err instanceof ApiError ? err.message : "Failed to load volumes",
        err instanceof ApiError ? String(err.status) : undefined
      );
    }
  }

  poolSelect.addEventListener("change", () => {
    const hostId = selectedHostIdRef.current;
    const poolName = poolSelect.value;
    if (hostId && poolName) {
      loadVolumes(hostId, poolName);
    } else {
      volumeSelect.innerHTML = "";
      volumeSelect.appendChild(createOption("", "Select pool first"));
    }
  });

  if (selectedHostId) {
    loadPoolsAndNetworks(selectedHostId);
  }

  const footer = document.createElement("div");
  footer.className = "modal__footer";
  const cancelBtn = document.createElement("button");
  cancelBtn.type = "button";
  cancelBtn.textContent = "Cancel";
  const submitBtn = document.createElement("button");
  submitBtn.type = "submit";
  submitBtn.textContent = "Create";
  submitBtn.disabled = true;
  footer.appendChild(cancelBtn);
  footer.appendChild(submitBtn);
  form.appendChild(footer);

  const hostSelect = hostSelectorContainer.querySelector("select");
  const clearInvalid = (): void => {
    formError.textContent = "";
    formError.style.display = "none";
    hostSelect?.removeAttribute("aria-invalid");
    poolSelect.removeAttribute("aria-invalid");
    volumeSelect.removeAttribute("aria-invalid");
    sizeInput.removeAttribute("aria-invalid");
  };
  const showError = (msg: string, field: HTMLElement): void => {
    clearInvalid();
    formError.textContent = msg;
    formError.style.display = "";
    field.setAttribute("aria-invalid", "true");
  };

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    clearInvalid();
    const hostId = selectedHostIdRef.current;
    if (!hostId) {
      showError("Select a host", hostSelect ?? formError);
      return;
    }
    const pool = poolSelect.value?.trim();
    if (!pool) {
      showError("Select a storage pool", poolSelect);
      return;
    }
    const isExisting = diskExisting.checked;
    let diskName = "";
    let sizeMB = 0;
    if (isExisting) {
      diskName = volumeSelect.value?.trim() ?? "";
      if (!diskName) {
        showError("Select a volume", volumeSelect);
        return;
      }
    } else {
      sizeMB = parseInt(sizeInput.value, 10);
      if (!Number.isFinite(sizeMB) || sizeMB <= 0) {
        showError("Enter a valid disk size (MB)", sizeInput);
        return;
      }
    }
    submitBtn.disabled = true;
    try {
      await createVM({
        host_id: hostId,
        pool,
        disk: isExisting ? { name: diskName } : { size_mb: sizeMB },
        network: networkSelect.value?.trim() || undefined,
        display_name: displayNameInput.value?.trim() || undefined,
      });
      onSuccess();
      wrappedOnClose();
    } catch (err) {
      submitBtn.disabled = false;
      if (err instanceof ApiError && err.status === 401) return;
      const msg = err instanceof ApiError ? err.message : "Failed to create VM";
      showError(msg, poolSelect);
      addAlert(
        "create_failure",
        msg,
        err instanceof ApiError ? String(err.status) : undefined
      );
    }
  });

  modal.appendChild(form);
  overlay.appendChild(modal);

  const cleanupFocusTrap = setupFocusTrap(overlay);
  const wrappedOnClose = (): void => {
    cleanupFocusTrap();
    onClose();
  };

  overlay.addEventListener("click", (e) => {
    if (e.target === overlay) wrappedOnClose();
  });

  overlay.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      wrappedOnClose();
    }
  });

  closeBtn.addEventListener("click", wrappedOnClose);
  cancelBtn.addEventListener("click", wrappedOnClose);

  container.appendChild(overlay);
  toggleDiskFields();
}

function createOption(value: string, text: string): HTMLOptionElement {
  const opt = document.createElement("option");
  opt.value = value;
  opt.textContent = text;
  return opt;
}
