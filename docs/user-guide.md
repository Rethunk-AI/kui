# KUI User Guide

How to use the KUI web interface for VM management. For product overview, see [PRD](prd.md).

---

## Login

After setup completes, open KUI in a browser. Enter the admin username and password configured during setup. Sessions use JWT cookies; logout clears the session. See [decision-log §2](prd/decision-log.md) (Auth, Session).

---

## VM List

The main view shows:

- **Claimed VMs** — grouped by month/year (last access or created date). Each VM shows display name, status (running, shut off, paused, etc.), and relative time.
- **Orphans** — libvirt domains not yet claimed by KUI. See [Claiming Orphans](#claiming-orphans).
- **Host status** — online/offline per host in the header.

---

## Host Selector

The header shows the current default host. Use the host selector dropdown to switch hosts. Changes persist to your preferences. The selected host affects VM create, clone, and console operations. See [decision-log §2](prd/decision-log.md) (Default host).

---

## Create VM

Two ways to create a VM:

### Pool + path

1. Choose host and storage pool.
2. Choose disk:
   - **Existing volume** — pick from the pool’s volumes.
   - **New disk** — specify size; KUI creates a volume in the pool.
3. Optionally override CPU, RAM, network (from config defaults).
4. Submit. KUI defines the domain and inserts metadata; VM is created but not started.

### Clone

1. Select a source VM (must be stopped).
2. Choose target host and pool.
3. Optionally set a name (default: `{source}`).
4. Submit. KUI copies the disk and defines the new domain.

See [spec-vm-lifecycle-create](../specs/done/spec-vm-lifecycle-create/spec.md) §2–3 for create and clone flows.

---

## Lifecycle Actions

From the VM list or detail view:

| Action | Effect |
|--------|--------|
| **Start** | Start the VM |
| **Stop** | Graceful shutdown first (configurable timeout); force stop if needed |
| **Pause** | Suspend the VM |
| **Resume** | Resume from pause |
| **Destroy** | Immediate hard stop and remove runtime |

---

## Console

Click **Console** to open a VM console in a window:

- **VNC (noVNC)** — graphical console; default when available.
- **Serial (xterm.js)** — text console; used if VNC fails or you set `console_preference` to serial.

Console preference is per-VM; set it via the VM config edit. See [decision-log §2](prd/decision-log.md) (Console protocol).

---

## Templates

1. **Save VM as template** — From a stopped VM, save it as a template (domain XML + disk copy). Stored in Git.
2. **List templates** — View saved templates in the templates section.
3. **Create VM from template** — (v2) Not yet in MVP.

See [spec-template-management](../specs/done/spec-template-management/spec.md).

---

## Claiming Orphans

Orphans are libvirt domains not in KUI’s metadata. To claim one:

1. Find the orphan in the **Orphans** section.
2. Click **Claim**. KUI adds metadata for that domain; it appears in the main VM list.

---

## First-Run Checklist

When the VM list is empty and you haven’t dismissed it, a checklist appears:

- Create VM from pool or disk path
- Clone an existing VM

Click **Dismiss** to hide it. Dismissal is stored in preferences. See [decision-log §2](prd/decision-log.md) (Empty state, First-run).

---

## Alerts

Transient alerts (host offline, errors) appear in the alerts panel. They clear on refresh. See [decision-log §2](prd/decision-log.md) (Notifications).

---

## VM Config Edit

Edit display name, console preference (novnc, xterm), and domain settings (CPU, RAM, network) via the VM detail or edit flow. Domain edits require the VM to be stopped. See [spec-vm-lifecycle-create](../specs/done/spec-vm-lifecycle-create/spec.md) §7.
