# Service Level Indicators (SLIs) and Objectives (SLOs) for Task API

## Overview

This document defines the reliability targets for the Task API service. These SLOs are used to:
- Guide engineering priorities and resource allocation
- Trigger alerts when service quality degrades
- Measure the effectiveness of reliability improvements
- Set expectations with stakeholders about service reliability

---

## Service Level Indicators (SLIs)

### 1. Availability SLI

**Definition**: The ratio of successful HTTP requests to total requests.

```
Availability = (Successful Requests / Total Requests) × 100%
```

**Measurement**:
- Successful request: HTTP status code 2xx or 3xx
- Total requests: All HTTP requests received
- Metrics: `sum(rate(http_requests_total{status=~"2..|3.."}[1m])) / sum(rate(http_requests_total[1m]))`

**Exclusions**:
- Client errors (4xx) are excluded from availability calculation as they indicate client issues, not service failures
- Requests from health checks and synthetic monitoring

---

### 2. Latency SLI

**Definition**: The ratio of requests served within the latency threshold.

```
Latency SLI = (Requests with latency < threshold / Total Requests) × 100%
```

**Thresholds**:
- P50 (median): < 100ms
- P95: < 500ms  
- P99: < 1000ms

**Measurement**:
- `histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket[1m])) by (le))`
- `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[1m])) by (le))`
- `histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[1m])) by (le))`

---

### 3. Throughput SLI

**Definition**: The number of requests processed per second.

**Measurement**:
- `sum(rate(http_requests_total[1m]))`

**Baseline**: Established from historical traffic patterns (typically reviewed weekly)

---

## Service Level Objectives (SLOs)

| SLI | Target | Window | Alert Threshold |
|-----|--------|--------|-----------------|
| Availability | 99.9% | 30 days | < 99.5% over 1 hour |
| Latency (P99) | < 1000ms | 30 days | > 1500ms over 5 minutes |
| Latency (P95) | < 500ms | 30 days | > 750ms over 5 minutes |
| Latency (P50) | < 100ms | 30 days | > 200ms over 5 minutes |

---

## Error Budget

The error budget represents the acceptable amount of failure within an SLO period.

### Calculation

```
Error Budget = 100% - SLO Target
Error Budget (30 days) = 100% - 99.9% = 0.1%
Error Budget (minutes) = 0.001 × 30 days × 24 hours × 60 minutes = 43.2 minutes
```

### Error Budget Policy

| Remaining Budget | Action |
|-----------------|--------|
| > 50% | Normal operations, deploy freely |
| 25-50% | Monitor closely, consider slowing deployments |
| 10-25% | Freeze non-critical deployments, prioritize reliability |
| < 10% | Deployment freeze, all hands on reliability improvements |

### Error Budget Tracking Query

```promql
# Availability error budget remaining (as percentage)
(
  1 - (
    (1 - (sum(rate(http_requests_total{status=~"2..|3.."}[30d])) / sum(rate(http_requests_total[30d])))) / 0.001
  )
) * 100
```

---

## Alerting Rules

Alerts are configured in `alerts.yml` with the following structure:

| Alert Name | SLO | Condition | Severity |
|------------|-----|-----------|----------|
| TaskAPIHighErrorRate | Availability | Error rate > 5% for 2m | Critical |
| TaskAPIHighLatency | Latency | P99 > 1s for 5m | Warning |
| TaskAPIDown | Availability | Target down for 1m | Critical |

### Burn Rate Alerting (Optional Advanced)

For more sophisticated SLO tracking, implement burn rate alerts:

```yaml
# Fast burn (2% budget in 1 hour)
- alert: TaskAPIErrorBudgetFastBurn
  expr: |
    (
      sum(rate(http_requests_total{status=~"5.."}[1h])) 
      / sum(rate(http_requests_total[1h]))
    ) > (14.4 * 0.001)  # 14.4x the acceptable error rate
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Error budget burning fast - 2% will be consumed in 1 hour"

# Slow burn (5% budget in 6 hours)  
- alert: TaskAPIErrorBudgetSlowBurn
  expr: |
    (
      sum(rate(http_requests_total{status=~"5.."}[6h])) 
      / sum(rate(http_requests_total[6h]))
    ) > (2.88 * 0.001)  # 2.88x the acceptable error rate
  for: 30m
  labels:
    severity: warning
  annotations:
    summary: "Error budget burning steadily - 5% will be consumed in 6 hours"
```

---

## Dashboard Integration

The Grafana dashboard (`task-api-dashboard.json`) includes:

1. **SLO Status Panel**: Shows current 30-day rolling availability percentage
2. **Error Budget Panel**: Displays remaining error budget as a gauge
3. **Latency Heatmap**: Visualizes latency distribution over time

### Key Dashboard Queries

```promql
# Current availability (30-day rolling)
sum(rate(http_requests_total{status=~"2..|3.."}[30d])) / sum(rate(http_requests_total[30d])) * 100

# Error budget remaining
(1 - ((1 - (sum(rate(http_requests_total{status=~"2..|3.."}[30d])) / sum(rate(http_requests_total[30d])))) / 0.001)) * 100

# Requests per second
sum(rate(http_requests_total[1m]))
```

---

## Review Cadence

| Activity | Frequency | Participants |
|----------|-----------|--------------|
| SLO Dashboard Review | Daily | On-call engineer |
| Error Budget Report | Weekly | Engineering team |
| SLO Target Review | Quarterly | Engineering + Product |
| Incident Post-mortem | Per incident | All stakeholders |

---

## Implementation Notes

### Why 99.9% (not 99.99% or 99.999%)?

- **99.9%** allows ~43 minutes downtime/month - appropriate for an internal API
- **99.99%** allows ~4 minutes downtime/month - requires significant infrastructure investment
- **99.999%** allows ~26 seconds downtime/month - impractical for most services

The 99.9% target balances reliability with development velocity and cost.

### Why 30-day window?

- Long enough to smooth out transient issues
- Short enough to drive meaningful action
- Aligns with monthly reporting cycles

### How to Adjust SLOs

1. **Gather data**: Review historical availability and latency metrics
2. **Assess impact**: Consider user experience and business requirements
3. **Propose change**: Document rationale in this file
4. **Get approval**: Engineering lead + Product sign-off
5. **Update**: Modify this document, alerts, and dashboards
6. **Communicate**: Notify all stakeholders of the change

---

## References

- Google SRE Book: [Service Level Objectives](https://sre.google/sre-book/service-level-objectives/)
- Prometheus Documentation: [Recording Rules](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/)
- Grafana Documentation: [Dashboard Best Practices](https://grafana.com/docs/grafana/latest/dashboards/best-practices/)
