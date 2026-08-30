#!/usr/bin/env python3
"""Generate Grafana dashboard JSON for Task API monitoring."""
import json
import os

# Get the directory where this script is located
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.dirname(SCRIPT_DIR)

panels = []

# Panel 1: Request Rate
panels.append({
    "datasource": {"type": "prometheus", "uid": "prometheus"},
    "fieldConfig": {
        "defaults": {
            "color": {"mode": "palette-classic"},
            "custom": {
                "axisCenteredZero": False,
                "axisColorMode": "text",
                "axisLabel": "",
                "axisPlacement": "auto",
                "barAlignment": 0,
                "drawStyle": "line",
                "fillOpacity": 10,
                "gradientMode": "none",
                "hideFrom": {"legend": False, "tooltip": False, "viz": False},
                "lineInterpolation": "linear",
                "lineWidth": 1,
                "pointSize": 5,
                "scaleDistribution": {"type": "linear"},
                "showPoints": "never",
                "spanNulls": False,
                "stacking": {"group": "A", "mode": "none"},
                "thresholdsStyle": {"mode": "off"}
            },
            "mappings": [],
            "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]},
            "unit": "reqps"
        },
        "overrides": []
    },
    "gridPos": {"h": 8, "w": 12, "x": 0, "y": 0},
    "id": 1,
    "options": {
        "legend": {"calcs": [], "displayMode": "list", "placement": "bottom", "showLegend": True},
        "tooltip": {"mode": "multi", "sort": "none"}
    },
    "targets": [{
        "datasource": {"type": "prometheus", "uid": "prometheus"},
        "editorMode": "code",
        "expr": "sum(rate(http_requests_total[1m])) by (method, path)",
        "legendFormat": "{{method}} {{path}}",
        "range": True,
        "refId": "A"
    }],
    "title": "Request Rate (req/s)",
    "type": "timeseries"
})

# Panel 2: Error Rate
panels.append({
    "datasource": {"type": "prometheus", "uid": "prometheus"},
    "fieldConfig": {
        "defaults": {
            "color": {"mode": "thresholds"},
            "mappings": [],
            "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}, {"color": "red", "value": 0.05}]},
            "unit": "percentunit"
        },
        "overrides": []
    },
    "gridPos": {"h": 8, "w": 12, "x": 12, "y": 0},
    "id": 2,
    "options": {
        "colorMode": "value",
        "graphMode": "area",
        "justifyMode": "auto",
        "orientation": "auto",
        "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
        "textMode": "auto"
    },
    "targets": [{
        "datasource": {"type": "prometheus", "uid": "prometheus"},
        "editorMode": "code",
        "expr": "sum(rate(http_requests_total{status=~\"5..\"}[1m])) / sum(rate(http_requests_total[1m]))",
        "range": True,
        "refId": "A"
    }],
    "title": "Error Rate (5xx)",
    "type": "stat"
})

# Panel 3: Latency Percentiles
panels.append({
    "datasource": {"type": "prometheus", "uid": "prometheus"},
    "fieldConfig": {
        "defaults": {
            "color": {"mode": "palette-classic"},
            "custom": {
                "axisCenteredZero": False,
                "axisColorMode": "text",
                "axisLabel": "",
                "axisPlacement": "auto",
                "barAlignment": 0,
                "drawStyle": "line",
                "fillOpacity": 10,
                "gradientMode": "none",
                "hideFrom": {"legend": False, "tooltip": False, "viz": False},
                "lineInterpolation": "linear",
                "lineWidth": 1,
                "pointSize": 5,
                "scaleDistribution": {"type": "linear"},
                "showPoints": "never",
                "spanNulls": False,
                "stacking": {"group": "A", "mode": "none"},
                "thresholdsStyle": {"mode": "off"}
            },
            "mappings": [],
            "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]},
            "unit": "s"
        },
        "overrides": []
    },
    "gridPos": {"h": 8, "w": 24, "x": 0, "y": 8},
    "id": 3,
    "options": {
        "legend": {"calcs": [], "displayMode": "list", "placement": "bottom", "showLegend": True},
        "tooltip": {"mode": "multi", "sort": "none"}
    },
    "targets": [
        {
            "datasource": {"type": "prometheus", "uid": "prometheus"},
            "editorMode": "code",
            "expr": "histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket[1m])) by (le))",
            "legendFormat": "P50",
            "range": True,
            "refId": "A"
        },
        {
            "datasource": {"type": "prometheus", "uid": "prometheus"},
            "editorMode": "code",
            "expr": "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[1m])) by (le))",
            "legendFormat": "P95",
            "range": True,
            "refId": "B"
        },
        {
            "datasource": {"type": "prometheus", "uid": "prometheus"},
            "editorMode": "code",
            "expr": "histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[1m])) by (le))",
            "legendFormat": "P99",
            "range": True,
            "refId": "C"
        }
    ],
    "title": "Latency Percentiles (P50/P95/P99)",
    "type": "timeseries"
})

# Panel 4: Total Tasks
panels.append({
    "datasource": {"type": "prometheus", "uid": "prometheus"},
    "fieldConfig": {
        "defaults": {
            "color": {"mode": "thresholds"},
            "mappings": [],
            "thresholds": {"mode": "absolute", "steps": [{"color": "blue", "value": None}]}
        },
        "overrides": []
    },
    "gridPos": {"h": 8, "w": 12, "x": 0, "y": 16},
    "id": 4,
    "options": {
        "colorMode": "value",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "auto",
        "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
        "textMode": "auto"
    },
    "targets": [{
        "datasource": {"type": "prometheus", "uid": "prometheus"},
        "editorMode": "code",
        "expr": "task_api_tasks_total",
        "range": True,
        "refId": "A"
    }],
    "title": "Total Tasks",
    "type": "stat"
})

# Panel 5: Completed Tasks
panels.append({
    "datasource": {"type": "prometheus", "uid": "prometheus"},
    "fieldConfig": {
        "defaults": {
            "color": {"mode": "thresholds"},
            "mappings": [],
            "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]}
        },
        "overrides": []
    },
    "gridPos": {"h": 8, "w": 12, "x": 12, "y": 16},
    "id": 5,
    "options": {
        "colorMode": "value",
        "graphMode": "none",
        "justifyMode": "auto",
        "orientation": "auto",
        "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
        "textMode": "auto"
    },
    "targets": [{
        "datasource": {"type": "prometheus", "uid": "prometheus"},
        "editorMode": "code",
        "expr": "task_api_tasks_done",
        "range": True,
        "refId": "A"
    }],
    "title": "Completed Tasks",
    "type": "stat"
})

dashboard = {
    "annotations": {"list": []},
    "editable": True,
    "fiscalYearStartMonth": 0,
    "graphTooltip": 0,
    "id": None,
    "links": [],
    "liveNow": False,
    "panels": panels,
    "refresh": "10s",
    "schemaVersion": 38,
    "style": "dark",
    "tags": ["task-api"],
    "templating": {"list": []},
    "time": {"from": "now-15m", "to": "now"},
    "timepicker": {},
    "timezone": "",
    "title": "Task API Dashboard",
    "uid": "task-api-dashboard",
    "version": 1,
    "weekStart": ""
}

# Output to the correct location in the project
output_dir = os.path.join(PROJECT_ROOT, "monitoring", "grafana", "provisioning", "dashboards")
os.makedirs(output_dir, exist_ok=True)
output_path = os.path.join(output_dir, "task-api-dashboard.json")

with open(output_path, "w") as f:
    json.dump(dashboard, f, indent=2)

print(f"Dashboard JSON written to {output_path}")
