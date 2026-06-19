local g = import 'github.com/grafana/grafonnet/gen/grafonnet-v10.0.0/main.libsonnet';
local panel = g.panel;
local prom = g.query.prometheus;

local ts(title, expr, unit, y) =
  panel.timeSeries.new(title)
  + panel.timeSeries.queryOptions.withTargets([
    prom.new('Prometheus', expr) + prom.withLegendFormat('{{name}}'),
  ])
  + panel.timeSeries.standardOptions.withUnit(unit)
  + panel.timeSeries.gridPos.withW(12) + panel.timeSeries.gridPos.withH(8)
  + panel.timeSeries.gridPos.withX(if y % 16 == 0 then 0 else 12)
  + panel.timeSeries.gridPos.withY(y);

g.dashboard.new('Capacity / USE')
+ g.dashboard.withUid('capacity-use')
+ g.dashboard.withRefresh('30s')
+ g.dashboard.time.withFrom('now-1h')
+ g.dashboard.withPanels([
  ts('CPU (cores)', 'sum by (name) (rate(container_cpu_usage_seconds_total{name=~"compose-.+"}[2m]))', 'short', 0),
  ts('Memory', 'sum by (name) (container_memory_usage_bytes{name=~"compose-.+"})', 'bytes', 0),
  ts('Disk usage (%)', 'max by (name) (container_fs_usage_bytes{name=~"compose-.+"} / container_fs_limit_bytes) * 100', 'percent', 8),
  ts('Network RX', 'sum by (name) (rate(container_network_receive_bytes_total{name=~"compose-.+"}[2m]))', 'Bps', 8),
])
