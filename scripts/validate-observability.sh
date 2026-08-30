#!/bin/bash
# Validate observability setup with actual traffic
# Usage: ./scripts/validate-observability.sh

set -euo pipefail

BASE_URL=${BASE_URL:-http://localhost:8080}
PROM_URL=${PROM_URL:-http://localhost:9090}

echo "============================================"
echo "Observability Validation"
echo "============================================"
echo ""

# Pre-test expectations
echo "=== Pre-Test Expectations ==="
cat << 'EOF'
| Panel                  | Expected Behavior           |
|------------------------|-----------------------------|
| Request Rate           | > 0 req/s during test       |
| Error Rate             | ~10% (404s from /tasks/999) |
| P99 Latency            | < 500ms                     |
| Task Count             | Increases during test       |
EOF
echo ""

# Check Prometheus target
echo "=== Checking Prometheus Target ==="
TARGET_HEALTH=$(curl -s "$PROM_URL/api/v1/targets" | python3 -c '
import sys,json
d=json.load(sys.stdin)
for t in d["data"]["activeTargets"]:
    if t["labels"]["job"]=="task-api":
        print(t["health"])
' 2>/dev/null || echo "unknown")

if [ "$TARGET_HEALTH" = "up" ]; then
    echo "PASS: Prometheus target is UP"
else
    echo "FAIL: Prometheus target is $TARGET_HEALTH"
fi
echo ""

# Generate traffic
echo "=== Generating Traffic ==="
./scripts/generate-traffic.sh 30
echo ""

# Wait for metrics to be scraped
echo "=== Waiting for scrape ==="
sleep 12
echo ""

# Validate metrics
echo "=== Validating Metrics ==="

# Request count
TOTAL_REQUESTS=$(curl -s "$PROM_URL/api/v1/query?query=sum(increase(http_requests_total[5m]))" | python3 -c '
import sys,json
d=json.load(sys.stdin)
if d["data"]["result"]:
    print(d["data"]["result"][0]["value"][1])
else:
    print("0")
' 2>/dev/null || echo "0")
echo "Total requests (5m): $TOTAL_REQUESTS"

# Error count
ERROR_REQUESTS=$(curl -s "$PROM_URL/api/v1/query?query=sum(increase(http_requests_total{status=~\"5..\"}[5m]))" | python3 -c '
import sys,json
d=json.load(sys.stdin)
if d["data"]["result"]:
    print(d["data"]["result"][0]["value"][1])
else:
    print("0")
' 2>/dev/null || echo "0")
echo "Error requests (5m): $ERROR_REQUESTS"

# P99 Latency
P99=$(curl -s "$PROM_URL/api/v1/query?query=histogram_quantile(0.99,sum(rate(http_request_duration_seconds_bucket[5m]))by(le))" | python3 -c '
import sys,json
d=json.load(sys.stdin)
if d["data"]["result"]:
    print(d["data"]["result"][0]["value"][1])
else:
    print("N/A")
' 2>/dev/null || echo "N/A")
echo "P99 Latency: ${P99}s"

# Task count
TASK_COUNT=$(curl -s "$PROM_URL/api/v1/query?query=task_api_tasks_total" | python3 -c '
import sys,json
d=json.load(sys.stdin)
if d["data"]["result"]:
    print(d["data"]["result"][0]["value"][1])
else:
    print("0")
' 2>/dev/null || echo "0")
echo "Task count: $TASK_COUNT"

echo ""
echo "=== Validation Complete ==="
echo ""
echo "Next steps:"
echo "1. Open Grafana at http://localhost:3000 (admin/admin123)"
echo "2. Navigate to Dashboards > Task API Dashboard"
echo "3. Verify panels show data matching the metrics above"
