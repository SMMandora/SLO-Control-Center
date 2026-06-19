import { fetchAllTraces } from "../api/client";
import { usePoll } from "../hooks/usePoll";
import { RecentTraces } from "../components/RecentTraces";

export default function Traces() {
  const traces = usePoll(fetchAllTraces, []);
  return (
    <div className="p-6">
      <h1 className="text-xl font-semibold text-white mb-4">Recent Traces</h1>
      <p className="text-xs text-slate-500 mb-3">
        Recent server traces from Tempo. Open a trace in Grafana for the full waterfall.
      </p>
      <RecentTraces traces={traces} />
    </div>
  );
}
