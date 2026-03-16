/**
 * Console launcher: noVNC/xterm window orchestration.
 * Spec §2: Winbox.js canvas, noVNC primary, xterm.js fallback.
 */
import RFB from "@novnc/novnc";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { openWinBox } from "./winbox-adapter";
import { addAlert } from "./alerts";
import type { VM } from "./api";

export type ConsoleType = "vnc" | "serial";

function getWebSocketUrl(path: string): string {
  const url = new URL(path, window.location.href);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}

function buildVncUrl(hostId: string, libvirtUuid: string): string {
  return getWebSocketUrl(
    `/api/hosts/${encodeURIComponent(hostId)}/vms/${encodeURIComponent(libvirtUuid)}/vnc`
  );
}

function buildSerialUrl(hostId: string, libvirtUuid: string): string {
  return getWebSocketUrl(
    `/api/hosts/${encodeURIComponent(hostId)}/vms/${encodeURIComponent(libvirtUuid)}/serial`
  );
}

function resolvePreferredType(consolePreference: string | null): ConsoleType | null {
  if (!consolePreference) return null;
  const p = consolePreference.toLowerCase();
  if (p === "vnc" || p === "novnc") return "vnc";
  if (p === "serial" || p === "xterm") return "serial";
  return null;
}

function createVncContainer(
  hostId: string,
  libvirtUuid: string,
  onFailure: () => void
): { container: HTMLElement; cleanup: () => void } {
  const container = document.createElement("div");
  container.className = "console-container console-container--vnc";
  container.style.cssText =
    "width:100%;height:100%;min-height:200px;background:rgb(40,40,40);";

  const url = buildVncUrl(hostId, libvirtUuid);
  let rfb: { disconnect: () => void; addEventListener: (n: string, fn: () => void) => void; scaleViewport: boolean } | null = null;

  try {
    rfb = new RFB(container, url);
  } catch (err) {
    onFailure();
    return { container, cleanup: () => {} };
  }

  rfb.addEventListener("securityfailure", () => {
    onFailure();
  });
  rfb.scaleViewport = true;

  const cleanup = (): void => {
    if (rfb) {
      try {
        rfb.disconnect();
      } catch {
        /* ignore */
      }
      rfb = null;
    }
  };

  return { container, cleanup };
}

function createSerialContainer(
  hostId: string,
  libvirtUuid: string,
  onFailure: () => void
): { container: HTMLElement; cleanup: () => void } {
  const container = document.createElement("div");
  container.className = "console-container console-container--serial";
  container.style.cssText =
    "width:100%;height:100%;min-height:200px;padding:8px;box-sizing:border-box;";

  const term = new Terminal({ cursorBlink: true });
  const fitAddon = new FitAddon();
  term.loadAddon(fitAddon);
  term.open(container);
  fitAddon.fit();

  const url = buildSerialUrl(hostId, libvirtUuid);
  const ws = new WebSocket(url);

  ws.binaryType = "arraybuffer";

  ws.onopen = (): void => {
    term.focus();
  };

  ws.onmessage = (ev: MessageEvent): void => {
    if (typeof ev.data === "string") {
      term.write(ev.data);
    } else if (ev.data instanceof ArrayBuffer) {
      term.write(new Uint8Array(ev.data));
    }
  };

  ws.onerror = (): void => {
    onFailure();
  };

  ws.onclose = (): void => {
    term.write("\r\n[Connection closed]\r\n");
  };

  term.onData((data: string) => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(data);
    }
  });

  const resizeObserver = new ResizeObserver(() => {
    fitAddon.fit();
  });
  resizeObserver.observe(container);

  const cleanup = (): void => {
    resizeObserver.disconnect();
    ws.close();
    term.dispose();
  };

  return { container, cleanup };
}

export interface OpenConsoleOptions {
  hostId: string;
  libvirtUuid: string;
  displayName?: string | null;
  consolePreference?: string | null;
}

/**
 * Opens a Winbox.js window with noVNC or xterm.js console.
 * Console selection: use VM console_preference if set; else try VNC first, fall back to serial on failure.
 */
export function openConsole(opts: OpenConsoleOptions): void {
  const { hostId, libvirtUuid, displayName, consolePreference } = opts;
  const title = displayName?.trim() || "Console";

  const preferred = resolvePreferredType(consolePreference ?? null);

  const trySerial = (): void => {
    const onSerialFailure = (): void => {
      addAlert(
        "console_failure",
        "Serial console failed",
        `VM ${displayName || libvirtUuid}`
      );
    };

    const { container, cleanup } = createSerialContainer(
      hostId,
      libvirtUuid,
      onSerialFailure
    );

    openWinBox(title, container, {
      onclose: () => {
        cleanup();
        return false;
      },
    });
  };

  const tryVnc = (): void => {
    const onVncFailure = (): void => {
      addAlert(
        "console_failure",
        "VNC console failed, trying serial",
        `VM ${displayName || libvirtUuid}`
      );
      trySerial();
    };

    const { container, cleanup } = createVncContainer(
      hostId,
      libvirtUuid,
      onVncFailure
    );

    openWinBox(title, container, {
      onclose: () => {
        cleanup();
        return false;
      },
    });
  };

  if (preferred === "serial") {
    trySerial();
  } else {
    tryVnc();
  }
}

/**
 * Opens console for a VM. Convenience wrapper around openConsole.
 */
export function openConsoleForVM(vm: VM): void {
  openConsole({
    hostId: vm.host_id,
    libvirtUuid: vm.libvirt_uuid,
    displayName: vm.display_name,
    consolePreference: vm.console_preference,
  });
}
