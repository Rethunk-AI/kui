import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderInlineHostSelector } from "./InlineHostSelector";

describe("InlineHostSelector", () => {
  let container: HTMLElement;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
  });

  it("renders with default label", () => {
    renderInlineHostSelector(container, {
      hosts: [],
      selectedHostId: null,
      onChange: () => {},
    });
    expect(container.querySelector(".inline-host-selector__label")?.textContent).toContain("Host");
  });

  it("renders with custom label", () => {
    renderInlineHostSelector(container, {
      hosts: [],
      selectedHostId: null,
      onChange: () => {},
      label: "Target host",
    });
    expect(container.querySelector(".inline-host-selector__label")?.textContent).toContain("Target host");
  });

  it("renders hosts", () => {
    renderInlineHostSelector(container, {
      hosts: [
        { id: "h1", uri: "qemu:///system" },
        { id: "h2", uri: "qemu:///session" },
      ],
      selectedHostId: "h1",
      onChange: () => {},
    });
    const select = container.querySelector("select");
    expect(select?.options.length).toBe(2);
    expect(select?.value).toBe("h1");
  });

  it("onChange called on change", () => {
    const onChange = vi.fn();
    renderInlineHostSelector(container, {
      hosts: [{ id: "h1", uri: "qemu:///system" }],
      selectedHostId: "h1",
      onChange,
    });
    const select = container.querySelector("select") as HTMLSelectElement;
    select.dispatchEvent(new Event("change"));
    expect(onChange).toHaveBeenCalledWith("h1");
  });

  it("required sets aria-required", () => {
    renderInlineHostSelector(container, {
      hosts: [{ id: "h1", uri: "qemu:///system" }],
      selectedHostId: "h1",
      onChange: () => {},
      required: true,
    });
    expect(container.querySelector("select")?.getAttribute("aria-required")).toBe("true");
  });
});
