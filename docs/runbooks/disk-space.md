# Runbook: DiskSpaceLow (warn)

**Alert:** a container filesystem is >90% full
(`max by (name)(container_fs_usage_bytes / container_fs_limit_bytes) > 0.9`) for
5 minutes. Metrics come from cadvisor.

## Symptoms
- Warning for `DiskSpaceLow` on a container `name`.
- Possible downstream failures: writes failing, Prometheus/Tempo/Loki unable to
  persist, Postgres erroring on insert.

## Likely causes
- Unbounded log or data growth (Prometheus TSDB, Tempo blocks, Loki chunks,
  Postgres).
- A volume too small for retention settings.
- A runaway process writing temp files.

## First checks
- Which container and how full:
  ```promql
  max by (name) (container_fs_usage_bytes / container_fs_limit_bytes)
  ```
- Inspect docker disk usage:
  ```bash
  docker system df
  docker compose -f deploy/compose/docker-compose.yml ps
  ```

## Mitigation
- Reclaim space: prune unused images/volumes (`docker system prune`), or reduce
  retention (`--storage.tsdb.retention.time` for Prometheus, block retention for
  Tempo/Loki).
- Grow the volume if persistently tight.
- For the data stores, confirm retention/compaction is running.

> **Note (v1):** in Docker Compose this alert is best-effort (containers share the
> host filesystem). The live "fill disk to 95%" chaos scenario lands in the K8s
> sub-project (#6), where pod ephemeral-storage limits make it deterministic.

## Related dashboards
- Capacity / USE (built in a later sub-project from the cadvisor metrics added here).
