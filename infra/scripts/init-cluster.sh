#!/usr/bin/env bash
# ============================================================
# K3s cluster bootstrap helper (RINCO)
# ============================================================
# Boots a k3s server in-place and applies the base manifests.
# For multi-node HA, see deployments/k3s/ha/README.md.
# ============================================================

set -euo pipefail
PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

echo "==> Installing prerequisites"
command -v kubectl >/dev/null || (curl -sfL https://dl.k8s.io/release/v1.30.0/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl && chmod +x /usr/local/bin/kubectl)
command -v helm >/dev/null    || (curl -sfL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash)
command -v kustomize >/dev/null || (curl -sfL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v5.4.3/kustomize_v5.4.3_linux_amd64 -o /usr/local/bin/kustomize && chmod +x /usr/local/bin/kustomize)

echo "==> Bootstrap K3s server"
bash "${PROJECT_ROOT}/deployments/k3s/setup-k3s.sh"

echo "==> Apply base manifests"
kubectl apply -k "${PROJECT_ROOT}/deployments/k3s/base"

echo "==> Waiting for all pods ready"
kubectl wait --for=condition=Ready pods --all -n kube-system --timeout=180s

echo "✓ Cluster ready."
echo "    KUBECONFIG=/etc/rancher/k3s/k3s.yaml"
