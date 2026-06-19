package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	cfg := LoadConfig()
	shutdown := initTracer(context.Background())
	defer shutdown()

	store, err := NewStore(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	var pub Publisher
	if p, err := NewRedisPublisher(cfg.RedisURL); err != nil {
		log.Printf("redis disabled: %v", err)
	} else {
		pub = p
	}
	srv := NewServer(store, cfg.Chaos, rand.Float64, pub)
	handler := otelhttp.NewHandler(srv, "orders-api")
	log.Printf("orders-api listening on %s", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, handler))
}
