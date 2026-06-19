import type { AlertItem } from "../api/types";
import { fetchAlerts } from "../api/client";
import { usePoll } from "../hooks/usePoll";
import { SeverityBadge } from "../components/SeverityBadge";

export function AlertsView({ alerts }: { alerts: AlertItem[] }) {
  return (
    <div className="p-6">
      <h1 className="text-xl font-semibold text-white mb-4">Active Alerts</h1>
      <table className="w-full text-sm text-slate-200">
        <thead>
          <tr className="text-slate-400 text-left">
            <th className="py-1">Severity</th>
            <th>Alert</th>
            <th>Service</th>
            <th>State</th>
            <th>Since</th>
            <th>Runbook</th>
          </tr>
        </thead>
        <tbody>
          {alerts.map((a, i) => (
            <tr key={a.alertname + a.service + i} className="border-t border-slate-700">
              <td className="py-2">
                <SeverityBadge severity={a.severity} />
              </td>
              <td>{a.alertname}</td>
              <td>{a.service || "—"}</td>
              <td>
                <span className={a.state === "firing" ? "text-rose-400" : "text-amber-400"}>
                  {a.state}
                </span>
              </td>
              <td className="text-slate-400">{a.activeAt ? new Date(a.activeAt).toLocaleTimeString() : "—"}</td>
              <td>
                {a.runbookUrl ? (
                  <a className="text-violet-400 hover:underline" href={a.runbookUrl}>
                    runbook
                  </a>
                ) : (
                  "—"
                )}
              </td>
            </tr>
          ))}
          {alerts.length === 0 && (
            <tr>
              <td colSpan={6} className="py-3 text-slate-500">
                No active alerts 🎉
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

export default function Alerts() {
  const alerts = usePoll(fetchAlerts, []);
  return <AlertsView alerts={alerts} />;
}
