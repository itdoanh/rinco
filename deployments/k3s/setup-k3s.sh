#!/usr/bin/env bash
# ============================================================
# K3s cluster bootstrap (RINCO)
# ============================================================
# Single-server mode for dev/staging; for HA use the offline
# multi-node install + an external PostgreSQL datastore.
# ============================================================

set -euo pipefail

: "${K3S_VERSION:=v1.30.4+k3s1}"
: "${RINCO_DOMAIN:=rinco.local}"
: "${ARGO_NS:=argocd}"
: "${CERTMGR_NS:=cert-manager}"
: "${TRAEFIK_ENABLED:=true}"
: "${INSTALL_LONGHORN:=true}"
: "${EXTERNAL_DB_DSN:=}"

info() { printf '\033[1;34m[i]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[!]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[x]\033[0m %s\n' "$*" && exit 1; }

info "Installing K3s ${K3S_VERSION}"
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="${K3S_VERSION}" sh -s - \
  --disable=traefik \
  --disable=servicelb \
  --write-kubeconfig-mode=0644 \
  --disable-cloud-controller \
  --prefer-bundled-binaries \
  --node-label "rinco.io/role=control-plane" \
  --node-label "rinco.io/region=ap-southeast-1"

mkdir -p "$HOME/.kube"
[ ! -f "$HOME/.kube/config" ] && cp /etc/rancher/k3s/k3s.yaml "$HOME/.kube/config" && chmod 600 "$HOME/.kube/config"
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

info "Waiting for nodes to be Ready"
kubectl wait --for=condition=Ready node --all --timeout=180s

# ============================================================
# Traefik ingress
# ============================================================
if [ "${TRAEFIK_ENABLED}" = "true" ]; then
  info "Installing Traefik (Helm)"
  helm repo add traefik https://traefik.github.io/charts 2>/dev/null || true
  helm repo update
  helm upgrade --install traefik traefik/traefik \
    --namespace traefik --create-namespace \
    --version 'v25.0.0' \
    --set ports.web.port=80 \
    --set ports.websecure.port=443 \
    --set ports.traefik.port=8080 \
    --set ports.otlp.port=4317 \
    --set ports.otlp.expose.port=4317 \
    --set ports.otlp.exposedPort=4317 \
    --set metrics.prometheus.enabled=true \
    --values "${SCRIPT_DIR:-.}/values/traefik-values.yaml" || \
      warn "Traefik already installed"
fi

# ============================================================
# cert-manager (production)
# ============================================================
info "Installing cert-manager"
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.1/cert-manager.yaml || true
kubectl -n "${CERTMGR_NS}" wait --for=condition=Available --timeout=180s deploy/cert-manager-webhook || true

info "Applying staging/issuers/ClusterIssuer"
kubectl apply -f deploy/k3s/conftest/  || true
kubectl apply -f deploy/k3s/cert-issuers.yaml  || true

# ============================================================
# ArgoCD (GitOps)
# ============================================================
info "Installing ArgoCD"
kubectl create namespace "${ARGO_NS}" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n "${ARGO_NS}" -f https://raw.githubusercontent.com/argoproj/argo-cd/v2.11.0/manifests/install.yaml
kubectl -n "${ARGO_NS}" wait --for=condition=Available --timeout=180s deploy/argocd-server || true

# ============================================================
# Longhorn distributed storage
# ============================================================
if [ "${INSTALL_LONGHORN}" = "true" ]; then
  info "Installing Longhorn"
  helm repo add longhorn https://charts.longhorn.io 2>/dev/null || true
  helm repo update
  helm upgrade --install longhorn longhorn/longhorn \
    --namespace longhorn-system --create-namespace \
    --version '1.6.3' \
    --set defaultSettings.defaultDataPath="/var/lib/longhorn" \
    --set persistence.defaultClass=true || warn "Longhorn install issue (continuing)"
fi

# ============================================================
# Sealed Secrets (so we don't commit plaintext)
# ============================================================
info "Installing Sealed Secrets"
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.27.1/controller.yaml || true

info "✓ K3s bootstrap complete"
info "    kubeconfig: /etc/rancher/k3s/k3s.yaml"
info "    ArgoCD UI : kubectl port-forward svc/argocd-server -n ${ARGO_NS} 8080:443"
