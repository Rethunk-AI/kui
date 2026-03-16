/**
 * VM list UI with month/year grouping.
 * Spec §6: flat list, group by last_access (or created_at fallback), display_name, status, relative-time.
 * Orphans in separate section with Claim.
 */
import { ApiError, claimVM, type VM, type Orphan, type VMsResponse } from "../lib/api";
import { addAlert } from "../lib/alerts";

const GROUP_BY_LAST_ACCESS = "last_access";
const GROUP_BY_CREATED_AT = "created_at";

function parseISODate(s: string | null | undefined): Date | null {
  if (!s || s.trim() === "") return null;
  const d = new Date(s);
  return isNaN(d.getTime()) ? null : d;
}

function getGroupKey(vm: VM, groupBy: string): string {
  const dt =
    groupBy === GROUP_BY_LAST_ACCESS
      ? parseISODate(vm.last_access) ?? parseISODate(vm.created_at)
      : parseISODate(vm.created_at) ?? parseISODate(vm.last_access);
  if (!dt) return "Unknown";
  const y = dt.getFullYear();
  const m = dt.getMonth() + 1;
  return `${y}-${String(m).padStart(2, "0")}`;
}

function formatGroupHeader(key: string): string {
  if (key === "Unknown") return "Unknown";
  const [y, m] = key.split("-").map(Number);
  const monthNames = [
    "January", "February", "March", "April", "May", "June",
    "July", "August", "September", "October", "November", "December",
  ];
  return `${monthNames[m - 1]} ${y}`;
}

function relativeTime(date: Date): string {
  const now = new Date();
  const sec = Math.floor((now.getTime() - date.getTime()) / 1000);
  if (sec < 60) return "just now";
  if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
  if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`;
  if (sec < 2592000) return `${Math.floor(sec / 86400)}d ago`;
  if (sec < 31536000) return `${Math.floor(sec / 2592000)}mo ago`;
  return `${Math.floor(sec / 31536000)}y ago`;
}

function getRelativeTimeLabel(vm: VM): string {
  if (vm.status === "running") {
    const started = parseISODate(vm.updated_at) ?? parseISODate(vm.last_access);
    if (started) return `started ${relativeTime(started)}`;
  }
  const dt = parseISODate(vm.last_access) ?? parseISODate(vm.created_at);
  if (dt) return `accessed ${relativeTime(dt)}`;
  return "";
}

function groupVMs(vms: VM[], groupBy: string): Map<string, VM[]> {
  const map = new Map<string, VM[]>();
  for (const vm of vms) {
    const key = getGroupKey(vm, groupBy);
    const arr = map.get(key) ?? [];
    arr.push(vm);
    map.set(key, arr);
  }
  const sortedKeys = [...map.keys()].sort((a, b) => {
    if (a === "Unknown") return 1;
    if (b === "Unknown") return -1;
    return b.localeCompare(a);
  });
  const result = new Map<string, VM[]>();
  for (const k of sortedKeys) {
    const arr = map.get(k)!;
    arr.sort((a, b) => {
      const da = parseISODate(a.last_access ?? a.created_at)?.getTime() ?? 0;
      const db = parseISODate(b.last_access ?? b.created_at)?.getTime() ?? 0;
      return db - da;
    });
    result.set(k, arr);
  }
  return result;
}

export interface VMListProps {
  data: VMsResponse;
  groupBy?: "last_access" | "created_at";
  onRefresh: () => void;
  onOpenConsole?: (vm: VM) => void;
  onOpenCreateModal?: () => void;
  onOpenCloneModal?: (vm: VM) => void;
}

export function renderVMList(
  container: HTMLElement,
  props: VMListProps
): void {
  container.innerHTML = "";
  const {
    data,
    groupBy = "last_access",
    onRefresh,
    onOpenConsole,
    onOpenCreateModal,
    onOpenCloneModal,
  } = props;

  const vmListEl = document.createElement("div");
  vmListEl.className = "vm-list";

  const headerRow = document.createElement("div");
  headerRow.className = "vm-list__header-row";
  if (onOpenCreateModal) {
    const createBtn = document.createElement("button");
    createBtn.type = "button";
    createBtn.className = "vm-list__btn vm-list__btn--create";
    createBtn.textContent = "Create VM";
    createBtn.addEventListener("click", onOpenCreateModal);
    headerRow.appendChild(createBtn);
  }
  vmListEl.appendChild(headerRow);

  if (data.vms.length === 0 && data.orphans.length === 0) {
    const emptyP = document.createElement("p");
    emptyP.className = "vm-list__empty";
    emptyP.textContent = "No VMs";
    vmListEl.appendChild(emptyP);
    container.appendChild(vmListEl);
    return;
  }

  if (data.vms.length > 0) {
    const grouped = groupVMs(data.vms, groupBy);
    for (const [key, vms] of grouped) {
      const header = document.createElement("h3");
      header.className = "vm-list__group-header";
      header.textContent = formatGroupHeader(key);
      vmListEl.appendChild(header);

      const ul = document.createElement("ul");
      ul.className = "vm-list__rows";
      for (const vm of vms) {
        const li = document.createElement("li");
        li.className = "vm-list__row";
        const hostStatus = data.hosts[vm.host_id] ?? "unknown";
        const displayName = vm.display_name ?? vm.libvirt_uuid ?? "VM";
        const relTime = getRelativeTimeLabel(vm);

        li.innerHTML = `
          <div class="vm-list__row-main">
            <span class="vm-list__name">${escapeHtml(displayName)}</span>
            <span class="vm-list__status vm-list__status--${escapeHtml(vm.status.toLowerCase())}" title="Host: ${escapeHtml(hostStatus)}">${escapeHtml(vm.status)}</span>
            ${relTime ? `<span class="vm-list__rel-time">${escapeHtml(relTime)}</span>` : ""}
          </div>
          <div class="vm-list__row-actions">
            ${onOpenCloneModal ? `<button type="button" class="vm-list__btn vm-list__btn--clone" title="Clone VM">Clone</button>` : ""}
            <button type="button" class="vm-list__btn vm-list__btn--console" data-host="${escapeAttr(vm.host_id)}" data-uuid="${escapeAttr(vm.libvirt_uuid)}" title="Open console">Console</button>
          </div>
        `;

        const cloneBtn = li.querySelector(".vm-list__btn--clone");
        if (cloneBtn && onOpenCloneModal) {
          cloneBtn.addEventListener("click", () => onOpenCloneModal(vm));
        }
        const consoleBtn = li.querySelector(".vm-list__btn--console");
        if (consoleBtn && onOpenConsole) {
          consoleBtn.addEventListener("click", () => {
            onOpenConsole(vm);
          });
        } else if (consoleBtn) {
          (consoleBtn as HTMLButtonElement).disabled = true;
          (consoleBtn as HTMLButtonElement).title = "Console unavailable";
        }
        ul.appendChild(li);
      }
      vmListEl.appendChild(ul);
    }
  }

  container.appendChild(vmListEl);

  if (data.orphans.length > 0) {
    const orphansSection = document.createElement("section");
    orphansSection.className = "vm-list-orphans";
    const orphansHeader = document.createElement("div");
    orphansHeader.className = "vm-list-orphans__header";
    orphansHeader.innerHTML = `
      <button type="button" class="vm-list-orphans__toggle" aria-expanded="true">
        Orphan VMs (${data.orphans.length})
      </button>
    `;
    const orphansBody = document.createElement("div");
    orphansBody.className = "vm-list-orphans__body";
    const orphansList = document.createElement("ul");
    orphansList.className = "vm-list-orphans__list";

    let expanded = true;
    orphansHeader.querySelector(".vm-list-orphans__toggle")?.addEventListener("click", () => {
      expanded = !expanded;
      orphansBody.classList.toggle("vm-list-orphans__body--collapsed", !expanded);
      (orphansHeader.querySelector(".vm-list-orphans__toggle") as HTMLButtonElement).setAttribute("aria-expanded", String(expanded));
    });

    for (const orphan of data.orphans) {
      const li = document.createElement("li");
      li.className = "vm-list-orphans__item";
      li.innerHTML = `
        <span class="vm-list-orphans__name">${escapeHtml(orphan.name)}</span>
        <span class="vm-list-orphans__host">${escapeHtml(orphan.host_id)}</span>
        <button type="button" class="vm-list__btn vm-list__btn--claim" data-host="${escapeAttr(orphan.host_id)}" data-uuid="${escapeAttr(orphan.libvirt_uuid)}" data-name="${escapeAttr(orphan.name)}">Claim</button>
      `;
      const claimBtn = li.querySelector(".vm-list__btn--claim");
      if (claimBtn) {
        claimBtn.addEventListener("click", () => {
          handleClaim(orphan, claimBtn as HTMLButtonElement, onRefresh);
        });
      }
      orphansList.appendChild(li);
    }
    orphansBody.appendChild(orphansList);
    orphansSection.appendChild(orphansHeader);
    orphansSection.appendChild(orphansBody);
    container.appendChild(orphansSection);
  }
}

function escapeHtml(s: string): string {
  const div = document.createElement("div");
  div.textContent = s;
  return div.innerHTML;
}

function escapeAttr(s: string): string {
  return escapeHtml(s).replace(/"/g, "&quot;");
}

async function handleClaim(
  orphan: Orphan,
  btn: HTMLButtonElement,
  onRefresh: () => void
): Promise<void> {
  btn.disabled = true;
  try {
    await claimVM(orphan.host_id, orphan.libvirt_uuid, orphan.name);
    onRefresh();
  } catch (err) {
    btn.disabled = false;
    const msg = err instanceof ApiError ? err.message : "Claim failed";
    addAlert("api_error", msg, err instanceof ApiError ? String(err.status) : undefined);
  }
}
