# Implementation Notes and Decision Record

## 1. Key Assumptions

**Assumption 1: Distroless base image provides sufficient runtime for the Go application.**
The application uses only the Go standard library plus Prometheus client (pure Go, no CGO). If the app required CA certificates for outbound TLS or a shell for debugging, `distroless/static:nonroot` would be insufficient and I would need `distroless/base:nonroot` or `alpine`.

**Assumption 2: Docker Hub rate limits will not affect CI runners.**
The workflow pulls `golang:1.26` and `gcr.io/distroless/static:nonroot` from public registries. If rate limits are hit, the `docker-build` and `docker-push` jobs would fail. Mitigation: mirror images to GHCR or use authenticated pulls.

**Assumption 3: The deployment target is a local Kubernetes cluster (kind).**
The `deploy` job uses `kubectl` with a kubeconfig secret. If no cluster is available, the deployment step cannot run online. I validated the deployment manifests locally using `docker compose` instead.

## 2. Delivery Path

**PR workflow** (`.github/workflows/ci.yml`):
1. `lint` job: `go vet ./...` + `golangci-lint-action@v6`
2. `test` job: `go test -race -count=1 ./...`
3. `build` job: `CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w"`
4. `docker-build` job: builds image tagged `task-api:${{ github.sha }}`

**Push to main workflow** (continues from above):
5. `docker-push` job: pushes to `ghcr.io/${{ github.repository }}:${{ github.sha }}` and `:latest`
6. `deploy` job: `kubectl set image deployment/task-api app=ghcr.io/...:${{ github.sha }}` then `kubectl rollout status`

**Artifact identification**: Images tagged with commit SHA (`${{ github.sha }}`) for traceability.

**Rollback unit**: `kubectl rollout undo deployment/task-api` reverts to previous ReplicaSet.

**Validation boundary**: GHCR push and Kubernetes deployment were not run online (no cluster credentials). Validated locally with `docker compose up -d` and `docker build -t task-api .`.

## 3. One Actual Validation

**What I validated**: That the Docker image would report as `healthy` (not just have a HEALTHCHECK instruction) while staying under 15 MiB.

**Expected**: The container health status would transition from `starting` to `healthy` within 30 seconds, and `docker image inspect task-api --format '{{.Size}}'` would return less than 15,728,640 bytes.

**Commands run**:
```bash
docker build -t task-api .
docker run -d -p 8080:8080 --name task-api task-api
docker inspect --format='{{.State.Health.Status}}' task-api
docker image inspect task-api --format '{{.Size}}'
```

**Evidence**:
- `deploy/evidence/docker-health.txt`: `healthy`
- `deploy/evidence/image-size.txt`: `8852804` bytes (8.44 MB) without Prometheus, `12854596` bytes (12.25 MB) with Prometheus client

**Why this worked**: I added a `-healthcheck` flag to the Go binary (`main.go:69-92`) that probes `http://127.0.0.1:8080/healthz` internally. This avoids needing `curl`/`wget`/`sh` in the distroless image, which has none of these tools.

**Change made**: Initially the Dockerfile used `debian:bookworm-slim` (~75 MiB base). I switched to `gcr.io/distroless/static:nonroot` (~2 MiB base) and added `-ldflags="-s -w"` to strip the binary. The repeated test showed 8.44 MB (without deps) and 12.25 MB (with Prometheus client), both under the 15 MiB limit.

## 4. Two Engineering Trade-offs

**Trade-off 1: Distroless over Alpine for the runtime image**

- **Constraint**: Image must be <15 MiB and run as non-root.
- **Options**: `alpine:3.23` (~5 MiB, has shell), `distroless/static:nonroot` (~2 MiB, no shell), `debian:bookworm-slim` (~75 MiB, has shell)
- **Choice**: `distroless/static:nonroot` for minimal attack surface and size.
- **Validation**: `docker scout cves task-api` reported `0C 0H 0M 0L` vulnerabilities.
- **Remaining risk**: Cannot `exec` into the container for debugging (no shell).
- **Would change if**: Production requires live debugging via `kubectl exec`. Would switch to `distroless/base:nonroot` (adds ~4 MiB) or `alpine`.

**Trade-off 2: Global Prometheus registry with per-test override**

- **Constraint**: Tests must not fail with "duplicate metrics collector registration attempted" when multiple tests create `NewPrometheusMiddleware()`.
- **Options**: (a) Use a global registry and reset between tests, (b) Accept a `Registerer` parameter in the constructor, (c) Use `promauto.With(reg)` with a per-test registry.
- **Choice**: Added `NewPrometheusMiddlewareWithRegistry(reg prometheus.Registerer)` for tests while keeping `NewPrometheusMiddleware()` using the default global registry for production.
- **Validation**: `go test -race -count=1 ./...` passes with the new `TestMetricsCounterIncremented` and `TestMetricsDurationRecorded` tests using per-test registries.
- **Remaining risk**: Production code must use the default constructor; accidentally passing a test registry would isolate metrics from the global `/metrics` endpoint.
- **Would change if**: The codebase grows to need multiple middleware instances. Would refactor to always accept a registry explicitly.

## 5. Actual Time Spent

- **Actual time spent**: ~3 hours (including dependency resolution, Docker Compose healthcheck debugging, and all bonus implementations)
- **Work deliberately left out**: Image signing with cosign. This is an optional bonus item beyond the core requirements.
- **Bonus items completed** (4 of 6 bonus items from README):
  1. **Alert rules** (`monitoring/alerts.yml`): 6 Prometheus alert rules including HighErrorRate, HighLatency, ServiceDown, ContainerRestart, LowRequestVolume, MemoryPressure
  2. **SLI/SLO definition** (`monitoring/sli-slo.md`): 99.9% availability over 30 days, error budget policy, burn rate alerting examples
  3. **Image scanning in CI** (`.github/workflows/ci.yml`): Trivy scanner integrated, uploads SARIF to GitHub Security tab, warns on CRITICAL vulnerabilities
  4. **Staging-to-production promotion** (`.github/workflows/promote.yml`): Manual approval workflow with validation gates, promotion tracking, and automatic rollback
- **What I would do next with another 60 minutes**: Implement graceful shutdown with signal handling and demonstrate repeatable validation of in-flight request completion.
--------------------------------------------------------------------------------------------------------
## 6. Use of AI **All AI-suggested code was reviewed, tested, and modified based on actual results.**
-----------------------------------------------------------------------------------------------------------
- **Tool**: Cline (AI coding agent)
- **Transcripts**: `deploy/ai-transcripts/session-01-prompt-enhanced.md`
- **Sessions**: Multiple sessions over ~3 hours 

**Example modified output**: 
1. **Dockerfile Choice**: The AI initially suggested `alpine:3.23` as the base image. I rejected this after calculating that `alpine` (5 MiB) + Go binary with Prometheus client (~10 MiB) would leave minimal headroom under the 15 MiB limit. I chose `gcr.io/distroless/static:nonroot` (~2 MiB) instead, which provided more margin. Evidence: `docker scout cves` showed 0 vulnerabilities for distroless.

2. **CI/CD Job Structure**: The AI created separate `docker-build` and `scan` jobs. I combined them into `docker-build-and-scan` because GitHub Actions jobs run on separate runners, and the image built in one job was not accessible in the other.

3. **Prompt Engineering**: I iteratively improved the AI prompts, adding troubleshooting guidance, expected outputs, and time estimates. The final `prompt-enhanced.md` includes 13 detailed prompts with validation steps.

**Tasks where AI provided significant help**:
- Repository analysis and workflow documentation
- Bug fixing (generate-dashboard.py had severe syntax errors)
- Bonus implementations (alerts, SLI/SLO, promotion pipeline)
- GitHub Actions troubleshooting (8 separate issues fixed)

