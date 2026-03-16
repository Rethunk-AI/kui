# Winbox UX Research

Research for KUI VM list/console interface. **Decision: Winbox.js standardized for Canvas.** Decision log: [prd/decision-log.md](../prd/decision-log.md). PRD: [prd.md](../prd.md).

---

## Summary

Two relevant "Winbox" concepts:

1. **MikroTik Winbox** — Desktop GUI for RouterOS management. MDI (Multiple Document Interface): child windows in a work area; multi-session support for multiple devices.
2. **WinBox.js** — HTML5 window manager for the web ([Reddit](https://www.reddit.com/r/programming/comments/xm0h9q/winbox_is_a_modern_html5_window_manager_for_the/)). May support draggable windows in browser.

---

## MikroTik Winbox

- **Interface**: Toolbar at top, menu bar on left, work area for child windows
- **MDI**: Child windows attached to main window; shown in work area
- **Multi-session**: Configure multiple devices simultaneously; "Open In New Window" opens new window per device
- **UX**: Dense, functional; parallels CLI; power-user oriented
- **Sources**: [MikroTik Winbox](https://mikrotik.com/winbox), [Winbox Documentation](https://help.mikrotik.com/docs/spaces/ROS/pages/328129/WinBox)

---

## Implications for KUI

- **Canvas-based VM windows**: MDI/work-area pattern fits "all of my machines" — each VM console or detail in a child window within a main canvas
- **Draggable**: MikroTik Winbox child windows stay in work area; for browser, WinBox.js or similar may enable true draggable/floating windows
- **Multi-VM interaction**: Multi-session pattern — user can have multiple VM consoles open in same view

---

## Related Research

- [xyflow-canvas-ui-research.md](xyflow-canvas-ui-research.md) — Deep research on xyflow (React Flow), Winbox.js, Golden Layout, FlexLayout-react; architectural recommendation for VM console interface (topology vs windowing).

---

## Raw Results

Full JSON: [winbox-ux-research.json](winbox-ux-research.json)
