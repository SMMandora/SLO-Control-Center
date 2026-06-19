local g = import 'github.com/grafana/grafonnet/gen/grafonnet-v10.0.0/main.libsonnet';
local panel = g.panel;
local q = g.query.prometheus;

g.dashboard.new('SLO Overview - orders-api')
+ g.dashboard.withUid('slo-overview')
+ g.dashboard.withRefresh('15s')
+ g.dashboard.time.withFrom('now-6h')
+ g.dashboard.withPanels([
  panel.stat.new('Availability (5m)')
  + panel.stat.queryOptions.withTargets([
    q.new('Prometheus', 'slo:orders_api:availability:ratio_5m * 100'),
  ])
  + panel.stat.standardOptions.withUnit('percent')
  + panel.stat.gridPos.withW(6) + panel.stat.gridPos.withH(6)
  + panel.stat.gridPos.withX(0) + panel.stat.gridPos.withY(0),

  panel.gauge.new('Error Budget Remaining')
  + panel.gauge.queryOptions.withTargets([
    q.new('Prometheus', 'slo:orders_api:error_budget_remaining_ratio * 100'),
  ])
  + panel.gauge.standardOptions.withUnit('percent')
  + panel.gauge.standardOptions.withMin(0) + panel.gauge.standardOptions.withMax(100)
  + panel.gauge.gridPos.withW(6) + panel.gauge.gridPos.withH(6)
  + panel.gauge.gridPos.withX(6) + panel.gauge.gridPos.withY(0),

  panel.stat.new('p95 Latency')
  + panel.stat.queryOptions.withTargets([
    q.new('Prometheus', 'slo:orders_api:latency_p95_seconds * 1000'),
  ])
  + panel.stat.standardOptions.withUnit('ms')
  + panel.stat.gridPos.withW(6) + panel.stat.gridPos.withH(6)
  + panel.stat.gridPos.withX(12) + panel.stat.gridPos.withY(0),

  panel.timeSeries.new('Burn Rate (1h / 6h / 24h)')
  + panel.timeSeries.queryOptions.withTargets([
    q.new('Prometheus', 'slo:orders_api:burnrate_1h') + q.withLegendFormat('1h'),
    q.new('Prometheus', 'slo:orders_api:burnrate_6h') + q.withLegendFormat('6h'),
    q.new('Prometheus', 'slo:orders_api:burnrate_24h') + q.withLegendFormat('24h'),
  ])
  + panel.timeSeries.gridPos.withW(12) + panel.timeSeries.gridPos.withH(8)
  + panel.timeSeries.gridPos.withX(0) + panel.timeSeries.gridPos.withY(6),

  panel.timeSeries.new('p95 Latency (ms)')
  + panel.timeSeries.queryOptions.withTargets([
    q.new('Prometheus', 'slo:orders_api:latency_p95_seconds * 1000') + q.withLegendFormat('p95'),
  ])
  + panel.timeSeries.standardOptions.withUnit('ms')
  + panel.timeSeries.gridPos.withW(12) + panel.timeSeries.gridPos.withH(8)
  + panel.timeSeries.gridPos.withX(12) + panel.timeSeries.gridPos.withY(6),
])
