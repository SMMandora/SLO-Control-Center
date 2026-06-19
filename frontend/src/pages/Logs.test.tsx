import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { LogsView } from "./Logs";
import type { LogLine } from "../api/types";

describe("LogsView", () => {
  it("renders a log line with level and message", () => {
    const logs: LogLine[] = [
      { tsMs: 1700000000000, level: "error", service: "compose-payments-worker-1", traceID: "abcdef12", line: "payment failed" },
    ];
    render(<LogsView logs={logs} level="" onLevel={vi.fn()} />);
    expect(screen.getByText("payment failed")).toBeInTheDocument();
    expect(screen.getByText("abcdef12")).toBeInTheDocument();
    // "error" appears both as a filter button and the line's level cell
    expect(screen.getAllByText("error").length).toBeGreaterThan(1);
  });
});
