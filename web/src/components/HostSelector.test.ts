import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHostSelector } from "./HostSelector";

describe("HostSelector", () => {
  let container: HTMLElement;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
  });

  it("renders with no hosts", () => {
    renderHostSelector(container, {
      hosts: [],
      selectedHostId: null,
      onChange: () => {},
    });
    const select = container.querySelector("select");
    expect(select?.options.length).toBe(1);
    expect(select?.options[0].textContent).toBe("No hosts");
  });

  it("renders hosts with selection", () => {
    renderHostSelector(container, {
      hosts: [
        { id: "h1", uri: "qemu:///system" },
        { id: "h2", uri: "qemu:///session" },
      ],
      selectedHostId: "h2",
      onChange: () => {},
    });
    const select = container.querySelector("select");
    expect(select?.options.length).toBe(2);
    expect(select?.value).toBe("h2");
  });

  it("onChange called when selection changes", () => {
    const onChange = vi.fn();
    renderHostSelector(container, {
      hosts: [
        { id: "h1", uri: "qemu:///system" },
        { id: "h2", uri: "qemu:///session" },
      ],
      selectedHostId: "h1",
      onChange,
    });
    const select = container.querySelector("select") as HTMLSelectElement;
    select.value = "h2";
    select.dispatchEvent(new Event("change"));
    expect(onChange).toHaveBeenCalledWith("h2");
  });

  it("disabled when props.disabled", () => {
    renderHostSelector(container, {
      hosts: [{ id: "h1", uri: "qemu:///system" }],
      selectedHostId: "h1",
      onChange: () => {},
      disabled: true,
    });
    expect((container.querySelector("select") as HTMLSelectElement)?.disabled).toBe(true);
  });
});
