/**
 * Winbox.js integration adapter.
 * Mounts DOM nodes for console windows.
 * Tracks open instances for Escape-to-close (WinBox does not support Escape natively).
 */
// @ts-expect-error winbox has no types
import WinBox from "winbox/src/js/winbox.js";
import "winbox/dist/css/winbox.min.css";

export interface WinBoxOptions {
  onclose?: (force?: boolean) => boolean | void;
}

interface WinBoxInstance {
  close: () => void;
}

const openWinBoxes: WinBoxInstance[] = [];

export function openWinBox(
  title: string,
  mountNode: HTMLElement,
  options?: WinBoxOptions
): unknown {
  const originalOnclose = options?.onclose;
  let instance: WinBoxInstance;
  const opts = {
    mount: mountNode,
    ...options,
    onclose: (force?: boolean) => {
      const idx = openWinBoxes.indexOf(instance);
      if (idx >= 0) openWinBoxes.splice(idx, 1);
      return originalOnclose?.(force);
    },
  };
  instance = new WinBox(title, opts) as WinBoxInstance;
  openWinBoxes.push(instance);
  return instance;
}

/**
 * Closes the topmost WinBox instance, if any.
 * Returns true if a WinBox was closed, false otherwise.
 */
export function closeTopmostWinBox(): boolean {
  const topmost = openWinBoxes[openWinBoxes.length - 1];
  if (!topmost) return false;
  topmost.close();
  return true;
}
