import type { SloSummary } from "../api/types";

export function ErrorBudgetTable({ rows }: { rows: SloSummary[] }) {
  return (
    <table className="w-full text-sm text-slate-200">
      <thead>
        <tr className="text-slate-400 text-left">
          <th className="py-1">Service</th>
          <th>SLO</th>
          <th>Availability</th>
          <th>Error Budget</th>
          <th>Burn 1h</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => (
          <tr key={r.service} className="border-t border-slate-700">
            <td className="py-2">{r.service}</td>
            <td>{r.targetPct}%</td>
            <td>{r.sliPct}%</td>
            <td>
              {r.errorBudgetRemainingPct}% remaining
              <span className="text-slate-500"> ({r.errorBudgetRemainingCount})</span>
            </td>
            <td>{r.burnRate["1h"]}x</td>
          </tr>
        ))}
        {rows.length === 0 && (
          <tr>
            <td colSpan={5} className="py-3 text-slate-500">
              Waiting for data…
            </td>
          </tr>
        )}
      </tbody>
    </table>
  );
}
