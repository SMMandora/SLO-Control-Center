local g = import 'github.com/grafana/grafonnet/gen/grafonnet-v10.0.0/main.libsonnet';
local panel = g.panel;
local prom = g.query.prometheus;
local loki = g.query.loki;

g.dashboard.new('Incident Investigation')
+ g.dashboard.withUid('incident-investigation')
+ g.dashboard.withRefresh('15s')
+ g.dashboard.time.withFrom('now-1h')
+ g.dashboard.withPanels([
  panel.timeSeries.new('Error Rate (per second)')
  + panel.timeSeries.queryOptions.withTargets([
    prom.new('Prometheus', 'sum(rate(payments_processed_total{status="failure"}[5m]))')
    + prom.withLegendFormat('payments failures/s'),
    prom.new('Prometheus', 'sum(rate(http_requests_total{job="orders-api",status=~"5.."}[5m]))')
    + prom.withLegendFormat('orders 5xx/s'),
  ])
  + panel.timeSeries.gridPos.withW(24) + panel.timeSeries.gridPos.withH(8)
  + panel.timeSeries.gridPos.withX(0) + panel.timeSeries.gridPos.withY(0),

  panel.timeSeries.new('Burn Rate (1h) by Service')
  + panel.timeSeries.queryOptions.withTargets([
    prom.new('Prometheus', 'slo:orders_api:burnrate_1h') + prom.withLegendFormat('orders-api'),
    prom.new('Prometheus', 'slo:payments_worker:burnrate_1h') + prom.withLegendFormat('payments-worker'),
    prom.new('Prometheus', 'slo:notification_svc:burnrate_1h') + prom.withLegendFormat('notification-svc'),
  ])
  + panel.timeSeries.gridPos.withW(24) + panel.timeSeries.gridPos.withH(8)
  + panel.timeSeries.gridPos.withX(0) + panel.timeSeries.gridPos.withY(8),

  panel.logs.new('Error Logs (click trace_id → Tempo)')
  + panel.logs.queryOptions.withTargets([
    loki.new('Loki', '{level="error"}'),
  ])
  + panel.logs.options.withShowTime(true)
  + panel.logs.gridPos.withW(24) + panel.logs.gridPos.withH(10)
  + panel.logs.gridPos.withX(0) + panel.logs.gridPos.withY(16),
])
