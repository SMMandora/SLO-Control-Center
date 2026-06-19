package main

import "testing"

func TestMergeCapacity(t *testing.T) {
	cpu := []Sample{{Labels: map[string]string{"name": "compose-orders-api-1"}, Value: 0.05}}
	mem := []Sample{{Labels: map[string]string{"name": "compose-orders-api-1"}, Value: 52428800}} // 50 MB
	disk := []Sample{{Labels: map[string]string{"name": "compose-orders-api-1"}, Value: 0.42}}
	items := mergeCapacity(cpu, mem, disk)
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	it := items[0]
	if it.Name != "compose-orders-api-1" {
		t.Fatalf("name %q", it.Name)
	}
	if it.CPUPct != 5 {
		t.Fatalf("cpu %v", it.CPUPct)
	}
	if it.MemMB != 50 {
		t.Fatalf("mem %v", it.MemMB)
	}
	if it.DiskPct != 42 {
		t.Fatalf("disk %v", it.DiskPct)
	}
}
