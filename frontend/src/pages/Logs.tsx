import { useState } from "react";
import type { LogLine } from "../api/types";
import { fetchLogs } from "../api/client";
import { usePoll } from "../hooks/usePoll";

const LEVEL_COLORS: Record<string, string> = {
  error: "text-rose-400",
  warn: "text-amber-400",
  info: "text-sky-400",
};

export function LogsView({ logs, level, onLevel }: { logs: LogLine[]; level: string; onLevel: (l: string) => void }) {
  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl font-semibold text-white">Logs</h1>
        <div className="flex gap-1 text-xs">
          {["", "error", "warn"].map((l) => (
            <button
              key={l || "all"}
              onClick={() => onLevel(l)}
              className={`px-2 py-1 rounded ${level === l ? "bg-violet-600 text-white" : "bg-slate-800 text-slate-300"}`}
            >
              {l || "all"}
            </button>
          ))}
        </div>
      </div>
      <div className="font-mono text-xs space-y-1">
        {logs.map((l, i) => (
          <div key={i} className="flex gap-3 border-b border-slate-800 py-1">
            <span className="text-slate-500 shrink-0">
              {l.tsMs ? new Date(l.tsMs).toLocaleTimeString() : "—"}
            </span>
            <span className={`shrink-0 w-12 ${LEVEL_COLORS[l.level] ?? "text-slate-400"}`}>
              {l.level || "—"}
            </span>
            <span className="text-slate-400 shrink-0 w-44 truncate">{l.service}</span>
            <span className="text-slate-200 break-all">{l.line}</span>
            {l.traceID && <span className="text-violet-400 shrink-0">{l.traceID.slice(0, 8)}</span>}
          </div>
        ))}
        {logs.length === 0 && <div className="text-slate-500">No logs in window.</div>}
      </div>
    </div>
  );
}

export default function Logs() {
  const [level, setLevel] = useState("");
  const logs = usePoll(() => fetchLogs(level), [], 10000, [level]);
  return <LogsView logs={logs} level={level} onLevel={setLevel} />;
}
