import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { ComplianceSeries } from "../api/types";

export function ComplianceChart({ series }: { series: ComplianceSeries }) {
  const data = series.points.map((p) => ({
    date: new Date(p.t * 1000).toLocaleDateString(),
    SLI: p.sliPct,
  }));
  return (
    <div className="h-64 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data}>
          <defs>
            <linearGradient id="sli" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#34d399" stopOpacity={0.5} />
              <stop offset="95%" stopColor="#34d399" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
          <XAxis dataKey="date" tick={{ fill: "#94a3b8", fontSize: 11 }} />
          <YAxis domain={[0, 100]} tick={{ fill: "#94a3b8", fontSize: 11 }} />
          <Tooltip
            contentStyle={{ background: "#1e293b", border: "1px solid #334155", color: "#e2e8f0" }}
          />
          <Area type="monotone" dataKey="SLI" stroke="#34d399" fill="url(#sli)" />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
