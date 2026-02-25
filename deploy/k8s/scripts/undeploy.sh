#!/bin/bash

set -e

# Undeployment script for Logos Platform from Kubernetes

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}INFO:${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}WARN:${NC} $1"
}

log_error() {
    echo -e "${RED}ERROR:${NC} $1" >&2
}

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    log_error "kubectl is not installed"
    exit 1
fi

# Check if helm is available
if ! command -v helm &> /dev/null; then
    log_error "helm is not installed"
    exit 1
fi

# Uninstall Logos Platform using Helm
namespace="logging-system"
if helm list -n "$namespace" | grep -q "logos"; then
    log_info "Uninstalling Logos Platform"
    helm uninstall logos -n "$namespace"
fi

# Uninstall infrastructure services
log_info "Uninstalling infrastructure services"
for release in etcd kafka elasticsearch postgresql; do
    if helm list -n "$namespace" | grep -q "$release"; then
        helm uninstall "$release" -n "$namespace"
    fi
done

# Delete monitoring stack
log_info "Deleting monitoring stack"
kubectl delete -f ../monitoring/ || true

# Delete logging stack
log_info "Deleting logging stack"
kubectl delete -f ../logging/ || true

# Delete network policies
log_info "Deleting network policies"
kubectl delete -f ../network-policies/ || true

# Delete ingress
log_info "Deleting ingress"
kubectl delete -f ../ingress/ || true

# Delete storage classes
log_info "Deleting storage classes"
kubectl delete -f ../storage/storage-classes.yaml || true

# Delete persistent volumes
log_info "Deleting persistent volumes"
kubectl delete pvc --all -n "$namespace" || true

# Delete namespace
log_info "Deleting namespace $namespace"
kubectl delete namespace "$namespace" || true

log_info "Undeployment completed successfully"
