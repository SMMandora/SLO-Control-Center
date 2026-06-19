import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ServiceHealthMap } from "./ServiceHealthMap";
import type { SloSummary } from "../api/types";

function svc(service: string, healthy: boolean): SloSummary {
  return {
    service,
    sliPct: 99,
    targetPct: 99,
    errorBudgetRemainingPct: 50,
    errorBudgetRemainingCount: 1,
    burnRate: { "1h": 0, "6h": 0, "24h": 0 },
    p95Ms: 1,
    healthy,
  };
}

describe("ServiceHealthMap", () => {
  it("renders a 3-node dependency chain with arrows between", () => {
    render(
      <ServiceHealthMap
        rows={[
          svc("notification-svc", true),
          svc("orders-api", true),
          svc("payments-worker", false),
        ]}
      />,
    );
    const nodes = screen.getAllByTestId("health-node");
    expect(nodes).toHaveLength(3);
    // dependency order is enforced regardless of input order
    expect(nodes.map((n) => n.textContent)).toEqual([
      "orders-api",
      "payments-worker",
      "notification-svc",
    ]);
    expect(screen.getAllByTestId("health-arrow")).toHaveLength(2);
  });
});
