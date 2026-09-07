#!/bin/bash
set -e

# ===========================================
# RINCO Kubernetes Deployment Script
# ===========================================

echo "🚀 Deploying to Kubernetes..."

# Navigate to project root
cd "$(dirname "$0")/.."

# Check prerequisites
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed"
    exit 1
fi

if ! command -v helm &> /dev/null; then
    echo "❌ helm is not installed"
    exit 1
fi

# Environment variables
NAMESPACE=${NAMESPACE:-production}
RELEASE_NAME=${RELEASE_NAME:-rinco}
CHART_PATH=${CHART_PATH:-./deployments/helm/rinco}
VALUES_FILE=${VALUES_FILE:-./deployments/helm/rinco/values.prod.yaml}

# Get version from git tag
VERSION=${VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo "latest")}

# Set image tag
export IMAGE_TAG=$VERSION

echo "Deploying RINCO to namespace: $NAMESPACE"
echo "Release: $RELEASE_NAME"
echo "Version: $VERSION"

# Create namespace if not exists
echo "📦 Creating namespace..."
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Apply secrets (if exists)
if [ -f "deployments/k8s/secrets.yaml" ]; then
    echo "🔐 Applying secrets..."
    kubectl apply -f deployments/k8s/secrets.yaml -n $NAMESPACE
fi

# Apply ConfigMaps
if [ -f "deployments/k8s/configmaps.yaml" ]; then
    echo "⚙️ Applying ConfigMaps..."
    kubectl apply -f deployments/k8s/configmaps.yaml -n $NAMESPACE
fi

# Deploy with Helm
echo "🚀 Deploying with Helm..."
helm upgrade --install $RELEASE_NAME $CHART_PATH \
    --namespace $NAMESPACE \
    --values $VALUES_FILE \
    --set image.tag=$VERSION \
    --wait \
    --timeout 10m

# Wait for rollout
echo "⏳ Waiting for rollout..."
kubectl rollout status deployment/auth-service -n $NAMESPACE --timeout=5m
kubectl rollout status deployment/tenant-service -n $NAMESPACE --timeout=5m
kubectl rollout status deployment/lead-service -n $NAMESPACE --timeout=5m
kubectl rollout status deployment/crm-service -n $NAMESPACE --timeout=5m
kubectl rollout status deployment/landing-service -n $NAMESPACE --timeout=5m
kubectl rollout status deployment/landing -n $NAMESPACE --timeout=5m
kubectl rollout status deployment/admin-portal -n $NAMESPACE --timeout=5m

# Show deployment status
echo ""
echo "✅ Deployment completed!"
echo ""
kubectl get pods -n $NAMESPACE
