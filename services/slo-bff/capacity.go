package main

import (
	"context"
	"math"
	"sort"
)

// CapacityItem is per-container resource usage for the Capacity tab.
type CapacityItem struct {
	Name    string  `json:"name"`
	CPUPct  float64 `json:"cpuPct"`
	MemMB   float64 `json:"memMB"`
	DiskPct float64 `json:"diskPct"`
}

// only the app containers we care about (cadvisor reports many).
var capacityContainers = map[string]bool{
	"compose-orders-api-1":       true,
	"compose-payments-worker-1":  true,
	"compose-notification-svc-1": true,
	"compose-mock-gateway-1":     true,
	"compose-mock-receiver-1":    true,
	"compose-slo-bff-1":          true,
}

// mergeCapacity merges per-container CPU%, memory bytes, and disk-usage ratio
// vectors (keyed by the cadvisor `name` label) into CapacityItems.
func mergeCapacity(cpu, mem, disk []Sample) []CapacityItem {
	items := map[string]*CapacityItem{}
	get := func(name string) *CapacityItem {
		if items[name] == nil {
			items[name] = &CapacityItem{Name: name}
		}
		return items[name]
	}
	for _, s := range cpu {
		if n := s.Labels["name"]; n != "" {
			get(n).CPUPct = round2(s.Value * 100)
		}
	}
	for _, s := range mem {
		if n := s.Labels["name"]; n != "" {
			get(n).MemMB = round2(s.Value / (1024 * 1024))
		}
	}
	for _, s := range disk {
		if n := s.Labels["name"]; n != "" {
			get(n).DiskPct = round2(s.Value * 100)
		}
	}
	out := make([]CapacityItem, 0, len(items))
	for _, it := range items {
		if math.IsNaN(it.DiskPct) {
			it.DiskPct = 0
		}
		out = append(out, *it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func buildCapacity(ctx context.Context, p Prom) []CapacityItem {
	cpu, _ := p.QueryVector(ctx, `sum by (name) (rate(container_cpu_usage_seconds_total{name=~"compose-.+"}[2m]))`)
	mem, _ := p.QueryVector(ctx, `sum by (name) (container_memory_usage_bytes{name=~"compose-.+"})`)
	disk, _ := p.QueryVector(ctx, `max by (name) (container_fs_usage_bytes{name=~"compose-.+"} / container_fs_limit_bytes)`)
	all := mergeCapacity(cpu, mem, disk)
	out := make([]CapacityItem, 0, len(all))
	for _, it := range all {
		if capacityContainers[it.Name] {
			out = append(out, it)
		}
	}
	return out
}
