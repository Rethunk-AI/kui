/**
 * Clone VM modal with InlineHostSelector for target host.
 * Spec §3: contextual host override; POST /api/hosts/{id}/vms/{uuid}/clone.
 */
import {
  ApiError,
  cloneVM,
  fetchHostPools,
  type Host,
  type VM,
} from "../lib/api";
import { addAlert } from "../lib/alerts";
import { setupFocusTrap } from "../lib/focus-trap";
import { renderInlineHostSelector } from "./InlineHostSelector";

export interface CloneVMModalProps {
  sourceVM: VM;
  hosts: Host[];
  defaultHostId: string | null;
  onClose: () => void;
  onSuccess: () => void;
}

export function renderCloneVMModal(
  container: HTMLElement,
  props: CloneVMModalProps
): void {
  const { sourceVM, hosts, defaultHostId, onClose, onSuccess } = props;
  const selectedHostId =
    defaultHostId ?? hosts.find((h) => h.id !== sourceVM.host_id)?.id ?? hosts[0]?.id ?? null;

  container.innerHTML = "";
  const overlay = document.createElement("div");
  overlay.className = "modal-overlay";
  overlay.setAttribute("role", "dialog");
  overlay.setAttribute("aria-modal", "true");
  overlay.setAttribute("aria-labelledby", "clone-vm-modal-title");

  const modal = document.createElement("div");
  modal.className = "modal";

  const header = document.createElement("div");
  header.className = "modal__header";
  const title = document.createElement("h2");
  title.id = "clone-vm-modal-title";
  title.textContent = "Clone VM";
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
  formError.id = "clone-vm-form-error";
  formError.style.display = "none";
  form.appendChild(formError);

  const sourceInfo = document.createElement("div");
  sourceInfo.className = "modal__field";
  sourceInfo.innerHTML = `
    <span class="modal__label">Source</span>
    <span class="modal__readonly">${escapeHtml(sourceVM.display_name ?? sourceVM.libvirt_uuid)} (${escapeHtml(sourceVM.host_id)})</span>
  `;
  form.appendChild(sourceInfo);

  const hostSelectorContainer = document.createElement("div");
  hostSelectorContainer.className = "modal__field";
  renderInlineHostSelector(hostSelectorContainer, {
    hosts,
    selectedHostId,
    onChange: (id) => {
      selectedHostIdRef.current = id;
      loadPools(id);
    },
    label: "Target host",
    required: true,
    ariaDescribedBy: "clone-vm-form-error",
  });
  form.appendChild(hostSelectorContainer);

  const selectedHostIdRef = { current: selectedHostId };

  const poolSelect = document.createElement("select");
  poolSelect.name = "target_pool";
  poolSelect.required = true;
  poolSelect.setAttribute("aria-required", "true");
  poolSelect.setAttribute("aria-label", "Target pool");
  poolSelect.setAttribute("aria-describedby", "clone-vm-form-error");
  const poolLabel = document.createElement("label");
  poolLabel.className = "modal__label";
  poolLabel.textContent = "Target pool";
  poolLabel.appendChild(poolSelect);
  const poolField = document.createElement("div");
  poolField.className = "modal__field";
  poolField.appendChild(poolLabel);
  const poolHint = document.createElement("p");
  poolHint.className = "modal__hint";
  poolHint.id = "clone-vm-pool-hint";
  poolHint.style.display = "none";
  poolHint.textContent = "No storage pools on this host. Create one in virt-manager or virsh.";
  poolField.appendChild(poolHint);
  form.appendChild(poolField);

  const nameInput = document.createElement("input");
  nameInput.type = "text";
  nameInput.name = "target_name";
  nameInput.placeholder = `${sourceVM.display_name ?? sourceVM.libvirt_uuid}-clone`;
  nameInput.setAttribute("aria-describedby", "clone-vm-form-error");
  const nameLabel = document.createElement("label");
  nameLabel.className = "modal__label";
  nameLabel.textContent = "Clone name";
  nameLabel.appendChild(nameInput);
  const nameField = document.createElement("div");
  nameField.className = "modal__field";
  nameField.appendChild(nameLabel);
  form.appendChild(nameField);

  async function loadPools(hostId: string): Promise<void> {
    poolSelect.innerHTML = "";
    poolSelect.appendChild(createOption("", "Loading…"));
    try {
      const poolsResp = await fetchHostPools(hostId);
      poolSelect.innerHTML = "";
      poolSelect.appendChild(createOption("", "Select pool"));
      for (const p of poolsResp) {
        poolSelect.appendChild(createOption(p.name, p.name));
      }
      if (poolsResp.length === 0) {
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
        err instanceof ApiError ? err.message : "Failed to load pools",
        err instanceof ApiError ? String(err.status) : undefined
      );
    }
  }

  if (selectedHostId) {
    loadPools(selectedHostId);
  }

  const footer = document.createElement("div");
  footer.className = "modal__footer";
  const cancelBtn = document.createElement("button");
  cancelBtn.type = "button";
  cancelBtn.textContent = "Cancel";
  const submitBtn = document.createElement("button");
  submitBtn.type = "submit";
  submitBtn.textContent = "Clone";
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
    nameInput.removeAttribute("aria-invalid");
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
    const targetHostId = selectedHostIdRef.current;
    if (!targetHostId) {
      showError("Select a target host", hostSelect ?? formError);
      return;
    }
    const targetPool = poolSelect.value?.trim();
    if (!targetPool) {
      showError("Select a target pool", poolSelect);
      return;
    }
    submitBtn.disabled = true;
    try {
      await cloneVM(sourceVM.host_id, sourceVM.libvirt_uuid, {
        target_host_id: targetHostId,
        target_pool: targetPool,
        target_name: nameInput.value?.trim() || undefined,
      });
      onSuccess();
      wrappedOnClose();
    } catch (err) {
      submitBtn.disabled = false;
      if (err instanceof ApiError && err.status === 401) return;
      const msg = err instanceof ApiError ? err.message : "Failed to clone VM";
      showError(msg, poolSelect);
      addAlert(
        "clone_failure",
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
}

function createOption(value: string, text: string): HTMLOptionElement {
  const opt = document.createElement("option");
  opt.value = value;
  opt.textContent = text;
  return opt;
}

function escapeHtml(s: string): string {
  const div = document.createElement("div");
  div.textContent = s;
  return div.innerHTML;
}
