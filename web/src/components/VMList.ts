/**
 * VM list UI with month/year grouping.
 * Spec §6: flat list, group by last_access (or created_at fallback), display_name, status, relative-time.
 * Orphans in separate section with Claim. Bulk claim/destroy with checkboxes.
 */
import {
  ApiError,
  bulkClaimOrphans,
  bulkDestroyOrphans,
  claimVM,
  recoverVM,
  type Host,
  type Orphan,
  type VM,
  type VMsResponse,
} from "../lib/api";
import { addAlert } from "../lib/alerts";
import { showToast } from "../lib/toast";

function orphanKey(o: Orphan): string {
  return `${o.host_id}:${o.libvirt_uuid}`;
}

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
  hosts: Host[];
  groupBy?: "last_access" | "created_at";
  onRefresh: () => void;
  onOpenConsole?: (vm: VM) => void;
  onOpenCreateModal?: () => void;
  onOpenCloneModal?: (vm: VM) => void;
  onOpenDomainXMLEditor?: (vm: VM) => void;
  /** Called when selection changes (arrow keys or click). Parent stores selection for shortcuts. */
  onRowSelect?: (vm: VM, index: number) => void;
}

export function renderVMList(
  container: HTMLElement,
  props: VMListProps
): void {
  container.innerHTML = "";
  const { data, hosts, groupBy = "last_access",
    onRefresh,
    onOpenConsole,
    onOpenCreateModal,
    onOpenCloneModal,
    onOpenDomainXMLEditor,
    onRowSelect,
  } = props;

  const safeData = {
    ...data,
    vms: data.vms ?? [],
    orphans: data.orphans ?? [],
  };

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
    createBtn.disabled = hosts.length === 0;
    if (hosts.length === 0) {
      createBtn.title = "Add hosts in setup first";
    }
    createBtn.addEventListener("click", onOpenCreateModal);
    headerRow.appendChild(createBtn);
  }
  vmListEl.appendChild(headerRow);

  if (safeData.vms.length === 0 && safeData.orphans.length === 0) {
    const emptyP = document.createElement("p");
    emptyP.className = "vm-list__empty";
    emptyP.textContent = "No VMs";
    vmListEl.appendChild(emptyP);
    container.appendChild(vmListEl);
    return;
  }

  if (safeData.vms.length > 0) {
    const grouped = groupVMs(safeData.vms, groupBy);
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

        const hostStatus = safeData.hosts[vm.host_id] ?? "unknown";
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
            ${onOpenDomainXMLEditor ? `<button type="button" class="vm-list__btn vm-list__btn--edit-xml" title="Edit domain XML">Edit XML</button>` : ""}
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
        const editXmlBtn = li.querySelector(".vm-list__btn--edit-xml");
        if (editXmlBtn && onOpenDomainXMLEditor) {
          editXmlBtn.addEventListener("click", () => onOpenDomainXMLEditor(vm));
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

  if (safeData.orphans.length > 0) {
    const selectedOrphanIds = new Set<string>();
    const orphansSection = document.createElement("section");
    orphansSection.className = "vm-list-orphans";
    orphansSection.setAttribute("aria-labelledby", "orphans-heading");
    const orphansHeader = document.createElement("div");
    orphansHeader.className = "vm-list-orphans__header";
    const orphansTitle = document.createElement("h2");
    orphansTitle.id = "orphans-heading";
    orphansTitle.className = "vm-list-orphans__title";
    const selectAllCheckbox = document.createElement("input");
    selectAllCheckbox.type = "checkbox";
    selectAllCheckbox.className = "vm-list-orphans__select-all";
    selectAllCheckbox.setAttribute("aria-label", "Select all orphan VMs");
    const toggleBtn = document.createElement("button");
    toggleBtn.type = "button";
    toggleBtn.className = "vm-list-orphans__toggle";
    toggleBtn.setAttribute("aria-expanded", "true");
    toggleBtn.textContent = `Orphan VMs (${safeData.orphans.length})`;
    orphansTitle.appendChild(selectAllCheckbox);
    orphansTitle.appendChild(toggleBtn);
    orphansHeader.appendChild(orphansTitle);

    const orphansBody = document.createElement("div");
    orphansBody.className = "vm-list-orphans__body";
    const bulkBar = document.createElement("div");
    bulkBar.className = "vm-list-orphans__bulk-bar";
    bulkBar.hidden = true;
    const bulkClaimBtn = document.createElement("button");
    bulkClaimBtn.type = "button";
    bulkClaimBtn.className = "vm-list__btn vm-list__btn--bulk-claim";
    const bulkDestroyBtn = document.createElement("button");
    bulkDestroyBtn.type = "button";
    bulkDestroyBtn.className = "vm-list__btn vm-list__btn--bulk-destroy";
    bulkBar.appendChild(bulkClaimBtn);
    bulkBar.appendChild(bulkDestroyBtn);

    const orphansList = document.createElement("ul");
    orphansList.className = "vm-list-orphans__list";

    const updateBulkBar = (): void => {
      const n = selectedOrphanIds.size;
      bulkBar.hidden = n === 0;
      bulkClaimBtn.textContent = `Claim selected (${n})`;
      bulkDestroyBtn.textContent = `Destroy selected (${n})`;
      selectAllCheckbox.checked = n > 0 && n === safeData.orphans.length;
      selectAllCheckbox.indeterminate = n > 0 && n < safeData.orphans.length;
    };

    selectAllCheckbox.addEventListener("change", () => {
      if (selectAllCheckbox.checked) {
        for (const o of safeData.orphans) selectedOrphanIds.add(orphanKey(o));
      } else {
        selectedOrphanIds.clear();
      }
      orphansList.querySelectorAll<HTMLInputElement>(".vm-list-orphans__checkbox").forEach((cb) => {
        cb.checked = selectAllCheckbox.checked;
      });
      updateBulkBar();
    });

    let expanded = true;
    toggleBtn.addEventListener("click", () => {
      expanded = !expanded;
      orphansBody.classList.toggle("vm-list-orphans__body--collapsed", !expanded);
      toggleBtn.setAttribute("aria-expanded", String(expanded));
    });

    for (const orphan of safeData.orphans) {
      const key = orphanKey(orphan);
      const li = document.createElement("li");
      li.className = "vm-list-orphans__item";
      const checkbox = document.createElement("input");
      checkbox.type = "checkbox";
      checkbox.className = "vm-list-orphans__checkbox";
      checkbox.dataset.key = key;
      checkbox.setAttribute("aria-label", `Select ${escapeAttr(orphan.name)}`);
      checkbox.addEventListener("change", () => {
        if (checkbox.checked) selectedOrphanIds.add(key);
        else selectedOrphanIds.delete(key);
        updateBulkBar();
      });
      li.appendChild(checkbox);
      const nameSpan = document.createElement("span");
      nameSpan.className = "vm-list-orphans__name";
      nameSpan.textContent = orphan.name;
      li.appendChild(nameSpan);
      const hostSpan = document.createElement("span");
      hostSpan.className = "vm-list-orphans__host";
      hostSpan.textContent = orphan.host_id;
      li.appendChild(hostSpan);
      const claimBtn = document.createElement("button");
      claimBtn.type = "button";
      claimBtn.className = "vm-list__btn vm-list__btn--claim";
      claimBtn.dataset.host = orphan.host_id;
      claimBtn.dataset.uuid = orphan.libvirt_uuid;
      claimBtn.dataset.name = orphan.name;
      claimBtn.textContent = "Claim";
      claimBtn.addEventListener("click", () => {
        handleClaim(orphan, claimBtn, onRefresh);
      });
      li.appendChild(claimBtn);
      orphansList.appendChild(li);
    }

    bulkClaimBtn.addEventListener("click", () => {
      handleBulkClaim(safeData.orphans, selectedOrphanIds, bulkClaimBtn, bulkDestroyBtn, onRefresh, updateBulkBar);
    });
    bulkDestroyBtn.addEventListener("click", () => {
      handleBulkDestroy(safeData.orphans, selectedOrphanIds, bulkClaimBtn, bulkDestroyBtn, onRefresh, updateBulkBar);
    });

    orphansBody.appendChild(bulkBar);
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
    if (err instanceof ApiError && err.status === 401) return;
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
    if (err instanceof ApiError && err.status === 401) return;
    const msg = err instanceof ApiError ? err.message : "Recover failed";
    showToast(msg, "warn");
    addAlert("api_error", msg, err instanceof ApiError ? String(err.status) : undefined);
  }
}

async function handleBulkClaim(
  orphans: Orphan[],
  selectedOrphanIds: Set<string>,
  claimBtn: HTMLButtonElement,
  destroyBtn: HTMLButtonElement,
  onRefresh: () => void,
  updateBulkBar: () => void
): Promise<void> {
  const items = orphans
    .filter((o) => selectedOrphanIds.has(orphanKey(o)))
    .map((o) => ({ host_id: o.host_id, libvirt_uuid: o.libvirt_uuid, display_name: o.name }));
  if (items.length === 0) return;
  claimBtn.disabled = true;
  destroyBtn.disabled = true;
  try {
    const resp = await bulkClaimOrphans(items);
    const claimedN = resp.claimed.length;
    const conflictN = resp.conflicts.length;
    if (conflictN === 0) {
      showToast(`Claimed ${claimedN} orphan(s)`, "success");
    } else if (claimedN === 0) {
      const first = resp.conflicts[0];
      showToast(`${first.reason}: ${first.host_id}/${first.libvirt_uuid}`, "warn");
      if (conflictN > 1) {
        addAlert("bulk_claim", `${conflictN} failed: ${resp.conflicts.map((c) => c.reason).join(", ")}`);
      }
    } else {
      showToast(`Claimed ${claimedN}, ${conflictN} failed`, "warn");
      addAlert("bulk_claim", resp.conflicts.map((c) => `${c.host_id}/${c.libvirt_uuid}: ${c.reason}`).join("; "));
    }
    selectedOrphanIds.clear();
    updateBulkBar();
    onRefresh();
  } catch (err) {
    claimBtn.disabled = false;
    destroyBtn.disabled = false;
    if (err instanceof ApiError && err.status === 401) return;
    const msg = err instanceof ApiError ? err.message : "Request failed";
    showToast(msg, "warn");
    addAlert("api_error", msg, err instanceof ApiError ? String(err.status) : undefined);
  }
}

async function handleBulkDestroy(
  orphans: Orphan[],
  selectedOrphanIds: Set<string>,
  claimBtn: HTMLButtonElement,
  destroyBtn: HTMLButtonElement,
  onRefresh: () => void,
  updateBulkBar: () => void
): Promise<void> {
  const items = orphans
    .filter((o) => selectedOrphanIds.has(orphanKey(o)))
    .map((o) => ({ host_id: o.host_id, libvirt_uuid: o.libvirt_uuid }));
  if (items.length === 0) return;
  if (!confirm(`Destroy ${items.length} orphan(s)? This cannot be undone.`)) return;
  claimBtn.disabled = true;
  destroyBtn.disabled = true;
  try {
    const resp = await bulkDestroyOrphans(items);
    const destroyedN = resp.destroyed.length;
    const failedN = resp.failed.length;
    if (failedN === 0) {
      showToast(`Destroyed ${destroyedN} orphan(s)`, "success");
    } else if (destroyedN === 0) {
      const first = resp.failed[0];
      showToast(`${first.reason}: ${first.host_id}/${first.libvirt_uuid}`, "warn");
      if (failedN > 1) {
        addAlert("bulk_destroy", `${failedN} failed: ${resp.failed.map((f) => f.reason).join(", ")}`);
      }
    } else {
      showToast(`Destroyed ${destroyedN}, ${failedN} failed`, "warn");
      addAlert("bulk_destroy", resp.failed.map((f) => `${f.host_id}/${f.libvirt_uuid}: ${f.reason}`).join("; "));
    }
    selectedOrphanIds.clear();
    updateBulkBar();
    onRefresh();
  } catch (err) {
    claimBtn.disabled = false;
    destroyBtn.disabled = false;
    if (err instanceof ApiError && err.status === 401) return;
    const msg = err instanceof ApiError ? err.message : "Request failed";
    showToast(msg, "warn");
    addAlert("api_error", msg, err instanceof ApiError ? String(err.status) : undefined);
  }
}
