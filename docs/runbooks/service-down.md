# Runbook: ServiceDown (page)

**Alert:** Prometheus scrape liveness failing (`up{job="<service>"} == 0`) for 2
minutes.

## Symptoms
- Paged for `ServiceDown` on a `job` (orders-api / payments-worker /
  notification-svc).
- Live Service Health Map node is red / missing; metrics for that service go flat.

## Likely causes
- The container crashed or was OOM-killed.
- A failed dependency at startup (Postgres/Redis not ready).
- A bad deploy that fails to bind its port or `/metrics` endpoint.

## First checks
- Container state and logs:
  ```bash
  docker compose -f deploy/compose/docker-compose.yml ps
  docker logs compose-<service>-1 --tail 50
  ```
- Confirm the scrape target is down in Prometheus → Status → Targets
  (http://localhost:9091/targets).
- Dependencies healthy?
  ```bash
  docker inspect -f '{{.State.Health.Status}}' compose-postgres-1
  docker inspect -f '{{.State.Health.Status}}' compose-redis-1
  ```

## Mitigation
- Restart the service:
  ```bash
  docker compose -f deploy/compose/docker-compose.yml up -d <service>
  ```
- If it crash-loops, read the logs for the panic/exit cause and fix config
  (DB/Redis URL, port, missing env).
- If a dependency is down, recover it first, then the service.

## Related dashboards
- SLO Overview (Services Healthy card), Incident Investigation.
