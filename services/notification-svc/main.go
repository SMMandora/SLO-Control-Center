package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var (
	delivered = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "notifications_delivered_total", Help: "Notification deliveries.",
	}, []string{"status"})

	deliveryLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "notification_delivery_latency_seconds", Help: "Emit-to-delivered latency.",
		Buckets: []float64{.1, .25, .5, 1, 2, 5, 10},
	}, []string{"status"})
)

const group = "notification-svcs"

func deliver(ctx context.Context, client *http.Client, receiver, orderID string) bool {
	for i := 0; i < 3; i++ {
		body := bytes.NewBufferString(`{"order_id":"` + orderID + `"}`)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, receiver+"/webhook", body)
		if err != nil {
			return false
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode/100 == 2 {
				return true
			}
		}
	}
	return false
}

func process(ctx context.Context, rdb *redis.Client, client *http.Client, receiver, consumer string) {
	res, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: group, Consumer: consumer,
		Streams: []string{"notifications", ">"}, Count: 10, Block: 5 * time.Second,
	}).Result()
	if err != nil && err != redis.Nil {
		return
	}
	for _, stream := range res {
		for _, msg := range stream.Messages {
			orderID, _ := msg.Values["order_id"].(string)
			emittedMs, _ := strconv.ParseInt(toStr(msg.Values["emitted_at"]), 10, 64)
			// Continue the distributed trace started in orders-api / payments-worker.
			mctx := extractTrace(ctx, msg.Values)
			mctx, span := otel.Tracer("notification-svc").Start(mctx, "deliver-notification")
			ok := deliver(mctx, client, receiver, orderID)
			doneMs := time.Now().UnixMilli()
			status, latency := classify(emittedMs, doneMs, ok)
			delivered.WithLabelValues(status).Inc()
			deliveryLatency.WithLabelValues(status).Observe(latency)
			if !ok {
				logger.Error("notification delivery failed", "order_id", orderID, "trace_id", traceID(mctx))
			}
			span.End()
			rdb.XAck(ctx, "notifications", group, msg.ID)
		}
	}
}

func toStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "0"
}

func main() {
	redisURL := envStr("REDIS_URL", "redis://redis:6379")
	receiver := envStr("RECEIVER_URL", "http://mock-receiver:8091")
	addr := envStr("ADDR", ":8081")

	shutdown := initTracer(context.Background())
	defer shutdown()

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}
	rdb := redis.NewClient(opt)
	ctx := context.Background()
	if err := rdb.XGroupCreateMkStream(ctx, "notifications", group, "0").Err(); err != nil &&
		err.Error() != "BUSYGROUP Consumer Group name already exists" {
		log.Printf("xgroup create: %v", err)
	}

	consumer := envStr("HOSTNAME", "notifier-1")
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	go func() {
		for {
			process(ctx, rdb, client, receiver, consumer)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	log.Printf("notification-svc listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envStr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
