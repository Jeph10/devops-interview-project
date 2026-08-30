# CI/CD & Promotion Workflow — Documentation & Verification Guide

> Source of truth: [`ci.yml`](ci.yml) (CI/CD pipeline) and [`prod.yml`](prod.yml) (staging→production promotion).
> This document explains what each stage does **and how to verify that it actually worked**.

## Overview

| File | Purpose | Trigger |
|------|---------|---------|
| `ci.yml` | Lint → test → build → scan → push → deploy to **staging** | push / PR to `main`, `workflow_dispatch` |
| `prod.yml` | Manual **promotion** of a validated image from staging to production | `workflow_dispatch` only |

Key properties:

- **Traceability** — images are tagged with the full commit SHA (`${{ github.sha }}`), plus `:latest` and environment tags (`:staging`, `:production`).
- **Self-hosted deploy** — `deploy-staging` runs on runner `mac-mini-1` (labels `[self-hosted, kube]`) because the target cluster (docker-desktop Kubernetes) is local-only; GitHub-hosted runners cannot reach `https://kubernetes.docker.internal:6443`.
- **Gated promotion** — production requires typed confirmation (`confirm=YES`), an image-existence check, a Trivy CRITICAL gate, and automatic rollback on failure.
- **Guardrails** — concurrency group cancels stale runs, least-privilege `permissions`, and `deploy-staging` only fires for pushes to `refs/heads/main`, never PRs.

## Pipeline Flow (actual job graph)

```text
CI/CD PIPELINE (ci.yml) — push to main / PRs to main / manual:

  lint ──▶ test ──▶ build ──▶ docker-build-and-scan ──▶ docker-push ──▶ deploy-staging
  (go vet +         (go test         (docker build +      (GHCR,       ([self-hosted, kube])
   golangci-lint)    -race)           Trivy SARIF)         SHA tag)          │
                                                                    ▼
                                             rollout status ✔ ──▶ retag & push :staging

PROMOTION PIPELINE (prod.yml) — manual workflow_dispatch (version + confirm=YES):

  validate-input ──▶ staging-health ──▶ security-gate ──▶ promote ──▶ post-promotion-verify
  (confirm=YES,        (health gate)      (Trivy            (deploy     (pods/svc/rollout/
   SHA/semver,                            CRITICAL gate)    prod, tag    smoke test)
   image exists)                                           :production,
                                                           audit record)

  promote fails ──▶ rollback job (kubectl rollout undo)
```

## CI/CD Pipeline (`ci.yml`)

### Jobs

| # | Job | Runner | Needs | What it does |
|---|-----|--------|-------|--------------|
| 1 | `lint` | `ubuntu-latest` | — | `go vet ./...` + `golangci-lint run --disable errcheck` |
| 2 | `test` | `ubuntu-latest` | lint | `go test -race -count=1 ./...`; uploads `test-results` artifact (5-day retention) |
| 3 | `build` | `ubuntu-latest` | test | `CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w"`; uploads `binary` artifact |
| 4 | `docker-build-and-scan` | `ubuntu-latest` | build | Builds `task-api:<sha>` + `:latest`, exports `image-digest`; Trivy (CRITICAL/HIGH/MEDIUM) → SARIF to the Security tab |
| 5 | `docker-push` | `ubuntu-latest` | docker-build-and-scan | Pushes `ghcr.io/${{ github.actor }}/devops-interview-project:{<sha>,latest}` with GHA build cache |
| 6 | `deploy-staging` | **`[self-hosted, kube]`** | docker-push | Real deploy: `kubectl apply` of `deploy/k8s-staging.yaml`, `set image` to the SHA tag, patches `imagePullPolicy`, waits on `rollout status` (180s), then retags/pushes `:staging` |

### Deploy-staging decision logic

```yaml
if KUBECONFIG_STAGING secret exists:
    write kubeconfig → kubectl apply --dry-run=client → set image
    → patch imagePullPolicy → rollout status (180s) → verify pods/svc → retag :staging
else:
    print simulation notice + setup instructions (run still succeeds, marked simulated)
```

### Outputs

- Image tags: `:<sha>` (traceable), `:latest` (branch head), `:staging` (deployed to staging)
- Artifacts: `test-results`, `binary`; SARIF report in **Security → Code scanning**
- Cluster state: namespace `task-api-staging` — deployment `task-api` (2 replicas, HPA 2→10), ClusterIP service, ingress `staging.task-api.example.com`

---

## Production Promotion Pipeline (`prod.yml`)

### Manual trigger

**GitHub → Actions → Promote to Production → Run workflow**

| Input | Required | Format | Example |
|-------|----------|--------|---------|
| `version` | yes | 40-hex commit SHA **or** semver | `e35bd33…` / `v1.0.0` |
| `confirm` | yes | literal `YES` | `YES` |

### Jobs

| # | Job | Needs | What it does | Failure behavior |
|---|-----|-------|--------------|------------------|
| 1 | `validate-input` | — | Rejects missing `YES`; validates SHA/semver; **verifies the image exists in GHCR** by pulling it | Fails fast |
| 2 | `staging-health` | validate-input | Staging health gate (simulated unless a staging URL is configured) | Warn-only (`if: always()`) |
| 3 | `security-gate` | validate-input | Re-scans the target image with Trivy; blocks promotion on new CRITICAL vulnerabilities | Fails promotion |
| 4 | `promote` | 1–3 | Applies `deploy/k8s-production.yaml` with the prod kubeconfig, waits for rollout, retags `:production`, writes `promotion-record.json` (who/when/what/run URL) | Triggers `rollback` |
| 5 | `post-promotion-verify` | promote | `kubectl get pods/svc` + `rollout status` + smoke test | — |
| 6 | `rollback` | promote (`if: failure()`) | `kubectl rollout undo` + status wait + error annotation on the run | — |

---

## ✅ Verification & Validation Guide

Every stage produces a checkable artifact. Verify locally, in GitHub Actions, and in the cluster.

### 1. Verify before you push (local)

| Check | Command | Pass criteria |
|-------|---------|---------------|
| Static analysis | `go vet ./...` | No output |
| Tests + races | `go test -race -count=1 ./...` | `ok` line, no `DATA RACE` |
| Binary builds | `CGO_ENABLED=0 go build -o /tmp/task-api .` | Binary created, no errors |
| Docker build | `docker build -t task-api:local .` | Image < 15 MiB |
| Container health | `docker run -d -p 8080:8080 task-api:local` then `docker inspect --format='{{.State.Health.Status}}' <cid>` | `healthy` |
| Workflow dry-run | `act -j lint -n` / `act -j test --container-architecture linux/amd64` | Jobs parse + execute locally |

### 2. Verify each pipeline stage (GitHub Actions)

| Stage | Where to look | Pass criteria |
|-------|---------------|---------------|
| lint / test / build | Run logs (`ubuntu-latest`) | Green; `test-results` artifact uploaded |
| Trivy scan | **Security → Code scanning alerts** | SARIF uploaded; no open CRITICAL alerts |
| docker-push | "Push manifest" log lines | Digest printed for `ghcr.io/<owner>/devops-interview-project:<sha>` |
| deploy-staging | Job log — runner shown as `mac-mini-1` | `✓ Staging deployment successful`; `rollout status` → `successfully rolled out` |

Via GitHub CLI:

```bash
gh run list --workflow=ci.yml --limit 5
gh run view <run-id>                          # job/step summary
gh run view <run-id> --log | grep -E 'Deploying|rolled out|tagged'
gh secret list                                # KUBECONFIG_STAGING present?
gh api repos/<owner>/<repo>/actions/runners \
  --jq '.runners[] | {name, status, labels: [.labels[].name]}'   # runner online?
```

### 3. Verify staging deployment (in the cluster)

```bash
# 1. Rollout completed (not stuck/progressing)
kubectl rollout status deployment/task-api -n task-api-staging --timeout=30s

# 2. Correct, commit-traceable image
kubectl get deployment task-api -n task-api-staging \
  -o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}'
# expect: ghcr.io/<owner>/devops-interview-project:<full-40-char-sha>

# 3. Image tag == commit that triggered the run
git rev-parse HEAD        # compare with the tag above

# 4. Pods Ready, not CrashLoopBackOff / ImagePullBackOff
kubectl get pods -n task-api-staging -l app=task-api

# 5. Service + HPA exist
kubectl get svc,hpa -n task-api-staging

# 6. Health + metrics respond
kubectl port-forward -n task-api-staging svc/task-api 8081:80 &
curl -sf http://localhost:8081/healthz                    # {"status":"ok"}
curl -s  http://localhost:8081/metrics | grep http_requests_total

# 7. :staging tag pushed to the registry
docker manifest inspect ghcr.io/<owner>/devops-interview-project:staging >/dev/null && echo OK
```

### 4. Verify production promotion (in the cluster)

```bash
kubectl get pods -n task-api-production -l app=task-api            # 3/3 Running
kubectl get deployment task-api -n task-api-production \
  -o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}'    # == promoted version
kubectl rollout history deployment/task-api -n task-api-production # promotion = new revision
curl -sf http://localhost:<prod-nodeport>/healthz                  # {"status":"ok"}
```

Promotion audit trail:

- `promotion-record.json` artifact on the `prod.yml` run (`promotion_id`, image, actor, workflow URL, timestamp)
- `:production` tag on GHCR points at the promoted digest

### 5. Reference: last fully validated run

| Field | Value |
|-------|-------|
| Run | `#33333382514` — **completed / success** (push to `main`) |
| Jobs | `lint`, `test`, `build`, `docker-build-and-scan`, `docker-push` (ubuntu-latest) ✅ · `deploy-staging` on `mac-mini-1` ✅ |
| Image | `ghcr.io/jeph10/devops-interview-project:e35bd330…` (SHA-tagged) |
| Cluster result | `task-api-staging`: 2/2 pods `Running`, rollout `successfully rolled out`, svc + HPA present, `/healthz` → `{"status":"ok"}` |

### 6. Verification failure playbook

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `deploy-staging` logs "simulating deployment" | `KUBECONFIG_STAGING` secret missing/empty | Re-add secret (helper: `python3 scripts/set-kubeconfig-secret.py`), re-run |
| Job queued forever, no runner | Self-hosted runner offline | `cd ~/actions-runner && ./run.sh`; confirm via `gh api .../actions/runners` → `online` |
| `ImagePullBackOff` | Cluster kubelet can't fetch GHCR (auth/rate limit) | Runner pre-pulls via `docker login ghcr.io` step; keep the `imagePullPolicy: IfNotPresent` patch |
| Rollout timeout (180s) | Probe/manifest regression | `kubectl logs` / `describe pod -n task-api-staging`, fix, re-run |
| Old image still running after green run | Deployment patch didn't apply | Re-run the `kubectl set image` step; confirm with the jsonpath check above |

---

## Required Secrets

| Secret | Used By | Description |
|--------|---------|-------------|
| `KUBECONFIG_STAGING` | ci.yml (`deploy-staging`) | Staging cluster kubeconfig, **base64-encoded** (`base64 -i ~/.kube/config`) |
| `KUBECONFIG_PROD` | prod.yml (`promote`, `post-promotion-verify`, `rollback`) | Production cluster kubeconfig, base64-encoded |
| `GITHUB_TOKEN` | both | Auto-provided by GitHub (GHCR login, SARIF upload) — no manual setup |

### Configuring Secrets

1. **Settings → Secrets and variables → Actions → New repository secret**
2. Name: `KUBECONFIG_STAGING`, value: output of `base64 -i ~/.kube/config` (or the helper: `python3 scripts/set-kubeconfig-secret.py`)
3. Repeat for `KUBECONFIG_PROD`
4. Verify: `gh secret list`

Secrets are masked automatically in run logs. Never commit kubeconfigs or tokens to the repo.

---

## Promotion Checklist

Before promoting to production:

- [ ] CI run green on `main` for the exact commit being promoted (`gh run list --workflow=ci.yml`)
- [ ] Staging healthy: `kubectl rollout status deployment/task-api -n task-api-staging` → `successfully rolled out`
- [ ] Staging `/healthz` → `{"status":"ok"}` (via port-forward)
- [ ] No open CRITICAL alerts under Security → Code scanning
- [ ] `version` input matches the SHA/semver validated on staging
- [ ] Rollback target confirmed: `kubectl rollout history deployment/task-api -n task-api-production`
- [ ] Team notified of the promotion window

---

## Rollback Procedure

### Automatic (prod.yml `rollback` job)

Fires on `promote` failure: `kubectl rollout undo deployment/task-api` → `rollout status` (120s) → error annotation on the run.

### Manual

```bash
# Previous revision
kubectl rollout undo deployment/task-api -n task-api-production

# Specific revision
kubectl rollout undo deployment/task-api -n task-api-production --to-revision=2

# History & status
kubectl rollout history deployment/task-api -n task-api-production
kubectl rollout status  deployment/task-api -n task-api-production --timeout=120s
```

**Post-rollback verification**: pods run the previous image (jsonpath check above), `/healthz` returns `{"status":"ok"}`, and Grafana error-rate/latency panels recover.

---

## Monitoring & Alerts

After promotion, verify:

1. **Grafana** (http://localhost:3000, `admin`/`admin123`) — error-rate, latency and request-rate panels show the new deployment's traffic
2. **Prometheus** (http://localhost:9090) — `up{job="task-api"}` == 1; add the production NodePort as a scrape target to watch production
3. **Pods** — `kubectl get pods -n task-api-production`

### Key metrics & alert wiring

| Metric | SLO / Threshold | Alert (`monitoring/alerts.yml`) | Severity |
|--------|-----------------|--------------------------------|----------|
| Availability | 99.9% (30d) | `TaskAPIHighErrorRate` / `TaskAPIDown` | Critical |
| Error rate | < 5% for 2m | `TaskAPIHighErrorRate` | Critical |
| P99 latency | < 1000ms for 5m | `TaskAPIHighLatency` | Warning |
| Container restarts | > 0 in 1h | `TaskAPIContainerRestart` | Warning |

Full definitions: [`monitoring/sli-slo.md`](../../monitoring/sli-slo.md) and [`monitoring/alerts.yml`](../../monitoring/alerts.yml).

---

## Troubleshooting

### Promotion fails at "Verify image exists"

```bash
docker pull ghcr.io/<owner>/devops-interview-project:<version>

# List available tags
curl -s -H "Authorization: Bearer $GITHUB_TOKEN" \
  https://ghcr.io/v2/<owner>/devops-interview-project/tags/list
```

### Deployment timeout

```bash
kubectl get pods -n task-api-production
kubectl logs -n task-api-production -l app=task-api --tail=50
kubectl get events -n task-api-production --sort-by='.lastTimestamp'
```

### Rollback stuck

```bash
kubectl rollout undo  deployment/task-api -n task-api-production
kubectl rollout status deployment/task-api -n task-api-production --timeout=300s
```

---

## Related Documentation

- [CI/CD pipeline](ci.yml) · [Promotion pipeline](prod.yml)
- Kubernetes manifests: [staging](../../deploy/k8s-staging.yaml) · [production](../../deploy/k8s-production.yaml)
- [Alert rules](../../monitoring/alerts.yml) · [SLI/SLO definition](../../monitoring/sli-slo.md)
- [Implementation notes](../../deploy/NOTES.md) — assumptions, trade-offs and evidence