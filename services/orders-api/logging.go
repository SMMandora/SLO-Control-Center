package main

import (
	"log/slog"
	"os"
)

// logger emits structured JSON logs so promtail can extract trace_id for
// log<->trace correlation in Grafana.
var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
