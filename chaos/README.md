# Chaos Scenarios

Each script perturbs the running stack, then **polls Prometheus to auto-verify**
the expected outcome, then reverts. Run them against a live stack
(`docker compose ... up -d`).

Requires `bash`, `curl`, and `python` on PATH. From the repo root:

```bash
bash chaos/error-burst.sh
bash chaos/service-down.sh
bash chaos/payments-latency.sh
```

| Scenario | Perturbation | Expected outcome (auto-verified) | Typical time |
|----------|--------------|----------------------------------|--------------|
| `error-burst.sh` | `CHAOS_ERROR_RATE=0.8` on orders-api | **FastBurn** (orders-api) alert fires | ~3–7 min |
| `service-down.sh` | `stop orders-api` | **ServiceDown** alert fires | ~2–3 min |
| `payments-latency.sh` | `LATENCY_MS=900` on mock-gateway | payments **p95 latency > 0.8s** (incident visible on dashboards) | ~1–2 min |

Notes:
- Burn-rate alerts use a 1h window, so `error-burst` needs a few minutes for the
  windowed error fraction to cross the threshold and then hold for `for: 2m`.
- `payments-latency` asserts the latency **metric** (deterministic and fast);
  sustained, the same latency erodes the payments budget toward a burn alert.
- The fourth spec scenario — **fill disk to 95% → DiskSpaceLow** — ships as an
  alert rule + runbook + promtool test, but its live run is deferred to the K8s
  sub-project (#6), where pod ephemeral-storage limits make it deterministic.
  In Docker Compose, containers share the host filesystem so a per-container
  fill is contrived.

Each script exits non-zero if the expected outcome does not occur within its
timeout, so they double as smoke tests.
