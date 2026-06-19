import type { SloSummary } from "../api/types";

// Dependency order of the mesh; nodes render left-to-right with arrows between.
const ORDER = ["orders-api", "payments-worker", "notification-svc"];

export function ServiceHealthMap({ rows }: { rows: SloSummary[] }) {
  const byName = new Map(rows.map((r) => [r.service, r]));
  const chain = ORDER.filter((n) => byName.has(n)).map((n) => byName.get(n)!);
  const extra = rows.filter((r) => !ORDER.includes(r.service));
  const nodes = [...chain, ...extra];

  return (
    <div className="flex items-center gap-3 flex-wrap">
      {nodes.map((r, i) => (
        <div key={r.service} className="flex items-center gap-3">
          <div
            data-testid="health-node"
            className={`px-5 py-3 rounded-full text-white text-sm font-medium ${
              r.healthy ? "bg-emerald-600" : "bg-rose-600"
            }`}
          >
            {r.service}
          </div>
          {i < nodes.length - 1 && (
            <span data-testid="health-arrow" className="text-slate-500 text-xl">
              →
            </span>
          )}
        </div>
      ))}
      {nodes.length === 0 && <div className="text-slate-500 text-sm">No services reporting</div>}
    </div>
  );
}
