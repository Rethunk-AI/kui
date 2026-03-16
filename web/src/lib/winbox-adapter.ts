/**
 * Winbox.js integration adapter.
 * Mounts DOM nodes for console windows.
 */
// @ts-expect-error winbox has no types
import WinBox from "winbox/src/js/winbox.js";
import "winbox/dist/css/winbox.min.css";

export interface WinBoxOptions {
  onclose?: (force?: boolean) => boolean | void;
}

export function openWinBox(
  title: string,
  mountNode: HTMLElement,
  options?: WinBoxOptions
): unknown {
  const opts = { mount: mountNode, ...options };
  return new WinBox(title, opts);
}
