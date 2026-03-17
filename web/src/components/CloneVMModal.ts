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
  closeBtn.addEventListener("click", onClose);
  header.appendChild(closeBtn);
  modal.appendChild(header);

  const form = document.createElement("form");
  form.className = "modal__form";

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
  });
  form.appendChild(hostSelectorContainer);

  const selectedHostIdRef = { current: selectedHostId };

  const poolSelect = document.createElement("select");
  poolSelect.name = "target_pool";
  poolSelect.required = true;
  poolSelect.setAttribute("aria-label", "Target pool");
  const poolLabel = document.createElement("label");
  poolLabel.className = "modal__label";
  poolLabel.textContent = "Target pool";
  poolLabel.appendChild(poolSelect);
  const poolField = document.createElement("div");
  poolField.className = "modal__field";
  poolField.appendChild(poolLabel);
  form.appendChild(poolField);

  const nameInput = document.createElement("input");
  nameInput.type = "text";
  nameInput.name = "target_name";
  nameInput.placeholder = `${sourceVM.display_name ?? sourceVM.libvirt_uuid}-clone`;
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
    } catch (err) {
      poolSelect.innerHTML = "";
      poolSelect.appendChild(createOption("", "Failed to load"));
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
  cancelBtn.addEventListener("click", onClose);
  const submitBtn = document.createElement("button");
  submitBtn.type = "submit";
  submitBtn.textContent = "Clone";
  footer.appendChild(cancelBtn);
  footer.appendChild(submitBtn);
  form.appendChild(footer);

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const targetHostId = selectedHostIdRef.current;
    if (!targetHostId) {
      addAlert("clone_failure", "Select a target host", undefined);
      return;
    }
    const targetPool = poolSelect.value?.trim();
    if (!targetPool) {
      addAlert("clone_failure", "Select a target pool", undefined);
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
      onClose();
    } catch (err) {
      submitBtn.disabled = false;
      addAlert(
        "clone_failure",
        err instanceof ApiError ? err.message : "Failed to clone VM",
        err instanceof ApiError ? String(err.status) : undefined
      );
    }
  });

  modal.appendChild(form);
  overlay.appendChild(modal);

  overlay.addEventListener("click", (e) => {
    if (e.target === overlay) onClose();
  });

  overlay.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      onClose();
    }
  });

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
