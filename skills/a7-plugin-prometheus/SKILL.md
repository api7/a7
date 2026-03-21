---
name: a7-plugin-prometheus
description: >-
  Skill for configuring the API7 Enterprise Edition (API7 EE) prometheus plugin via the a7 CLI.
  Covers enabling Prometheus metrics export on routes and globally, exposed
  metrics (HTTP status, latency, bandwidth, upstream health),
  and gateway group scoping.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: plugin
  apisix_version: ">=3.0.0"
  plugin_name: prometheus
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 config sync
---

# a7-plugin-prometheus

## Overview

The `prometheus` plugin exposes API7 EE metrics in Prometheus text format. It
tracks HTTP status codes, request latency, bandwidth, upstream health, and
system status. API7 EE includes a built-in metrics endpoint that can be
scraped by Prometheus; Grafana is used for visualization.

## When to Use

- Monitor request rates, error rates, and latency per route/service/consumer
- Track upstream health check status
- Observe system-wide performance and resource usage
- Build dashboards and alerts with Prometheus + Grafana

## Plugin Configuration Reference (Route/Service/Global Rule)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `prefer_name` | boolean | No | `false` | Use route/service name instead of ID in metric labels |

The plugin has minimal per-route config. Most configuration is managed via
API7 EE gateway group settings.

## Metrics Exported

### Core Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `apisix_http_status` | counter | HTTP status codes per route/service/consumer |
| `apisix_http_latency` | histogram | Request latency in ms (types: request, upstream, apisix) |
| `apisix_bandwidth` | counter | Bandwidth in bytes (types: ingress, egress) |
| `apisix_http_requests_total` | gauge | Total HTTP requests received |
| `apisix_nginx_http_current_connections` | gauge | Current connections by state |
| `apisix_upstream_status` | gauge | Upstream health (1=healthy, 0=unhealthy) |
| `apisix_node_info` | gauge | API7 EE node hostname and version |
| `apisix_shared_dict_capacity_bytes` | gauge | Shared memory capacity |
| `apisix_shared_dict_free_space_bytes` | gauge | Shared memory free space |

### API7 EE Built-in Metrics Endpoint

API7 Enterprise Edition provides a built-in metrics endpoint. By default,
this is available at `/apisix/prometheus/metrics` on the configured
prometheus port (usually `9091` or exposed via the data plane port `9080`).

## Step-by-Step: Enable Prometheus Metrics

### 1. Enable on a route

Enable metrics for gateway group `default`:

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "my-api",
  "uri": "/api/*",
  "plugins": {
    "prometheus": {
      "prefer_name": true
    }
  },
  "upstream": {
    "type": "roundrobin",
    "nodes": {
      "backend:8080": 1
    }
  }
}
EOF
```

### 2. Enable globally (all routes in a group)

Use a Global Rule to enable metrics for all routes in the `prod` group:

```bash
a7 global_rule create --gateway-group prod -f - <<'EOF'
{
  "id": "prometheus-global",
  "plugins": {
    "prometheus": {}
  }
}
EOF
```

### 3. Access metrics

Default endpoint: `http://<gateway-host>:9091/apisix/prometheus/metrics`

### 4. Configure Prometheus scrape

```yaml
# prometheus.yml
scrape_configs:
  - job_name: api7-ee
    scrape_interval: 15s
    static_configs:
      - targets: ['gateway-host:9091']
```

## Common Patterns

### Custom metric prefix and export port

These are typically configured in the API7 EE Dashboard or via gateway group
configuration.

### Extra labels on metrics

Extra labels can be added to capture additional context like upstream
addresses or specific header values.

### Custom histogram buckets

Default buckets for latency are: 1, 2, 5, 7, 10, 15, 20, 25, 30, 40, 50, 60,
70, 80, 90, 100, 200, 300, 400, 500, 1000, 2000, 5000, 10000, 30000, 60000 ms.

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| No metrics at endpoint | Plugin not enabled | Add `prometheus: {}` to route or global_rules |
| Metrics port unreachable | Export server not enabled | Check API7 EE gateway group settings |
| Missing route labels | `prefer_name: false` and route has no name | Set `prefer_name: true` and name your routes |
| High cardinality | Too many extra labels | Reduce `extra_labels` to avoid metric explosion |
| Config not applied | Wrong gateway group specified | Ensure `--gateway-group` matches the desired cluster |

## Config Sync Example

```yaml
version: "1"
gateway_group: default
global_rules:
  - id: prometheus-global
    plugins:
      prometheus:
        prefer_name: true
routes:
  - id: my-api
    uri: /api/*
    plugins:
      prometheus: {}
    upstream_id: my-upstream
```
