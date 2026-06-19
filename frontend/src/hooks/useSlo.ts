import { useEffect, useState } from "react";
import { fetchSlo, fetchCompliance, fetchRecentTraces } from "../api/client";
import type { SloSummary, ComplianceSeries, TraceRef } from "../api/types";

// useSlo polls the BFF every 15s and exposes the latest good snapshot.
export function useSlo() {
  const [summaries, setSummaries] = useState<SloSummary[]>([]);
  const [compliance, setCompliance] = useState<ComplianceSeries>({ points: [] });
  const [traces, setTraces] = useState<TraceRef[]>([]);

  useEffect(() => {
    let alive = true;
    const tick = async () => {
      try {
        const s = await fetchSlo();
        if (!alive) return;
        setSummaries(s);
        if (s[0]) setCompliance(await fetchCompliance(s[0].service));
        setTraces(await fetchRecentTraces());
      } catch {
        /* keep last good snapshot */
      }
    };
    tick();
    const id = setInterval(tick, 15000);
    return () => {
      alive = false;
      clearInterval(id);
    };
  }, []);

  return { summaries, compliance, traces };
}
