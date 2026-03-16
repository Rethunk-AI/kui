/**
 * Winbox.js integration adapter.
 * Mounts DOM nodes for console windows.
 */
// @ts-expect-error winbox has no types
import WinBox from "winbox/src/js/winbox.js";
import "winbox/dist/css/winbox.min.css";

export function openWinBox(title: string, mountNode: HTMLElement): unknown {
  return new WinBox(title, { mount: mountNode });
}
