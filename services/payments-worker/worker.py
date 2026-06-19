"""payments-worker: consumes the Redis 'payments' stream, charges the mock
gateway, emits a 'notifications' event on success, and exports SLO metrics +
distributed traces (continuing the trace started in orders-api)."""

import json
import os
import sys
import time

import redis
import requests
from opentelemetry import trace
from opentelemetry.instrumentation.requests import RequestsInstrumentor
from opentelemetry.trace import Status, StatusCode
from prometheus_client import Counter, Histogram, start_http_server

from classify import classify
from tracing import current_trace_id, extract_context, init_tracer, inject_context

PROCESSED = Counter("payments_processed_total", "Payments processed.", ["status"])
LATENCY = Histogram(
    "payments_e2e_latency_seconds",
    "Enqueue-to-done latency.",
    ["status"],
    buckets=(0.5, 1, 2, 5, 10, 30, 60, 120),
)

GATEWAY = os.getenv("GATEWAY_URL", "http://mock-gateway:8090")
REDIS_URL = os.getenv("REDIS_URL", "redis://redis:6379")
GROUP = "payments-workers"
CONSUMER = os.getenv("HOSTNAME", "worker-1")


def log_json(level: str, msg: str, **kw) -> None:
    print(json.dumps({"level": level, "msg": msg, **kw}), file=sys.stdout, flush=True)


def charge(order_id: str) -> bool:
    """POST /charge with up to 3 attempts; True on any 2xx. The instrumented
    requests client injects the active trace context into the call."""
    for _ in range(3):
        try:
            r = requests.post(f"{GATEWAY}/charge", json={"order_id": order_id}, timeout=5)
            if r.status_code // 100 == 2:
                return True
        except requests.RequestException:
            pass
    return False


def process(rdb, msg_id: str, fields: dict) -> None:
    order_id = fields.get("order_id", "")
    enqueued_ms = int(fields.get("enqueued_at", "0") or 0)
    parent = extract_context(fields)
    tracer = trace.get_tracer("payments-worker")
    with tracer.start_as_current_span("process-payment", context=parent) as span:
        ok = charge(order_id)
        done_ms = int(time.time() * 1000)
        status, latency = classify(enqueued_ms, done_ms, ok)
        PROCESSED.labels(status).inc()
        LATENCY.labels(status).observe(latency)
        if ok:
            # Re-inject the (continued) trace so notification-svc joins it too.
            event = {"order_id": order_id, "emitted_at": done_ms, **inject_context()}
            rdb.xadd("notifications", event)
        else:
            # Mark the span as an error so it surfaces as a "violation" in Tempo.
            span.set_status(Status(StatusCode.ERROR))
            log_json("error", "payment failed", order_id=order_id, trace_id=current_trace_id())
        rdb.xack("payments", GROUP, msg_id)


def main() -> None:
    init_tracer()
    RequestsInstrumentor().instrument()
    start_http_server(8000)
    rdb = redis.from_url(REDIS_URL, decode_responses=True)
    try:
        rdb.xgroup_create("payments", GROUP, id="0", mkstream=True)
    except redis.ResponseError as e:
        if "BUSYGROUP" not in str(e):
            raise
    while True:
        resp = rdb.xreadgroup(GROUP, CONSUMER, {"payments": ">"}, count=10, block=5000)
        for _stream, messages in resp or []:
            for msg_id, fields in messages:
                process(rdb, msg_id, fields)


if __name__ == "__main__":
    main()
