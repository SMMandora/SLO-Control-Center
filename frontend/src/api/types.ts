export interface BurnRate {
  "1h": number;
  "6h": number;
  "24h": number;
}

export interface SloSummary {
  service: string;
  sliPct: number;
  targetPct: number;
  errorBudgetRemainingPct: number;
  errorBudgetRemainingCount: number;
  burnRate: BurnRate;
  p95Ms: number;
  healthy: boolean;
}

export interface CompliancePoint {
  t: number;
  sliPct: number;
}

export interface ComplianceSeries {
  points: CompliancePoint[];
}

export interface TraceRef {
  traceID: string;
  service: string;
  name: string;
  durationMs: number;
  startedMs: number;
  grafanaUrl: string;
}

export interface ServiceDetail {
  service: string;
  sliPct: number;
  targetPct: number;
  healthy: boolean;
  ratePerSec: number;
  errorPct: number;
  p95Ms: number;
  dependencies: string[];
}

export interface AlertItem {
  alertname: string;
  severity: string;
  service: string;
  state: string;
  activeAt: string;
  summary: string;
  runbookUrl: string;
}

export interface Incident {
  id: string;
  title: string;
  service: string;
  severity: string;
  startedAt: string;
  alertCount: number;
}

export interface LogLine {
  tsMs: number;
  level: string;
  service: string;
  traceID: string;
  line: string;
}

export interface CapacityItem {
  name: string;
  cpuPct: number;
  memMB: number;
  diskPct: number;
}

export interface Runbook {
  name: string;
  title: string;
  markdown: string;
}
