import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { OverviewView } from "./Overview";
import type { SloSummary } from "../api/types";

const sample: SloSummary[] = [
  {
    service: "orders-api",
    sliPct: 99.93,
    targetPct: 99.9,
    errorBudgetRemainingPct: 84.2,
    errorBudgetRemainingCount: 84,
    burnRate: { "1h": 0.3, "6h": 0.5, "24h": 0.4 },
    p95Ms: 198,
    healthy: true,
  },
];

describe("OverviewView", () => {
  it("renders availability and a service row", () => {
    render(<OverviewView summaries={sample} compliance={{ points: [] }} />);
    // Availability value appears in both the stat card and the table row.
    expect(screen.getAllByText(/99.93%/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/orders-api/).length).toBeGreaterThan(0);
    expect(screen.getByText("Error Budgets by Service")).toBeInTheDocument();
  });
});
