/**
 * Domain XML editor modal.
 * Fetches domain XML via GET, edits in textarea, saves via PUT.
 * VM must be stopped to save.
 */
import {
  ApiError,
  fetchDomainXML,
  putDomainXML,
  type VM,
} from "../lib/api";
import { addAlert } from "../lib/alerts";
import { setupFocusTrap } from "../lib/focus-trap";

export interface DomainXMLEditorProps {
  vm: VM;
  onClose: () => void;
  onSuccess: () => void;
}

export function renderDomainXMLEditor(
  container: HTMLElement,
  props: DomainXMLEditorProps
): void {
  const { vm, onClose, onSuccess } = props;

  container.innerHTML = "";
  const overlay = document.createElement("div");
  overlay.className = "modal-overlay";
  overlay.setAttribute("role", "dialog");
  overlay.setAttribute("aria-modal", "true");
  overlay.setAttribute("aria-labelledby", "domain-xml-modal-title");

  const modal = document.createElement("div");
  modal.className = "modal modal--wide";

  const header = document.createElement("div");
  header.className = "modal__header";
  const title = document.createElement("h2");
  title.id = "domain-xml-modal-title";
  title.textContent = `Edit XML: ${escapeHtml(vm.display_name ?? vm.libvirt_uuid)}`;
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
  formError.id = "domain-xml-form-error";
  formError.style.display = "none";
  form.appendChild(formError);

  const note = document.createElement("p");
  note.className = "modal__readonly";
  note.textContent = "VM must be stopped to save. Invalid XML or forbidden elements (qemu:commandline, qemu:arg, qemu:env, qemu:init) will be rejected.";
  form.appendChild(note);

  const label = document.createElement("label");
  label.className = "modal__label";
  label.htmlFor = "domain-xml-textarea";
  label.textContent = "Domain XML";
  form.appendChild(label);

  const textarea = document.createElement("textarea");
  textarea.id = "domain-xml-textarea";
  textarea.name = "domain_xml";
  textarea.rows = 20;
  textarea.placeholder = "Loading…";
  textarea.setAttribute("aria-describedby", "domain-xml-form-error");
  textarea.setAttribute("aria-label", "Domain XML");
  textarea.className = "modal__textarea";
  form.appendChild(textarea);

  const footer = document.createElement("div");
  footer.className = "modal__footer";
  const cancelBtn = document.createElement("button");
  cancelBtn.type = "button";
  cancelBtn.textContent = "Cancel";
  const submitBtn = document.createElement("button");
  submitBtn.type = "submit";
  submitBtn.textContent = "Save";
  footer.appendChild(cancelBtn);
  footer.appendChild(submitBtn);
  form.appendChild(footer);

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

  const showError = (msg: string): void => {
    formError.textContent = msg;
    formError.style.display = "";
    textarea.setAttribute("aria-invalid", "true");
  };

  const clearError = (): void => {
    formError.textContent = "";
    formError.style.display = "none";
    textarea.removeAttribute("aria-invalid");
  };

  (async () => {
    try {
      const xml = await fetchDomainXML(vm.host_id, vm.libvirt_uuid);
      textarea.value = xml;
      textarea.placeholder = "";
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : "Failed to load domain XML";
      showError(msg);
      addAlert(
        "api_error",
        msg,
        err instanceof ApiError ? String(err.status) : undefined
      );
      textarea.placeholder = "Failed to load";
    }
  })();

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    clearError();
    const xml = textarea.value.trim();
    if (!xml) {
      showError("Domain XML cannot be empty");
      return;
    }
    submitBtn.disabled = true;
    try {
      await putDomainXML(vm.host_id, vm.libvirt_uuid, xml);
      onSuccess();
      wrappedOnClose();
    } catch (err) {
      submitBtn.disabled = false;
      let msg = err instanceof ApiError ? err.message : "Failed to save domain XML";
      try {
        if (err instanceof ApiError && err.message) {
          const parsed = JSON.parse(err.message);
          if (parsed?.error) msg = parsed.error;
        }
      } catch {
        // keep original msg
      }
      showError(msg);
      addAlert(
        "api_error",
        msg,
        err instanceof ApiError ? String(err.status) : undefined
      );
    }
  });

  container.appendChild(overlay);
}

function escapeHtml(s: string): string {
  const div = document.createElement("div");
  div.textContent = s;
  return div.innerHTML;
}
