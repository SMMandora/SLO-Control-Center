import type {
  SloSummary,
  ComplianceSeries,
  TraceRef,
  ServiceDetail,
  AlertItem,
  Incident,
  LogLine,
  CapacityItem,
  Runbook,
} from "./types";

const BASE = import.meta.env.VITE_BFF_URL ?? "http://localhost:9090";

async function get<T>(path: string): Promise<T> {
  const r = await fetch(`${BASE}${path}`);
  return r.json();
}

export const fetchSlo = () => get<SloSummary[]>("/api/slo");
export const fetchCompliance = (service: string) =>
  get<ComplianceSeries>(`/api/slo/${service}/compliance`);
export const fetchRecentTraces = () => get<TraceRef[]>("/api/traces/recent");
export const fetchAllTraces = () => get<TraceRef[]>("/api/traces/recent?status=any");
export const fetchServices = () => get<ServiceDetail[]>("/api/services");
export const fetchAlerts = () => get<AlertItem[]>("/api/alerts");
export const fetchIncidents = () => get<Incident[]>("/api/incidents");
export const fetchLogs = (level = "") => get<LogLine[]>(`/api/logs${level ? `?level=${level}` : ""}`);
export const fetchCapacity = () => get<CapacityItem[]>("/api/capacity");
export const fetchRunbooks = () => get<Runbook[]>("/api/runbooks");
