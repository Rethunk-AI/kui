/**
 * Console launcher: noVNC/xterm window orchestration.
 * Placeholder for spec-console-realtime integration.
 */
export function createConsoleContainer(): HTMLElement {
  const div = document.createElement("div");
  div.className = "console-container";
  div.style.cssText = "width:100%;height:100%;min-height:200px;";
  return div;
}
