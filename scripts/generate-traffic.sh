#!/bin/bash
# Generate test traffic for observability validation
# Usage: ./scripts/generate-traffic.sh [num_requests]

set -euo pipefail

NUM_REQUESTS=${1:-50}
BASE_URL=${BASE_URL:-http://localhost:8080}

echo "=== Generating $NUM_REQUESTS requests over ~2 minutes ==="
echo "Start time: $(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Create tasks
for i in $(seq 1 "$NUM_REQUESTS"); do
  curl -s -X POST "$BASE_URL/tasks" \
    -H 'Content-Type: application/json' \
    -d "{\"title\":\"task $i\"}" > /dev/null

  # Get existing task (200)
  curl -s "$BASE_URL/tasks/0" > /dev/null

  # Get non-existent task (404) - intentional failure
  if [ "$((i % 10))" -eq 0 ]; then
    curl -s "$BASE_URL/tasks/999" > /dev/null
  fi

  # Small delay to spread traffic over time
  sleep 0.1
done

echo "End time: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "=== Traffic generation complete ==="
echo ""
echo "Expected metrics:"
echo "  - http_requests_total{method=POST,path=/tasks,status=201}: $NUM_REQUESTS"
echo "  - http_requests_total{method=GET,path=/tasks/{id},status=200}: $NUM_REQUESTS"
echo "  - http_requests_total{method=GET,path=/tasks/{id},status=404}: $((NUM_REQUESTS / 10))"
