# Runbook: FastBurn (page)

**Alert:** error-budget burning faster than ~2% of the 28-day budget per hour
(`slo:<service>:burnrate_1h > 13.44`) for 2 minutes.

## Symptoms
- Paged for `FastBurn` on a specific `service`.
- SLO Overview shows the service's burn rate (1h) spiking and error budget
  dropping quickly.

## Likely causes
- A bad deploy or config change increasing the error rate.
- A failing dependency (e.g. `mock-gateway` declining charges, DB/Redis outage).
- Injected chaos (`CHAOS_ERROR_RATE` on orders-api) — confirm no chaos test is
  running.

## First checks
- Current SLI / burn rate:
  ```promql
  slo:orders_api:availability:ratio_5m
  slo:orders_api:burnrate_1h
  ```
- Where the errors are (by route/status):
  ```promql
  sum by (route, status) (rate(http_requests_total{job="orders-api",status=~"5.."}[5m]))
  ```
- Recent error traces: SLO Control Center → **Recent Violations → Trace**, or
  `GET http://localhost:9090/api/traces/recent`.
- Error logs with trace IDs: Incident Investigation dashboard (Loki panel).

## Mitigation
- If a recent deploy is the cause, **roll back**.
- If a dependency is failing, fail over / disable the dependency path; for the
  demo, check `mock-gateway` (`gateway_charges_total{result="declined"}`).
- If chaos-induced, stop the chaos: recreate the service with `CHAOS_ERROR_RATE=0`.

## Related dashboards
- SLO Overview, Incident Investigation.
