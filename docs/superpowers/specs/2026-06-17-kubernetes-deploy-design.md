# Design: Sub-project #6 — Kubernetes / Kustomize + Deploy + Demo

**Date:** 2026-06-17
**Status:** Approved (design)
**Sub-project:** #6 (final) of the SLO Control Center / Reference Observability Stack

## Goal

Run the whole stack on Kubernetes via Kustomize, deployable with `make deploy` to
a local **kind** cluster (staging-like), plus the deferred live disk-fill chaos
scenario and a scripted demo walkthrough.

## Locked decisions

- **kind** for the cluster (installed via `go install`; scriptable, no registry).
- Images built locally + `kind load`-ed; `imagePullPolicy: IfNotPresent`.
- **Demo = `docs/DEMO.md`** scripted walkthrough (no video capability).
- Configs reused from the existing files via Kustomize `configMapGenerator`
  (single source of truth); only **promtail needs a K8s-specific config**
  (`kubernetes_sd` tailing `/var/log/pods` instead of the Docker socket).

## Layout — `deploy/k8s/`

```
base/
  namespace.yaml
  apps.yaml          orders-api, payments-worker, notification-svc, mock-gateway, mock-receiver
  data.yaml          postgres, redis (emptyDir; not production-hardened)
  bff-frontend.yaml  slo-bff, frontend
  monitoring.yaml    prometheus, alertmanager, alert-receiver, grafana
  telemetry.yaml     tempo, loki, otel-collector, promtail (DaemonSet), cadvisor (DaemonSet)
  load.yaml          k6 Deployment
  kustomization.yaml resources + configMapGenerators (point at ../../observability/** + docs/runbooks)
overlays/staging/
  kustomization.yaml replicas, resource requests/limits, NodePort for frontend+grafana
  resources.yaml     patches
```

Each app service = Deployment (1 replica) + ClusterIP Service. Env mirrors the
compose env (DNS names become the K8s Service names within namespace `slo`).
prometheus/grafana/alertmanager/tempo/loki/otel mount their config from generated
ConfigMaps. cadvisor + promtail run as DaemonSets with the needed hostPath mounts.

## make targets

- `make k8s-up` — create the kind cluster (`deploy/k8s/kind-cluster.yaml`).
- `make k8s-images` — build all service images as `slo/<svc>:dev` + `kind load`.
- `make deploy` — k8s-images then `kubectl apply -k deploy/k8s/overlays/staging`.
- `make k8s-down` — delete the cluster.
- Access: `kubectl -n slo port-forward svc/frontend 3000:80` and `svc/grafana 3001:3000`.

## Disk-fill chaos — `chaos/disk-fill-k8s.sh`

orders-api Deployment gets `resources.limits.ephemeral-storage`. The script execs
a writer into a filesystem the cadvisor DaemonSet observes, fills it >90%, polls
the existing **DiskSpaceLow** rule until firing, then cleans up. (Same alert rule
as compose; cadvisor provides `container_fs_*` in-cluster.)

## Demo — `docs/DEMO.md`

A ~5-minute scripted walkthrough: bring up the stack → trigger a chaos scenario →
watch the alert fire in Alertmanager/UI → follow error-budget burn on the SLO
Overview → click a violation → trace → correlated logs → root cause. Lists every
URL and command.

## Verification

- `kubectl kustomize deploy/k8s/overlays/staging` builds cleanly (offline check).
- Live: deploy to kind; the **core SLO path** (apps + postgres/redis + prometheus
  + alertmanager + alert-receiver + slo-bff + frontend + grafana) reaches
  `Running`; frontend + Grafana reachable via port-forward; `/api/slo` returns
  data; an alert routes to alert-receiver. Telemetry components (tempo/loki/otel/
  promtail/cadvisor) are included in manifests; any not green under local
  resources are reported explicitly.

## Out of scope (still v1 non-goals)

TLS, secrets management, RBAC hardening, PersistentVolumes/StatefulSets,
multi-node/HA, an external image registry, ingress controllers.
