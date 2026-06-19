#!/usr/bin/env bash
# Shared helpers for chaos scenarios: perturb the stack, then poll Prometheus to
# verify the expected outcome (an alert firing or a metric crossing a threshold).
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE="docker compose -f $HERE/../deploy/compose/docker-compose.yml"
PROM="${PROM:-http://localhost:9091}"

# wait_for_alert <alertname> <state> <timeout_sec> [label=value]
wait_for_alert() {
  local name="$1" state="${2:-firing}" timeout="${3:-300}" lm="${4:-}"
  local deadline=$(( SECONDS + timeout ))
  echo "  … waiting up to ${timeout}s for alert '$name' to be $state ${lm:+($lm)}"
  while (( SECONDS < deadline )); do
    if curl -s "$PROM/api/v1/alerts" | NAME="$name" STATE="$state" LM="$lm" python -c "
import os,sys,json
d=json.load(sys.stdin)
name,state,lm=os.environ['NAME'],os.environ['STATE'],os.environ['LM']
for a in d.get('data',{}).get('alerts',[]):
    if a.get('labels',{}).get('alertname')==name and a.get('state')==state:
        if not lm: sys.exit(0)
        k,v=lm.split('=',1)
        if a.get('labels',{}).get(k)==v: sys.exit(0)
sys.exit(1)
"; then
      echo "  ✓ PASS: alert '$name' is $state ${lm:+($lm)}"
      return 0
    fi
    sleep 5
  done
  echo "  ✗ FAIL: alert '$name' did not reach $state within ${timeout}s"
  return 1
}

# wait_for_metric <promql> <gt_threshold> <timeout_sec>
wait_for_metric() {
  local q="$1" thr="$2" timeout="${3:-180}" val=""
  local deadline=$(( SECONDS + timeout ))
  echo "  … waiting up to ${timeout}s for '$q' > $thr"
  while (( SECONDS < deadline )); do
    val=$(curl -s -G "$PROM/api/v1/query" --data-urlencode "query=$q" \
      | python -c "import sys,json; r=json.load(sys.stdin)['data']['result']; print(r[0]['value'][1] if r else 'nan')")
    if python -c "import sys; v='$val'; sys.exit(0 if v not in ('nan','') and float(v) > $thr else 1)" 2>/dev/null; then
      echo "  ✓ PASS: '$q' = $val > $thr"
      return 0
    fi
    sleep 5
  done
  echo "  ✗ FAIL: '$q' did not exceed $thr within ${timeout}s (last=$val)"
  return 1
}
