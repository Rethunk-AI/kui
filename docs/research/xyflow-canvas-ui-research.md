# xyflow Canvas UI Research

**Decision: xyflow deferred for Canvas.** KUI uses Winbox.js for the Canvas (VM console windows). xyflow reserved for future use: network topology, hardware/infrastructure maps. See [decision-log.md](../prd/decision-log.md).

---

# Executive Summary

xyflow and Winbox.js serve fundamentally different and non-competing purposes. xyflow, which encompasses React Flow and Svelte Flow, is a specialized library for creating interactive, canvas-based, node-graph user interfaces. Its primary function is to render nodes and the edges that connect them, making it ideal for applications like workflow builders, data visualizers, and topology maps. It focuses on the diagramming canvas and its interactive elements. In contrast, Winbox.js is a lightweight, high-performance HTML5 window manager designed to create and manage floating, independent windows within a web application. It handles the creation, movement, resizing, and stacking of these windows, but does not concern itself with the content inside them. Therefore, they are not alternatives to each other; one is for building diagrammatic content (xyflow), and the other is for creating UI containers (Winbox.js).

# Architectural Recommendation

For building a multi-window VM console management interface, the recommended architectural approach is to combine several specialized libraries rather than choosing just one. The primary layout should be handled by a dedicated window or docking manager. Your choice depends on the desired user experience:

1.  **Primary Layout Manager**: 
    *   For a highly structured, IDE-like experience with docking, tabbing, splitting, and persistent layouts, choose a mature docking library like **Golden Layout** (framework-agnostic) or **FlexLayout-react** (React-focused).
    *   If the goal is a more free-form interface with minimal, lightweight floating windows that the user can arrange manually, **Winbox.js** is an excellent choice due to its performance and simplicity.

2.  **Console and Terminal Embedding**: 
    *   The windows or panes created by your layout manager will host the actual console content. For this, use specialized components:
    *   **VNC/RDP/SSH**: Use **noVNC** for direct web-based VNC access or **Apache Guacamole** as a clientless remote desktop gateway to handle VNC, RDP, and SSH sessions via HTML5.
    *   **Terminals**: Embed **xterm.js** to provide a fully-featured, high-performance terminal in the browser.

3.  **Infrastructure Visualization**: 
    *   Use **xyflow** (React Flow or Svelte Flow) within one of the panes or windows to display a topology or dependency graph of your infrastructure (e.g., VMs, hosts, networks, storage). This view provides a visual map of your environment. You can integrate auto-layout engines like Dagre or ELKjs for automatic arrangement of the nodes.

**Example Integration**: A practical architecture would use Golden Layout as the main application shell. This shell would contain several panes: a 'Topology' pane running an xyflow graph, and a 'Consoles' pane configured as a tabbed stack. When a user clicks on a VM node within the xyflow graph, it would trigger an API call to Golden Layout to open a new tab in the 'Consoles' pane, loading a noVNC or Guacamole session for that specific VM.

# Xyflow Overview

## Name

xyflow

## Description

xyflow is the umbrella brand and monorepo for React Flow and Svelte Flow. These are powerful open-source libraries designed for building interactive, node-based user interfaces such as graphs, flowcharts, and diagrams. They provide a performant canvas for rendering nodes and edges, focusing on the core visualization and interaction, while allowing developers to integrate their own data and layout algorithms. Svelte Flow was developed by the same team that created and maintains the popular React Flow library, ensuring a consistent philosophy and feature set across different frameworks.

## Supported Frameworks

React (via React Flow) and Svelte (via Svelte Flow)

## License

MIT Licensed

## Repository Url

https://github.com/xyflow/xyflow


# Xyflow Key Features

## Feature

Built-in Interactivity

## Description

Provides a rich set of interactions out-of-the-box, including panning and zooming the canvas, selecting single or multiple nodes, dragging nodes around the canvas, and creating or removing edges between nodes. This core functionality allows for the rapid development of dynamic and user-friendly graph interfaces.

## Feature

Customization and Extensibility

## Description

Offers extensive customization options. Developers can create custom node and edge types to fit specific application needs, apply custom styling and theming, and extend the library's functionality. This flexibility is a key aspect, emphasized in the design of both React Flow and Svelte Flow.

## Feature

Drag and Drop Integration

## Description

Supports dragging elements from outside the canvas and dropping them to create new nodes. This is demonstrated in examples using the native HTML Drag and Drop API, as well as libraries like react-draggable and Neodrag, facilitating intuitive workflow-building experiences.

## Feature

Advanced Graph Features

## Description

Includes support for more complex graph structures and user interactions, such as grouping nodes, creating subflows (nested graphs), collision detection, and implementing whiteboard-like features with tools like a lasso selection. These capabilities enable the creation of sophisticated visual editors and diagramming tools.

## Feature

Server-Side Rendering (SSR) and Performance

## Description

The library is optimized for performance, capable of handling large graphs. It also supports Server-Side Rendering (SSR), which is crucial for applications requiring fast initial load times and SEO benefits. Performance guidance is provided to help developers optimize their implementations.


# Xyflow Automatic Layouting

## Engine Name

Dagre

## Description

Dagre is a popular external JavaScript library used with xyflow for automatic graph layouting. It is not built into xyflow directly but is supported through integration examples and reusable hooks. It specializes in laying out directed graphs.

## Use Case

Dagre is best suited for creating clean, hierarchical, or layered 'tree-like' layouts from a set of nodes and edges. It is ideal for applications like workflow builders, dependency graphs, and organizational charts where the direction of flow is important and nodes need to be arranged automatically to minimize edge crossings and create a clear visual structure.


# Winbox Js Overview

## Name

Winbox.js

## Description

A modern, high-performance HTML5 window manager for the web. It allows for the creation of windows that can mount DOM elements or open URLs in iframes, providing a rich API for programmatic control over window behavior such as movement, resizing, and stacking.

## Key Attributes

Described as lightweight, having outstanding performance, no dependencies, and being themable. It offers a rich API for creating, moving, resizing, maximizing, minimizing, and focusing windows, as well as querying the window stack.

## License

MIT License

## Repository Url

https://github.com/nextapps-de/winbox


# Comparison Of Purposes

xyflow and Winbox.js are designed for entirely different tasks and are not interchangeable.

**xyflow (React Flow / Svelte Flow):**
*   **Primary Purpose**: To build interactive, node-based diagrams and editors. It provides a canvas where you can render nodes (blocks of information or UI) and edges (lines connecting the nodes).
*   **Core Functionality**: It manages the state and rendering of a graph structure. It offers out-of-the-box interactivity such as panning, zooming, selecting, and dragging nodes and connecting them with edges. It is highly customizable, allowing for custom node types and styles. While it doesn't have a built-in layout engine, it is designed to integrate with external layout algorithms like Dagre, ELKjs, and d3-hierarchy to automatically arrange nodes.
*   **Use Case**: Ideal for creating workflow editors, organizational charts, data processing pipelines, database schema visualizers, or any application where relationships between entities need to be displayed and manipulated graphically.

**Winbox.js:**
*   **Primary Purpose**: To act as a modern, lightweight window manager for web applications. It allows you to create floating, draggable, and resizable windows on top of your existing webpage.
*   **Core Functionality**: It provides a simple API to create and control windows (e.g., `new WinBox()`). It manages the window stack (z-index), focus, and state (minimized, maximized). Each window can host arbitrary HTML content, mount a DOM element, or load a URL in an iframe. It is dependency-free and focuses on performance and ease of use for windowing tasks.
*   **Use Case**: Perfect for multi-tasking interfaces, web-based "desktops," or any application where you need to spawn multiple independent, floating dialogs or content panes, similar to a traditional operating system's UI.

# Docking Layout Libraries

## Library Name

Golden Layout

## Description

A mature, framework-agnostic, multi-window and docking layout manager that allows for complex arrangements using rows, columns, and stacks.

## Key Features

Supports native popup windows, touch interactions, saving and loading layouts, comprehensive theming, and responsive design. It has a substantial API and offers integration with frameworks like Angular and Vue. Version 2 was ported to TypeScript.

## Use Case

Ideal for stable, IDE-like multi-pane applications and complex, persistent workspaces that require features like docking, tabbing, and popouts.

## Library Name

FlexLayout-react

## Description

A layout manager specifically for React that arranges components into multiple tabsets. It is noted for being dependency-light, requiring only React.

## Key Features

Features include docking, tabsets, popout tabs into new browser windows, splitters, tab dragging, theming, and mobile support. It can serialize and restore layouts from JSON, and component state is preserved when tabs are moved.

## Use Case

Best suited for React-focused projects that need a stable, IDE-like multi-pane application with docking and tabbing capabilities.

## Library Name

DockSpawn TS

## Description

A TypeScript-based docking framework designed to provide a Visual Studio-like docking experience for web applications.

## Key Features

Provides an IDE-like docking framework. Key features, inferred from its description, include docking panels, tabbing, splitting, and managing complex window layouts similar to professional development environments.

## Use Case

Used for creating web applications that require a user interface with IDE-like docking capabilities, mimicking the layout management of desktop applications like Visual Studio.


# Grid Layout Libraries

## Library Name

Gridstack.js

## Description

A modern TypeScript library that enables developers to create draggable, resizable, and responsive layouts, particularly for dashboards.

## Key Features

Core features include creating draggable and resizable widgets, building responsive layouts, and supporting inter-grid interactivity, which allows dragging items between multiple grids.

## Layout Style

Dashboard-style layouts with draggable and resizable tiles.

## Library Name

Muuri

## Description

A high-performance library for creating responsive, sortable, filterable, and draggable layouts. It is designed to be fast and highly customizable.

## Key Features

Features include a customizable layout system that can create almost any imaginable layout, built-in drag-and-drop support for sorting items (even between grids), and high performance through batched DOM operations and use of web workers.

## Layout Style

Creates versatile layouts, including masonry and dashboard styles. It is described as supporting 'infinite layouts' through custom layout functions.


# Remote Access Components For Consoles

## Component Name

noVNC

## Description

A high-performance VNC client that runs natively in the browser, utilizing HTML5 WebSockets and Canvas. It is considered a standard solution for web-based remote desktop access and is adopted by major cloud platforms.

## Supported Protocols

VNC

## Type

VNC Client

## Component Name

Apache Guacamole

## Description

A clientless remote desktop gateway that enables access to desktops through a web browser using HTML5. It is installed on a server and acts as a proxy for various remote access protocols.

## Supported Protocols

VNC, RDP, SSH

## Type

Remote Desktop Gateway

## Component Name

xterm.js

## Description

A fully-featured, high-performance frontend component that allows applications to embed terminals within a web browser. It is self-contained with zero dependencies and includes an optional GPU-accelerated renderer for enhanced performance.

## Supported Protocols

SSH (via websocket-backed terminals)

## Type

Terminal Emulator


# Suitability For Vm Management Interface

The suitability of these libraries for a VM console management interface depends on which aspect of the interface you are building.

*   **Node-Graph Library (xyflow)**: This type of library is **highly suitable for visualizing the infrastructure topology** but is **unsuitable for managing the multiple console windows themselves**. You would use xyflow to create a diagram pane showing VMs, hosts, and networks as nodes, with edges representing connections. Users could interact with this diagram to understand relationships and initiate actions. For example, clicking a 'VM' node could trigger the opening of its console. However, xyflow itself would not manage the resulting console window; it would remain a single canvas element.

*   **Window Manager (Winbox.js)**: This library is **well-suited for creating the individual, floating console windows**. If your vision is an interface where users can freely open, move, and resize multiple console windows (each containing a noVNC or xterm.js instance) in a free-form manner, Winbox.js is an excellent, lightweight choice. It provides the core windowing functionality needed to manage many simultaneous, independent sessions. Its main limitation is the lack of built-in docking, tabbing, or layout persistence.

*   **Docking Layout Manager (Golden Layout, FlexLayout-react)**: This category of library is often the **most suitable choice for the primary structure of a complex management interface**. These libraries go beyond simple floating windows to provide robust workspaces with features like docking panes to the sides, creating tabbed groups of consoles, splitting views, and saving/restoring the entire workspace layout. This is ideal for a power-user tool where a predictable and organized arrangement of multiple consoles, log viewers, and metric dashboards is required.

# Key Takeaways

- **Different Tools for Different Jobs**: xyflow and Winbox.js are not competitors. xyflow is for creating node-based diagrams, while Winbox.js is for managing floating UI windows.
- **Combine Libraries for Best Results**: The optimal solution for a VM management interface is to combine a layout manager with visualization and content libraries.
- **Choose Layout Based on UX**: For the main application structure, choose a docking manager like **Golden Layout** or **FlexLayout-react** for structured, persistent workspaces, or **Winbox.js** for simple, lightweight floating windows.
- **Use xyflow for Topology**: Integrate an xyflow canvas into one of your panes to provide a visual map of your infrastructure. Use its interactive features to trigger actions, like opening a console.
- **Embed Specialized Components**: The console windows/panes should host dedicated components like **noVNC** (for VNC), **Apache Guacamole** (for VNC/RDP/SSH), and **xterm.js** (for terminals).
- **Auto-Layout is an Integration**: xyflow does not include a built-in layout engine but supports integration with popular ones like Dagre, ELKjs, and d3-hierarchy to automatically arrange nodes.
- **Consider Alternatives**: Besides Winbox.js, mature docking frameworks like Golden Layout and FlexLayout-react offer more advanced features like tabbing, splitting, and layout saving, which are highly relevant for a complex console manager.
