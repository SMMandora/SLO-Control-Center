local g = import 'github.com/grafana/grafonnet/gen/grafonnet-v10.0.0/main.libsonnet';
local panel = g.panel;
local prom = g.query.prometheus;
local var = g.dashboard.variable;

// $service is the recording-rule prefix segment (underscore form).
g.dashboard.new('Service Drilldown')
+ g.dashboard.withUid('service-drilldown')
+ g.dashboard.withRefresh('15s')
+ g.dashboard.time.withFrom('now-6h')
+ g.dashboard.withVariables([
  var.custom.new('service', ['orders_api', 'payments_worker', 'notification_svc']),
])
+ g.dashboard.withPanels([
  panel.gauge.new('Error Budget Remaining')
  + panel.gauge.queryOptions.withTargets([
    prom.new('Prometheus', 'slo:$service:error_budget_remaining_ratio * 100'),
  ])
  + panel.gauge.standardOptions.withUnit('percent')
  + panel.gauge.standardOptions.withMin(0) + panel.gauge.standardOptions.withMax(100)
  + panel.gauge.gridPos.withW(8) + panel.gauge.gridPos.withH(6) + panel.gauge.gridPos.withX(0) + panel.gauge.gridPos.withY(0),

  panel.stat.new('p95 Latency')
  + panel.stat.queryOptions.withTargets([
    prom.new('Prometheus', 'slo:$service:latency_p95_seconds * 1000'),
  ])
  + panel.stat.standardOptions.withUnit('ms')
  + panel.stat.gridPos.withW(8) + panel.stat.gridPos.withH(6) + panel.stat.gridPos.withX(8) + panel.stat.gridPos.withY(0),

  panel.stat.new('Requests (28d)')
  + panel.stat.queryOptions.withTargets([
    prom.new('Prometheus', 'slo:$service:requests_total_28d'),
  ])
  + panel.stat.gridPos.withW(8) + panel.stat.gridPos.withH(6) + panel.stat.gridPos.withX(16) + panel.stat.gridPos.withY(0),

  panel.timeSeries.new('Burn Rate (1h / 6h / 24h)')
  + panel.timeSeries.queryOptions.withTargets([
    prom.new('Prometheus', 'slo:$service:burnrate_1h') + prom.withLegendFormat('1h'),
    prom.new('Prometheus', 'slo:$service:burnrate_6h') + prom.withLegendFormat('6h'),
    prom.new('Prometheus', 'slo:$service:burnrate_24h') + prom.withLegendFormat('24h'),
  ])
  + panel.timeSeries.gridPos.withW(24) + panel.timeSeries.gridPos.withH(8) + panel.timeSeries.gridPos.withX(0) + panel.timeSeries.gridPos.withY(6),

  panel.timeSeries.new('Dependency Liveness (up)')
  + panel.timeSeries.queryOptions.withTargets([
    prom.new('Prometheus', 'up{job=~"orders-api|payments-worker|notification-svc|mock-gateway|mock-receiver"}')
    + prom.withLegendFormat('{{job}}'),
  ])
  + panel.timeSeries.standardOptions.withMin(0) + panel.timeSeries.standardOptions.withMax(1)
  + panel.timeSeries.gridPos.withW(24) + panel.timeSeries.gridPos.withH(7) + panel.timeSeries.gridPos.withX(0) + panel.timeSeries.gridPos.withY(14),
])
