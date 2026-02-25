#!/bin/bash

set -e

# Deployment script for Logos Platform on Kubernetes

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

# Check if namespace exists
namespace="logging-system"
if ! kubectl get namespace "$namespace" &> /dev/null; then
    log_info "Creating namespace $namespace"
    kubectl create namespace "$namespace"
fi

# Add Bitnami chart repository
log_info "Adding Bitnami chart repository"
helm repo add bitnami https://charts.bitnami.com/bitnami || true
helm repo update

# Deploy monitoring stack
log_info "Deploying monitoring stack"
kubectl apply -f ../monitoring/

# Deploy logging stack
log_info "Deploying logging stack"
kubectl apply -f ../logging/

# Deploy network policies
log_info "Deploying network policies"
kubectl apply -f ../network-policies/

# Deploy ingress
log_info "Deploying ingress"
kubectl apply -f ../ingress/

# Deploy storage
log_info "Deploying storage classes"
kubectl apply -f ../storage/storage-classes.yaml

# Deploy Logos Platform using Helm
log_info "Deploying Logos Platform"
helm install logos ../charts/logos --namespace "$namespace"

log_info "Deployment completed successfully"
