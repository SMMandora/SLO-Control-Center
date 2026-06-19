package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunbookTitle(t *testing.T) {
	if got := runbookTitle("# Runbook: FastBurn (page)\n\nbody", "fb"); got != "Runbook: FastBurn (page)" {
		t.Fatalf("title %q", got)
	}
	if got := runbookTitle("no heading here", "fallback.md"); got != "fallback.md" {
		t.Fatalf("fallback %q", got)
	}
}

func TestReadRunbooks(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "fast-burn.md"), []byte("# Fast Burn\nbody"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Index\nshould be skipped"), 0o644)
	rbs := readRunbooks(dir)
	if len(rbs) != 1 {
		t.Fatalf("want 1 runbook (README skipped), got %d", len(rbs))
	}
	if rbs[0].Name != "fast-burn" || rbs[0].Title != "Fast Burn" {
		t.Fatalf("bad runbook: %+v", rbs[0])
	}
}
