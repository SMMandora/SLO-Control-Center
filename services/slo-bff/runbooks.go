package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Runbook is a rendered runbook document.
type Runbook struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
}

// runbookTitle extracts the first markdown H1, falling back to the filename.
func runbookTitle(md, fallback string) string {
	for _, line := range strings.Split(md, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(t, "# "))
		}
	}
	return fallback
}

// readRunbooks loads every *.md file in dir (excluding the index README).
func readRunbooks(dir string) []Runbook {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []Runbook{}
	}
	out := []Runbook{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || name == "README.md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		md := string(data)
		out = append(out, Runbook{
			Name:     strings.TrimSuffix(name, ".md"),
			Title:    runbookTitle(md, name),
			Markdown: md,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
