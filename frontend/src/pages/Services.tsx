import type { ServiceDetail } from "../api/types";
import { fetchServices } from "../api/client";
import { usePoll } from "../hooks/usePoll";

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs text-slate-400">{label}</div>
      <div className="text-slate-100">{value}</div>
    </div>
  );
}

export function ServicesView({ services }: { services: ServiceDetail[] }) {
  return (
    <div className="p-6 grid gap-4 md:grid-cols-3">
      {services.map((s) => (
        <div key={s.service} className="rounded-xl bg-slate-800/60 p-4 border border-slate-700">
          <div className="flex justify-between items-center">
            <span className="font-semibold text-white">{s.service}</span>
            <span
              className={`text-xs px-2 py-0.5 rounded ${s.healthy ? "bg-emerald-700" : "bg-rose-700"}`}
            >
              {s.healthy ? "healthy" : "breaching"}
            </span>
          </div>
          <div className="mt-3 grid grid-cols-3 gap-2">
            <Metric label="Availability" value={`${s.sliPct}%`} />
            <Metric label="Target" value={`${s.targetPct}%`} />
            <Metric label="p95" value={`${s.p95Ms}ms`} />
            <Metric label="Rate" value={`${s.ratePerSec}/s`} />
            <Metric label="Error" value={`${s.errorPct}%`} />
          </div>
          <div className="mt-3">
            <div className="text-xs text-slate-400 mb-1">Dependencies</div>
            <div className="flex gap-1 flex-wrap">
              {s.dependencies.map((d) => (
                <span key={d} className="text-xs bg-slate-700 px-2 py-0.5 rounded text-slate-200">
                  {d}
                </span>
              ))}
            </div>
          </div>
        </div>
      ))}
      {services.length === 0 && <div className="text-slate-500">Loading services…</div>}
    </div>
  );
}

export default function Services() {
  const services = usePoll(fetchServices, []);
  return <ServicesView services={services} />;
}
