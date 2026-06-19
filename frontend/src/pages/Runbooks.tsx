import { useState } from "react";
import ReactMarkdown from "react-markdown";
import type { Runbook } from "../api/types";
import { fetchRunbooks } from "../api/client";
import { usePoll } from "../hooks/usePoll";

export function RunbooksView({ runbooks }: { runbooks: Runbook[] }) {
  const [selected, setSelected] = useState<string | null>(null);
  const active = runbooks.find((r) => r.name === selected) ?? runbooks[0];
  return (
    <div className="p-6 flex gap-6">
      <aside className="w-56 shrink-0">
        <h2 className="text-slate-400 text-xs uppercase mb-2">Runbooks</h2>
        <ul className="space-y-1">
          {runbooks.map((r) => (
            <li key={r.name}>
              <button
                onClick={() => setSelected(r.name)}
                className={`text-left text-sm w-full px-2 py-1 rounded ${
                  active?.name === r.name ? "bg-slate-800 text-white" : "text-slate-300 hover:text-white"
                }`}
              >
                {r.title}
              </button>
            </li>
          ))}
          {runbooks.length === 0 && <li className="text-slate-500 text-sm">No runbooks.</li>}
        </ul>
      </aside>
      <article className="prose prose-invert max-w-none flex-1 text-slate-200">
        {active ? (
          <ReactMarkdown>{active.markdown}</ReactMarkdown>
        ) : (
          <div className="text-slate-500">Select a runbook.</div>
        )}
      </article>
    </div>
  );
}

export default function Runbooks() {
  const runbooks = usePoll(fetchRunbooks, [], 60000);
  return <RunbooksView runbooks={runbooks} />;
}
