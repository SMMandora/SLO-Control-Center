# Runbook: SlowBurn (ticket)

**Alert:** error-budget burning faster than ~10% of the 28-day budget over 6h
(`slo:<service>:burnrate_6h > 11.2`) for 15 minutes.

## Symptoms
- A ticket (not a page) for `SlowBurn` on a `service`.
- Steady, lower-grade budget erosion than `FastBurn` — often elevated latency
  rather than hard failures.

## Likely causes
- Latency regression pushing requests past the SLO deadline (e.g. payments not
  completing within 60s, notifications past 5s).
- A slow dependency (e.g. `mock-gateway` `LATENCY_MS` injected).
- Partial/intermittent errors below the fast-burn threshold.

## First checks
- p95 latency vs target:
  ```promql
  slo:payments_worker:latency_p95_seconds
  slo:orders_api:latency_p95_seconds
  ```
- Multi-window burn (is it accelerating?):
  ```promql
  slo:payments_worker:burnrate_1h
  slo:payments_worker:burnrate_6h
  ```
- Slow traces in Tempo (Incident Investigation → Tempo / Explore).

## Mitigation
- Identify the slow span in a trace (orders-api → payments-worker → mock-gateway)
  and address the slow hop.
- For the demo: if `mock-gateway` latency was injected, recreate it with
  `LATENCY_MS=0`.
- File/track the ticket; no immediate page-level action required unless it
  escalates to `FastBurn`.

## Related dashboards
- SLO Overview, Incident Investigation.
