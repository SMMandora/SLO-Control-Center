package main

import (
	"log/slog"
	"os"
)

// logger emits structured JSON so promtail can extract trace_id for correlation.
var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
