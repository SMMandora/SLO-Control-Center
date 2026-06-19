import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ServicesView } from "./Services";
import type { ServiceDetail } from "../api/types";

describe("ServicesView", () => {
  it("renders a service card with RED + deps", () => {
    const services: ServiceDetail[] = [
      {
        service: "payments-worker",
        sliPct: 92.5,
        targetPct: 99.5,
        healthy: false,
        ratePerSec: 48.2,
        errorPct: 7.5,
        p95Ms: 480,
        dependencies: ["redis", "mock-gateway"],
      },
    ];
    render(<ServicesView services={services} />);
    expect(screen.getByText("payments-worker")).toBeInTheDocument();
    expect(screen.getByText("breaching")).toBeInTheDocument();
    expect(screen.getByText("mock-gateway")).toBeInTheDocument();
    expect(screen.getByText("480ms")).toBeInTheDocument();
  });
});
