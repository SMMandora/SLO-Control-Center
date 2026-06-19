#!/usr/bin/env bash
# Scenario: take orders-api down.
# Expected outcome: ServiceDown alert fires (page) after up==0 for 2m.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

echo "[service-down] stopping orders-api"
$COMPOSE stop orders-api >/dev/null

echo "[service-down] expecting ServiceDown after its scrape fails for 2m"
wait_for_alert ServiceDown firing 240 job=orders-api
rc=$?

echo "[service-down] restarting orders-api"
$COMPOSE start orders-api >/dev/null
exit $rc
