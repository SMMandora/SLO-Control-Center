#!/usr/bin/env bash
# Scenario: inject 900ms latency into the mock payment gateway.
# Expected outcome: payments-worker end-to-end p95 latency breaches ~0.8s
# (the incident is visible on the SLO Overview / Incident dashboards). Sustained,
# this erodes the payments error budget and trends toward a burn alert.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

echo "[payments-latency] injecting LATENCY_MS=900 into mock-gateway"
GATEWAY_LATENCY_MS=900 $COMPOSE up -d --no-deps mock-gateway >/dev/null

echo "[payments-latency] expecting payments p95 latency to rise above 0.8s"
wait_for_metric 'slo:payments_worker:latency_p95_seconds' 0.8 180
rc=$?

echo "[payments-latency] reverting mock-gateway to LATENCY_MS=0"
GATEWAY_LATENCY_MS=0 $COMPOSE up -d --no-deps mock-gateway >/dev/null
exit $rc
