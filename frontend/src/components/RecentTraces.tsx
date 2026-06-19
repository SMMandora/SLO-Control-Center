import type { TraceRef } from "../api/types";

export function RecentTraces({ traces }: { traces: TraceRef[] }) {
  return (
    <table className="w-full text-sm text-slate-200">
      <thead>
        <tr className="text-slate-400 text-left">
          <th className="py-1">Service</th>
          <th>Operation</th>
          <th>Duration</th>
          <th>Trace</th>
        </tr>
      </thead>
      <tbody>
        {traces.map((t) => (
          <tr key={t.traceID} className="border-t border-slate-700">
            <td className="py-2">{t.service || "—"}</td>
            <td>{t.name || "—"}</td>
            <td>{t.durationMs}ms</td>
            <td>
              <a
                className="text-violet-400 hover:underline"
                href={t.grafanaUrl}
                target="_blank"
                rel="noreferrer"
              >
                {t.traceID.slice(0, 12)}…
              </a>
            </td>
          </tr>
        ))}
        {traces.length === 0 && (
          <tr>
            <td colSpan={4} className="py-3 text-slate-500">
              No recent violations 🎉
            </td>
          </tr>
        )}
      </tbody>
    </table>
  );
}
