#!/bin/bash
set -e

# ===========================================
# RINCO Docker Build Script
# ===========================================

echo "🔨 Building RINCO Docker images..."

# Navigate to project root
cd "$(dirname "$0")/.."

# Set Docker registry (default to ghcr.io)
REGISTRY=${REGISTRY:-ghcr.io}
USERNAME=${USERNAME:-$(git config user.name | tr '[:upper:]' '[:lower:]' | tr -d ' ')}

# Extract version from git tag or use default
VERSION=${VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo "latest")}

echo "Building version: $VERSION"
echo "Registry: $REGISTRY/$USERNAME"

# Build Go services
echo "🐹 Building Go services..."

# Auth Service
if [ -d "services/auth-service" ]; then
    echo "Building auth-service..."
    cd services/auth-service
    docker build -t "$REGISTRY/$USERNAME/auth-service:$VERSION" \
                 -t "$REGISTRY/$USERNAME/auth-service:latest" .
    cd ../..
fi

# Tenant Service
if [ -d "services/tenant-service" ]; then
    echo "Building tenant-service..."
    cd services/tenant-service
    docker build -t "$REGISTRY/$USERNAME/tenant-service:$VERSION" \
                 -t "$REGISTRY/$USERNAME/tenant-service:latest" .
    cd ../..
fi

# Lead Service
if [ -d "services/lead-service" ]; then
    echo "Building lead-service..."
    cd services/lead-service
    docker build -t "$REGISTRY/$USERNAME/lead-service:$VERSION" \
                 -t "$REGISTRY/$USERNAME/lead-service:latest" .
    cd ../..
fi

# CRM Service
if [ -d "services/crm-service" ]; then
    echo "Building crm-service..."
    cd services/crm-service
    docker build -t "$REGISTRY/$USERNAME/crm-service:$VERSION" \
                 -t "$REGISTRY/$USERNAME/crm-service:latest" .
    cd ../..
fi

# Landing Service
if [ -d "services/landing-service" ]; then
    echo "Building landing-service..."
    cd services/landing-service
    docker build -t "$REGISTRY/$USERNAME/landing-service:$VERSION" \
                 -t "$REGISTRY/$USERNAME/landing-service:latest" .
    cd ../..
fi

# Build Rust services
echo "🦀 Building Rust services..."

# Chat Engine
if [ -d "services/chat-engine" ]; then
    echo "Building chat-engine..."
    cd services/chat-engine
    docker build -t "$REGISTRY/$USERNAME/chat-engine:$VERSION" \
                 -t "$REGISTRY/$USERNAME/chat-engine:latest" .
    cd ../..
fi

# WebRTC SFU
if [ -d "services/webrtc-sfu" ]; then
    echo "Building webrtc-sfu..."
    cd services/webrtc-sfu
    docker build -t "$REGISTRY/$USERNAME/webrtc-sfu:$VERSION" \
                 -t "$REGISTRY/$USERNAME/webrtc-sfu:latest" .
    cd ../..
fi

# Build Python services
echo "🐍 Building Python services..."

# Lead Scoring
if [ -d "services/lead-scoring" ]; then
    echo "Building lead-scoring..."
    cd services/lead-scoring
    docker build -t "$REGISTRY/$USERNAME/lead-scoring:$VERSION" \
                 -t "$REGISTRY/$USERNAME/lead-scoring:latest" .
    cd ../..
fi

# Build Frontend applications
echo "⚛️ Building Frontend applications..."

# Landing Page
if [ -d "frontend/landing" ]; then
    echo "Building landing..."
    cd frontend/landing
    docker build -t "$REGISTRY/$USERNAME/landing:$VERSION" \
                 -t "$REGISTRY/$USERNAME/landing:latest" .
    cd ../..
fi

# Admin Portal
if [ -d "frontend/admin-portal" ]; then
    echo "Building admin-portal..."
    cd frontend/admin-portal
    docker build -t "$REGISTRY/$USERNAME/admin-portal:$VERSION" \
                 -t "$REGISTRY/$USERNAME/admin-portal:latest" .
    cd ../..
fi

# Tenant Site
if [ -d "frontend/tenant-site" ]; then
    echo "Building tenant-site..."
    cd frontend/tenant-site
    docker build -t "$REGISTRY/$USERNAME/tenant-site:$VERSION" \
                 -t "$REGISTRY/$USERNAME/tenant-site:latest" .
    cd ../..
fi

# Meeting UI
if [ -d "frontend/meeting-ui" ]; then
    echo "Building meeting-ui..."
    cd frontend/meeting-ui
    docker build -t "$REGISTRY/$USERNAME/meeting-ui:$VERSION" \
                 -t "$REGISTRY/$USERNAME/meeting-ui:latest" .
    cd ../..
fi

echo ""
echo "✅ Build completed!"
echo "Images built:"
echo "  - $REGISTRY/$USERNAME/auth-service:$VERSION"
echo "  - $REGISTRY/$USERNAME/tenant-service:$VERSION"
echo "  - $REGISTRY/$USERNAME/lead-service:$VERSION"
echo "  - $REGISTRY/$USERNAME/crm-service:$VERSION"
echo "  - $REGISTRY/$USERNAME/landing-service:$VERSION"
echo "  - $REGISTRY/$USERNAME/chat-engine:$VERSION"
echo "  - $REGISTRY/$USERNAME/webrtc-sfu:$VERSION"
echo "  - $REGISTRY/$USERNAME/lead-scoring:$VERSION"
echo "  - $REGISTRY/$USERNAME/landing:$VERSION"
echo "  - $REGISTRY/$USERNAME/admin-portal:$VERSION"
echo "  - $REGISTRY/$USERNAME/tenant-site:$VERSION"
echo "  - $REGISTRY/$USERNAME/meeting-ui:$VERSION"
