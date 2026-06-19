#!/usr/bin/env bash
# Scenario: inject a high error rate into orders-api.
# Expected outcome: FastBurn (orders-api) alert fires (page).
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

echo "[error-burst] injecting CHAOS_ERROR_RATE=0.8 into orders-api"
CHAOS_ERROR_RATE=0.8 $COMPOSE up -d --no-deps orders-api >/dev/null

echo "[error-burst] the 1h burn-rate window must cross 13.44 (≈2% budget/h), then hold for 2m"
wait_for_alert FastBurn firing 420 service=orders-api
rc=$?

echo "[error-burst] reverting orders-api to CHAOS_ERROR_RATE=0"
CHAOS_ERROR_RATE=0 $COMPOSE up -d --no-deps orders-api >/dev/null
exit $rc
