# 5-Minute Demo Walkthrough

A scripted tour of the SLO Control Center: trigger an incident, watch the alert
fire, follow the error-budget burn, and chase a failure from metric → trace →
log → root cause.

> This is the reproducible substitute for the demo video. Run it against the
> Docker Compose stack (fastest) or the Kubernetes deploy.

## 0. Bring it up (Docker Compose)

```bash
docker compose -f deploy/compose/docker-compose.yml up --build -d
```

Wait ~90s, then open:
- **SLO Control Center** → http://localhost:3000
- **Grafana** → http://localhost:3001
- **Alertmanager** → http://localhost:9093

## 1. The healthy baseline (30s)

On the **Overview** tab: three services, error budgets, the 28-day compliance
graph, and the live health map. Note `payments-worker` is already *breaching* its
99.5% target — the mock gateway flakes ~8% deterministically, so the SLO is
correctly catching a genuinely-degraded dependency. Open **Services** to see the
RED metrics + dependencies per service.

## 2. Trigger an incident (1 min)

```bash
bash chaos/error-burst.sh        # CHAOS_ERROR_RATE=0.8 on orders-api
```

The script injects errors and polls until **FastBurn (orders-api)** fires.
Meanwhile, watch:
- **Overview** → orders-api availability drops, 1h burn rate spikes, error budget
  falls.
- **Alerts** tab (and http://localhost:9093) → `FastBurn` appears, routed to the
  alert-receiver: `curl localhost:8092/alerts`.
- **Incidents** tab → a new incident derived from the firing alert.

## 3. Follow the burn → trace → log (2 min)

A failed payment is a **single distributed trace** across
orders-api → payments-worker → mock-gateway (trace context rides the Redis
streams).

1. **Overview** → *Recent Violations → Trace*, or the **Traces** tab → pick an
   error trace → **open in Grafana** for the full waterfall (see the failing
   `mock-gateway` span).
2. In Grafana the trace links to its **logs** (Tempo → Loki by `trace_id`); or use
   the **Logs** tab filtered to `error` — each line carries the `trace_id`.
3. Open the **Incident Investigation** dashboard
   (http://localhost:3001/d/incident-investigation): error-rate + burn-rate +
   correlated error logs in one place.

## 4. Open the runbook (30s)

The `FastBurn` alert's `runbook_url` → **Runbooks** tab → *Fast Burn* page:
symptoms, likely causes, first checks (PromQL), mitigation. The mitigation here:
stop the chaos. `error-burst.sh` already reverts `CHAOS_ERROR_RATE` to 0 on exit.

## 5. Other scenarios

```bash
bash chaos/service-down.sh       # stop orders-api -> ServiceDown alert
bash chaos/payments-latency.sh   # gateway LATENCY_MS=900 -> payments p95 breach
```

## Kubernetes variant

```bash
make k8s-up        # create the kind cluster
make deploy        # build + load images, apply the staging overlay
kubectl -n slo get pods
kubectl -n slo port-forward svc/frontend 3000:80    # UI at localhost:3000
kubectl -n slo port-forward svc/slo-bff 9090:9090   # UI's BFF
# Grafana is exposed via NodePort at http://localhost:30001
bash chaos/disk-fill-k8s.sh      # ephemeral-storage limit -> pod eviction
```

## Tear down

```bash
docker compose -f deploy/compose/docker-compose.yml down -v
make k8s-down
```
