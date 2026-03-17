/**
 * Toast notifications for transient feedback. Auto-dismiss after a few seconds.
 * Spec §4: transient events (host.online, vm.state_changed) use toast.
 */

const TOAST_DURATION_MS = 4000;
const TOAST_CONTAINER_ID = "kui-toast-container";
const TOAST_ANNOUNCER_ID = "kui-toast-announcer";

function ensureAnnouncer(): HTMLElement {
  let el = document.getElementById(TOAST_ANNOUNCER_ID);
  if (!el) {
    el = document.createElement("div");
    el.id = TOAST_ANNOUNCER_ID;
    el.setAttribute("aria-live", "polite");
    el.setAttribute("aria-atomic", "true");
    el.setAttribute("role", "status");
    el.className = "sr-only";
    document.body.appendChild(el);
  }
  return el;
}

function ensureContainer(): HTMLElement {
  let el = document.getElementById(TOAST_CONTAINER_ID);
  if (!el) {
    el = document.createElement("div");
    el.id = TOAST_CONTAINER_ID;
    el.className = "toast-container";
    document.body.appendChild(el);
  }
  return el;
}

export function showToast(message: string, type: "info" | "success" | "warn" = "info"): void {
  const announcer = ensureAnnouncer();
  announcer.textContent = message;
  setTimeout(() => {
    announcer.textContent = "";
  }, 1000);

  const container = ensureContainer();
  const toast = document.createElement("div");
  toast.className = `toast toast--${type}`;
  toast.textContent = message;
  container.appendChild(toast);

  const timer = setTimeout(() => {
    toast.classList.add("toast--dismissing");
    setTimeout(() => {
      toast.remove();
    }, 200);
  }, TOAST_DURATION_MS);

  toast.addEventListener("click", () => {
    clearTimeout(timer);
    toast.classList.add("toast--dismissing");
    setTimeout(() => toast.remove(), 200);
  });
}
