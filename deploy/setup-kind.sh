#!/bin/bash
# Setup a local kind cluster for testing deployment
# Usage: ./deploy/setup-kind.sh

set -euo pipefail

CLUSTER_NAME="task-api-ci"

echo "=== Creating kind cluster: ${CLUSTER_NAME} ==="
kind create cluster --name "${CLUSTER_NAME}" --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-port
    extraPortMappings:
      - containerPort: 80
        hostPort: 8081
        protocol: TCP
EOF

echo "=== Waiting for cluster to be ready ==="
kubectl --context "kind-${CLUSTER_NAME}" wait --for=condition=Ready nodes --all --timeout=120s

echo "=== Loading image into kind ==="
kind load docker-image task-api:latest --name "${CLUSTER_NAME}"

echo "=== Deploying application ==="
kubectl --context "kind-${CLUSTER_NAME}" apply -f deploy/k8s-deployment.yaml

echo "=== Waiting for rollout ==="
kubectl --context "kind-${CLUSTER_NAME}" rollout status deployment/task-api --timeout=120s

echo "=== Deployment complete ==="
echo "Access the service at: kubectl --context kind-${CLUSTER_NAME} port-forward svc/task-api 8080:80"
echo ""
echo "=== Cluster info ==="
kubectl --context "kind-${CLUSTER_NAME}" cluster-info
echo ""
echo "=== Pods ==="
kubectl --context "kind-${CLUSTER_NAME}" get pods -l app=task-api