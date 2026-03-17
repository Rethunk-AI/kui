/**
 * VM list UI with month/year grouping.
 * Spec §6: flat list, group by last_access (or created_at fallback), display_name, status, relative-time.
 * Orphans in separate section with Claim.
 */
import { ApiError, claimVM, recoverVM, type VM, type Orphan, type VMsResponse } from "../lib/api";
import { addAlert } from "../lib/alerts";
import { showToast } from "../lib/toast";

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

function isStuck(vm: VM): boolean {
  return vm.status === "crashed" || vm.status === "blocked";
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
  /** Called when selection changes (arrow keys or click). Parent stores selection for shortcuts. */
  onRowSelect?: (vm: VM, index: number) => void;
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
    onRowSelect,
  } = props;

  const vmListEl = document.createElement("div");
  vmListEl.className = "vm-list";

  const pageTitle = document.createElement("h1");
  pageTitle.className = "vm-list__title";
  pageTitle.textContent = "Virtual machines";
  vmListEl.appendChild(pageTitle);

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
    const flatVMs: VM[] = [];
    for (const [, vms] of grouped) {
      flatVMs.push(...vms);
    }

    const listboxContainer = document.createElement("div");
    listboxContainer.className = "vm-list__listbox";
    listboxContainer.setAttribute("role", "listbox");
    listboxContainer.setAttribute("aria-label", "Virtual machines");
    listboxContainer.tabIndex = -1;

    let selectedIndex = 0;
    const optionElements: HTMLElement[] = [];

    function setSelection(index: number): void {
      const newIndex = Math.max(0, Math.min(index, flatVMs.length - 1));
      if (newIndex === selectedIndex) return;
      selectedIndex = newIndex;
      for (let i = 0; i < optionElements.length; i++) {
        const el = optionElements[i];
        const selected = i === selectedIndex;
        el.setAttribute("aria-selected", String(selected));
        el.setAttribute("tabindex", selected ? "0" : "-1");
      }
      onRowSelect?.(flatVMs[selectedIndex], selectedIndex);
    }

    listboxContainer.addEventListener("keydown", (ev) => {
      if (!listboxContainer.contains(document.activeElement)) return;
      if (ev.key === "ArrowDown") {
        ev.preventDefault();
        setSelection(selectedIndex + 1);
        optionElements[selectedIndex]?.focus();
      } else if (ev.key === "ArrowUp") {
        ev.preventDefault();
        setSelection(selectedIndex - 1);
        optionElements[selectedIndex]?.focus();
      }
    });

    for (const [key, vms] of grouped) {
      const header = document.createElement("h2");
      header.className = "vm-list__group-header";
      header.textContent = formatGroupHeader(key);
      listboxContainer.appendChild(header);

      const ul = document.createElement("ul");
      ul.className = "vm-list__rows";
      for (let i = 0; i < vms.length; i++) {
        const vm = vms[i];
        const flatIndex = flatVMs.indexOf(vm);
        const li = document.createElement("li");
        li.className = "vm-list__row";
        li.setAttribute("role", "option");
        li.setAttribute("aria-selected", String(flatIndex === selectedIndex));
        li.tabIndex = flatIndex === selectedIndex ? 0 : -1;
        optionElements.push(li);

        const hostStatus = data.hosts[vm.host_id] ?? "unknown";
        const displayName = vm.display_name ?? vm.libvirt_uuid ?? "VM";
        const relTime = getRelativeTimeLabel(vm);

        const stuck = isStuck(vm);
        li.innerHTML = `
          <div class="vm-list__row-main">
            <span class="vm-list__name">${escapeHtml(displayName)}</span>
            <span class="vm-list__status vm-list__status--${escapeHtml(vm.status.toLowerCase())}" title="Host: ${escapeHtml(hostStatus)}">${escapeHtml(vm.status)}</span>
            ${stuck ? `<span class="vm-list__badge vm-list__badge--stuck">Stuck</span>` : ""}
            ${relTime ? `<span class="vm-list__rel-time">${escapeHtml(relTime)}</span>` : ""}
          </div>
          <div class="vm-list__row-actions">
            ${stuck ? `<button type="button" class="vm-list__btn vm-list__btn--recover" title="Recover stuck VM">Recover</button>` : ""}
            ${onOpenCloneModal ? `<button type="button" class="vm-list__btn vm-list__btn--clone" title="Clone VM">Clone</button>` : ""}
            <button type="button" class="vm-list__btn vm-list__btn--console" data-host="${escapeAttr(vm.host_id)}" data-uuid="${escapeAttr(vm.libvirt_uuid)}" title="Open console">Console</button>
          </div>
        `;

        li.addEventListener("click", (ev) => {
          if ((ev.target as HTMLElement).closest("button")) return;
          setSelection(flatIndex);
          li.focus();
        });

        const recoverBtn = li.querySelector(".vm-list__btn--recover");
        if (recoverBtn && stuck) {
          recoverBtn.addEventListener("click", () => {
            handleRecover(vm, recoverBtn as HTMLButtonElement, onRefresh);
          });
        }
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
      listboxContainer.appendChild(ul);
    }

    vmListEl.appendChild(listboxContainer);

    if (onRowSelect && flatVMs.length > 0) {
      onRowSelect(flatVMs[0], 0);
    }
  }

  container.appendChild(vmListEl);

  if (data.orphans.length > 0) {
    const orphansSection = document.createElement("section");
    orphansSection.className = "vm-list-orphans";
    orphansSection.setAttribute("aria-labelledby", "orphans-heading");
    const orphansHeader = document.createElement("div");
    orphansHeader.className = "vm-list-orphans__header";
    const orphansTitle = document.createElement("h2");
    orphansTitle.id = "orphans-heading";
    orphansTitle.className = "vm-list-orphans__title";
    orphansTitle.innerHTML = `
      <button type="button" class="vm-list-orphans__toggle" aria-expanded="true">
        Orphan VMs (${data.orphans.length})
      </button>
    `;
    orphansHeader.appendChild(orphansTitle);
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

async function handleRecover(
  vm: VM,
  btn: HTMLButtonElement,
  onRefresh: () => void
): Promise<void> {
  btn.disabled = true;
  try {
    await recoverVM(vm.host_id, vm.libvirt_uuid);
    onRefresh();
  } catch (err) {
    btn.disabled = false;
    const msg = err instanceof ApiError ? err.message : "Recover failed";
    showToast(msg, "warn");
    addAlert("api_error", msg, err instanceof ApiError ? String(err.status) : undefined);
  }
}
