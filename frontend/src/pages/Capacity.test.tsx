import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { CapacityView } from "./Capacity";
import type { CapacityItem } from "../api/types";

describe("CapacityView", () => {
  it("renders a container row with shortened name and memory", () => {
    const items: CapacityItem[] = [
      { name: "compose-orders-api-1", cpuPct: 5, memMB: 50, diskPct: 42 },
    ];
    render(<CapacityView items={items} />);
    expect(screen.getByText("orders-api")).toBeInTheDocument();
    expect(screen.getByText("50 MB")).toBeInTheDocument();
    expect(screen.getByText("42%")).toBeInTheDocument();
  });
});
