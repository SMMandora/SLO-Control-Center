import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { RecentTraces } from "./RecentTraces";
import type { TraceRef } from "../api/types";

describe("RecentTraces", () => {
  it("renders a trace row linking to Grafana", () => {
    const traces: TraceRef[] = [
      {
        traceID: "abcdef0123456789",
        service: "payments-worker",
        name: "process-payment",
        durationMs: 1200,
        startedMs: 1700000000000,
        grafanaUrl: "http://localhost:3001/explore?left=x",
      },
    ];
    render(<RecentTraces traces={traces} />);
    expect(screen.getByText("payments-worker")).toBeInTheDocument();
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "http://localhost:3001/explore?left=x");
  });

  it("shows an empty state with no traces", () => {
    render(<RecentTraces traces={[]} />);
    expect(screen.getByText(/No recent violations/)).toBeInTheDocument();
  });
});
