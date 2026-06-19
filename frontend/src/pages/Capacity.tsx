import type { CapacityItem } from "../api/types";
import { fetchCapacity } from "../api/client";
import { usePoll } from "../hooks/usePoll";

function Bar({ pct, color }: { pct: number; color: string }) {
  return (
    <div className="h-2 w-full rounded bg-slate-700 overflow-hidden">
      <div className={`h-full ${color}`} style={{ width: `${Math.min(100, Math.max(0, pct))}%` }} />
    </div>
  );
}

function short(name: string) {
  return name.replace(/^compose-/, "").replace(/-1$/, "");
}

export function CapacityView({ items }: { items: CapacityItem[] }) {
  return (
    <div className="p-6">
      <h1 className="text-xl font-semibold text-white mb-4">Capacity / USE</h1>
      <table className="w-full text-sm text-slate-200">
        <thead>
          <tr className="text-slate-400 text-left">
            <th className="py-1">Container</th>
            <th className="w-48">CPU</th>
            <th className="w-48">Memory</th>
            <th className="w-48">Disk</th>
          </tr>
        </thead>
        <tbody>
          {items.map((it) => (
            <tr key={it.name} className="border-t border-slate-700">
              <td className="py-2">{short(it.name)}</td>
              <td>
                <div className="flex items-center gap-2">
                  <Bar pct={it.cpuPct} color="bg-sky-500" />
                  <span className="text-xs text-slate-400 w-12">{it.cpuPct}%</span>
                </div>
              </td>
              <td className="text-slate-300">{it.memMB} MB</td>
              <td>
                <div className="flex items-center gap-2">
                  <Bar pct={it.diskPct} color={it.diskPct > 90 ? "bg-rose-500" : "bg-emerald-500"} />
                  <span className="text-xs text-slate-400 w-12">{it.diskPct}%</span>
                </div>
              </td>
            </tr>
          ))}
          {items.length === 0 && (
            <tr>
              <td colSpan={4} className="py-3 text-slate-500">
                Loading container metrics…
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

export default function Capacity() {
  const items = usePoll(fetchCapacity, []);
  return <CapacityView items={items} />;
}
