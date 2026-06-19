package main

import (
	"os"
	"strconv"
)

// Config is the runtime configuration assembled from environment variables.
type Config struct {
	Addr        string
	DatabaseURL string
	RedisURL    string
	Chaos       Chaos
}

func envFloat(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envStr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// LoadConfig reads configuration from the environment, applying defaults.
func LoadConfig() Config {
	return Config{
		Addr:        envStr("ADDR", ":8080"),
		DatabaseURL: envStr("DATABASE_URL", "postgres://orders:orders@postgres:5432/orders?sslmode=disable"),
		RedisURL:    envStr("REDIS_URL", "redis://redis:6379"),
		Chaos: Chaos{
			ErrorRate:  envFloat("CHAOS_ERROR_RATE", 0),
			LatencyMS:  envInt("CHAOS_LATENCY_MS", 0),
			LatencyPct: envFloat("CHAOS_LATENCY_PCT", 0),
		},
	}
}
