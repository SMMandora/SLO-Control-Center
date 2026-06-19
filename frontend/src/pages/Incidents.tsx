import type { Incident } from "../api/types";
import { fetchIncidents } from "../api/client";
import { usePoll } from "../hooks/usePoll";
import { SeverityBadge } from "../components/SeverityBadge";

export function IncidentsView({ incidents }: { incidents: Incident[] }) {
  return (
    <div className="p-6">
      <h1 className="text-xl font-semibold text-white mb-4">Incidents</h1>
      <p className="text-xs text-slate-500 mb-3">
        Derived from currently-firing alerts, grouped by alert and service.
      </p>
      <div className="space-y-2">
        {incidents.map((inc) => (
          <div
            key={inc.id}
            className="rounded-lg bg-slate-800/60 border border-slate-700 p-3 flex items-center justify-between"
          >
            <div>
              <div className="text-white">{inc.title}</div>
              <div className="text-xs text-slate-400">
                {inc.service || "—"} · since{" "}
                {inc.startedAt ? new Date(inc.startedAt).toLocaleString() : "—"} · {inc.alertCount}{" "}
                alert{inc.alertCount === 1 ? "" : "s"}
              </div>
            </div>
            <SeverityBadge severity={inc.severity} />
          </div>
        ))}
        {incidents.length === 0 && <div className="text-slate-500">No open incidents 🎉</div>}
      </div>
    </div>
  );
}

export default function Incidents() {
  const incidents = usePoll(fetchIncidents, []);
  return <IncidentsView incidents={incidents} />;
}
