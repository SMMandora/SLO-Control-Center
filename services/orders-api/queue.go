package main

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Publisher enqueues a payment job for a created order.
type Publisher interface {
	EnqueuePayment(ctx context.Context, orderID int64, item string, qty int) error
}

// RedisPublisher writes payment jobs to the "payments" Redis stream.
type RedisPublisher struct{ rdb *redis.Client }

// NewRedisPublisher connects to Redis using a redis:// URL.
func NewRedisPublisher(url string) (*RedisPublisher, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return &RedisPublisher{rdb: redis.NewClient(opt)}, nil
}

// EnqueuePayment XADDs a job carrying the enqueue timestamp (unix millis) so the
// worker can measure end-to-end latency against the SLO deadline. The active
// trace context is injected so the worker continues the same distributed trace.
func (p *RedisPublisher) EnqueuePayment(ctx context.Context, orderID int64, item string, qty int) error {
	values := map[string]any{
		"order_id":    strconv.FormatInt(orderID, 10),
		"item":        item,
		"qty":         qty,
		"enqueued_at": time.Now().UnixMilli(),
	}
	for k, v := range injectTrace(ctx) {
		values[k] = v
	}
	return p.rdb.XAdd(ctx, &redis.XAddArgs{Stream: "payments", Values: values}).Err()
}
