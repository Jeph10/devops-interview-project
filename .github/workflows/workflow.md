# Staging-to-Production Promotion Pipeline

This document describes the staging-to-production promotion workflow for the Task API.

## Overview

The promotion pipeline ensures that only validated, approved images are deployed to production. It implements a manual approval gate with automated validation checks.

## Workflow Files

| File | Purpose |
|------|---------|
| `ci.yml` | Main CI/CD pipeline - builds, tests, scans, and deploys to staging |
| `prod.yml` | Production promotion workflow - manual promotion from staging to production |

## Pipeline Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CI/CD PIPELINE (ci.yml)                           │
│                                                                             │
│  push to main ──▶ lint ──▶ test ──▶ build ──▶ docker-build ──▶ scan       │
│                                                                  │          │
│                                                                  ▼          │
│                                                           docker-push       │
│                                                                  │          │
│                                                                  ▼          │
│                                                           deploy-staging    │
│                                                                  │          │
│                                                                  ▼          │
│                                                           tag as :staging   │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                      PROMOTION PIPELINE (prod.yml)                          │
│                                                                             │
│  Manual Trigger ──▶ validate ──▶ staging-health ──▶ security-gate          │
│       (workflow_dispatch)                                    │              │
│                                                            ▼                │
│                                                     promote ──▶ verify      │
│                                                            │                │
│                                                            ▼                │
│                                                    tag as :production       │
│                                                                             │
│  On Failure: automatic rollback                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## CI/CD Pipeline (ci.yml)

### Jobs

1. **lint** - Static analysis with `go vet` and `golangci-lint`
2. **test** - Unit tests with race detector
3. **build** - Compile static Go binary
4. **docker-build** - Build Docker image with commit SHA tag
5. **scan** - Trivy vulnerability scan (uploads to GitHub Security tab)
6. **docker-push** - Push image to GHCR
7. **deploy-staging** - Deploy to staging Kubernetes cluster

### Triggers

- Push to `main` branch
- Pull request targeting `main`
- Manual workflow dispatch

### Output

- Image tagged as `:latest`, `:<sha>`, and `:staging`
- Deployed to `task-api-staging` namespace

---

## Production Promotion Pipeline (prod.yml)

### Manual Trigger

Access via: **GitHub → Actions → Promote to Production → Run workflow**

### Required Inputs

| Input | Description | Validation |
|-------|-------------|------------|
| `version` | Image tag to promote (commit SHA or semver) | Must be 40-char hex or `vX.Y.Z` |
| `confirm` | Confirmation flag | Must be exactly "YES" |

### Jobs

#### 1. validate-input
- Checks confirmation input is "YES"
- Validates version format (SHA or semver)
- Verifies image exists in GHCR registry

#### 2. staging-health
- Validates staging environment health
- Can be extended with actual health checks

#### 3. security-gate
- Re-scans image with Trivy
- Warns if CRITICAL vulnerabilities found
- Allows manual override for urgent fixes

#### 4. promote
- Deploys to production Kubernetes cluster
- Uses `KUBECONFIG_PROD` secret
- Tags image as `:production`
- Creates promotion record artifact

#### 5. post-promotion-verify
- Verifies deployment succeeded
- Runs smoke tests (simulated)

#### 6. rollback (on failure)
- Automatically rolls back on deployment failure
- Notifies via workflow failure status

### Promotion Record

Each promotion generates a JSON artifact:

```json
{
  "promotion_id": "promo-20260830-021900-12345",
  "timestamp": "2026-08-30T02:19:00Z",
  "image": "ghcr.io/your-org/task-api:a1b2c3d4e5f6...",
  "promoted_by": "username",
  "workflow_run": "https://github.com/.../actions/runs/...",
  "status": "success"
}
```

---

## Kubernetes Manifests

### Staging (`deploy/k8s-staging.yaml`)

```yaml
Namespace: task-api-staging
Replicas: 2
Resources: 64Mi-128Mi memory, 100m-200m CPU
HPA: 2-10 replicas
Ingress: staging.task-api.example.com
```

### Production (`deploy/k8s-production.yaml`)

```yaml
Namespace: task-api-production
Replicas: 3
Resources: 128Mi-256Mi memory, 200m-500m CPU
HPA: 3-20 replicas
Pod Anti-Affinity: Enabled
PodDisruptionBudget: minAvailable=2
Ingress: api.example.com (with TLS + rate limiting)
```

---

## Required Secrets

| Secret | Used By | Description |
|--------|---------|-------------|
| `KUBECONFIG_STAGING` | ci.yml (deploy-staging) | Staging cluster kubeconfig (base64) |
| `KUBECONFIG_PROD` | prod.yml (promote) | Production cluster kubeconfig (base64) |
| `GITHUB_TOKEN` | ci.yml, prod.yml | Auto-provided by GitHub Actions |

### Configuring Secrets

1. Go to repository **Settings → Secrets and variables → Actions**
2. Click **New repository secret**
3. Add each secret:
   - Name: `KUBECONFIG_STAGING`
   - Value: `$(base64 -i ~/.kube/config-staging)`
4. Repeat for `KUBECONFIG_PROD`

---

## Promotion Checklist

Before promoting to production:

- [ ] Staging deployment is healthy
- [ ] All tests passed in CI
- [ ] No new CRITICAL vulnerabilities in scan
- [ ] Version matches expected commit or release
- [ ] Team notified of promotion window

---

## Rollback Procedure

### Automatic Rollback

The `rollback` job in `prod.yml` automatically triggers on failure:

```bash
kubectl rollout undo deployment/task-api
kubectl rollout status deployment/task-api --timeout=120s
```

### Manual Rollback

```bash
# Rollback to previous revision
kubectl rollout undo deployment/task-api -n task-api-production

# Rollback to specific revision
kubectl rollout undo deployment/task-api -n task-api-production --to-revision=2

# Check rollout history
kubectl rollout history deployment/task-api -n task-api-production
```

---

## Monitoring & Alerts

After promotion, verify:

1. **Grafana Dashboard**: Check error rate and latency panels
2. **Prometheus Alerts**: Ensure no alerts firing
3. **Pod Status**: `kubectl get pods -n task-api-production`

### Key Metrics

| Metric | Threshold | Alert |
|--------|-----------|-------|
| Availability | > 99.9% | TaskAPIHighErrorRate |
| P99 Latency | < 1000ms | TaskAPIHighLatency |
| Error Rate | < 5% | TaskAPIHighErrorRate |

---

## Troubleshooting

### Promotion Fails at "Verify image exists"

```bash
# Check image in local registry
docker pull ghcr.io/your-org/task-api:<version>

# Check available tags
curl -H "Authorization: Bearer $GITHUB_TOKEN" \
  https://ghcr.io/v2/your-org/task-api/tags/list
```

### Deployment Timeout

```bash
# Check pod status
kubectl get pods -n task-api-production

# Check pod logs
kubectl logs -n task-api-production -l app=task-api

# Check events
kubectl get events -n task-api-production --sort-by='.lastTimestamp'
```

### Rollback Stuck

```bash
# Force rollback
kubectl rollout undo deployment/task-api -n task-api-production --force

# Check rollout status
kubectl rollout status deployment/task-api -n task-api-production --timeout=300s
```

---

## Related Documentation

- [Alert Rules](../../monitoring/alerts.yml) - Prometheus alert definitions
- [SLI/SLO Definition](../../monitoring/sli-slo.md) - Service level objectives
- [Implementation Notes](../../deploy/NOTES.md) - Engineering trade-offs and decisions
