import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { AlertsView } from "./Alerts";
import type { AlertItem } from "../api/types";

describe("AlertsView", () => {
  it("renders an alert row with runbook link", () => {
    const alerts: AlertItem[] = [
      {
        alertname: "FastBurn",
        severity: "page",
        service: "payments-worker",
        state: "firing",
        activeAt: "2026-06-18T10:00:00Z",
        summary: "fast burn",
        runbookUrl: "docs/runbooks/fast-burn.md",
      },
    ];
    render(<AlertsView alerts={alerts} />);
    expect(screen.getByText("FastBurn")).toBeInTheDocument();
    expect(screen.getByText("firing")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "runbook" })).toHaveAttribute(
      "href",
      "docs/runbooks/fast-burn.md",
    );
  });

  it("shows empty state", () => {
    render(<AlertsView alerts={[]} />);
    expect(screen.getByText(/No active alerts/)).toBeInTheDocument();
  });
});
