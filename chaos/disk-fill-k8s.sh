#!/usr/bin/env bash
# Scenario (Kubernetes): a pod exceeds its ephemeral-storage / emptyDir limit.
# Expected outcome: the kubelet evicts the pod (local storage capacity isolation)
# — the K8s-native equivalent of the DiskSpaceLow signal. Requires the kind
# cluster from `make k8s-up` + `make deploy`.
set -uo pipefail
NS=slo

echo "[disk-fill-k8s] creating disk-filler pod (emptyDir sizeLimit 64Mi)"
kubectl -n "$NS" delete pod disk-filler --ignore-not-found >/dev/null 2>&1
kubectl -n "$NS" apply -f - <<'YAML' >/dev/null
apiVersion: v1
kind: Pod
metadata:
  name: disk-filler
spec:
  restartPolicy: Never
  containers:
    - name: filler
      image: busybox:1.36
      command: ["sh", "-c", "dd if=/dev/zero of=/data/fill bs=1M count=256; sleep 600"]
      resources:
        limits:
          ephemeral-storage: "64Mi"
      volumeMounts:
        - { name: scratch, mountPath: /data }
  volumes:
    - name: scratch
      emptyDir:
        sizeLimit: "64Mi"
YAML

echo "[disk-fill-k8s] writing 256Mi into a 64Mi-limited volume; expecting eviction…"
deadline=$(( SECONDS + 150 ))
while (( SECONDS < deadline )); do
  phase=$(kubectl -n "$NS" get pod disk-filler -o jsonpath='{.status.phase}' 2>/dev/null)
  reason=$(kubectl -n "$NS" get pod disk-filler -o jsonpath='{.status.reason}' 2>/dev/null)
  if [ "$phase" = "Failed" ] || [ "$reason" = "Evicted" ]; then
    echo "  ✓ PASS: disk-filler was evicted/failed (reason=${reason:-$phase})"
    kubectl -n "$NS" describe pod disk-filler 2>/dev/null | grep -iE "evict|ephemeral|exceed" | head -3
    kubectl -n "$NS" delete pod disk-filler --ignore-not-found >/dev/null 2>&1
    exit 0
  fi
  sleep 5
done
echo "  ✗ FAIL: no eviction within timeout"
kubectl -n "$NS" delete pod disk-filler --ignore-not-found >/dev/null 2>&1
exit 1
