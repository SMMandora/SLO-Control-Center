import type { SloSummary, ComplianceSeries, TraceRef } from "../api/types";
import { StatCard } from "../components/StatCard";
import { ErrorBudgetTable } from "../components/ErrorBudgetTable";
import { ComplianceChart } from "../components/ComplianceChart";
import { ServiceHealthMap } from "../components/ServiceHealthMap";
import { RecentTraces } from "../components/RecentTraces";
import { useSlo } from "../hooks/useSlo";

// OverviewView is pure/prop-driven so it can be rendered in tests without fetching.
export function OverviewView({
  summaries,
  compliance,
  traces = [],
}: {
  summaries: SloSummary[];
  compliance: ComplianceSeries;
  traces?: TraceRef[];
}) {
  const primary = summaries[0];
  const healthy = summaries.filter((s) => s.healthy).length;
  return (
    <div className="space-y-6 p-6">
      <h1 className="text-xl font-semibold text-white">Global SLO Overview Dashboard</h1>
      <div className="grid grid-cols-2 md:grid-cols-6 gap-3">
        <StatCard label="Availability" value={primary ? `${primary.sliPct}%` : "—"} />
        <StatCard label="Latency p95" value={primary ? `${primary.p95Ms}ms` : "—"} sub="p95" />
        <StatCard
          label="Error Budget Remaining"
          value={primary ? `${primary.errorBudgetRemainingPct}%` : "—"}
        />
        <StatCard
          label="Current Burn Rate"
          value={primary ? `${primary.burnRate["1h"]}x` : "—"}
        />
        <StatCard label="Services Healthy" value={`${healthy} / ${summaries.length}`} />
        <StatCard label="Open Incidents" value="0" />
      </div>
      <div className="rounded-xl bg-slate-800/60 p-4 border border-slate-700">
        <h2 className="text-slate-200 mb-3">Error Budgets by Service</h2>
        <ErrorBudgetTable rows={summaries} />
      </div>
      <div className="rounded-xl bg-slate-800/60 p-4 border border-slate-700">
        <h2 className="text-slate-200 mb-3">28-Day Compliance Graph</h2>
        <ComplianceChart series={compliance} />
      </div>
      <div className="rounded-xl bg-slate-800/60 p-4 border border-slate-700">
        <h2 className="text-slate-200 mb-3">Live Service Health Map</h2>
        <ServiceHealthMap rows={summaries} />
      </div>
      <div className="rounded-xl bg-slate-800/60 p-4 border border-slate-700">
        <h2 className="text-slate-200 mb-3">Recent Violations → Trace</h2>
        <RecentTraces traces={traces} />
      </div>
    </div>
  );
}

export default function Overview() {
  const { summaries, compliance, traces } = useSlo();
  return <OverviewView summaries={summaries} compliance={compliance} traces={traces} />;
}
