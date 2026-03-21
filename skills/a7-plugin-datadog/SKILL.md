---
name: a7-plugin-datadog
description: >-
  Skill for configuring the API7 Enterprise Edition datadog plugin via the a7 CLI.
  Covers pushing custom metrics to Datadog via DogStatsD, metric tags,
  batching, plugin metadata for global DogStatsD server config, and
  Datadog Agent integration.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: plugin
  apisix_version: ">=3.0.0"
  plugin_name: datadog
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 config sync
---

# a7-plugin-datadog

## Overview

The `datadog` plugin in API7 Enterprise Edition (API7 EE) pushes per-request metrics to a Datadog Agent via the
DogStatsD protocol (UDP). It reports request counts, latency, bandwidth,
and upstream timing with automatic tags for route, service, consumer,
status code, and more.

## When to Use

- Monitor API7 EE with Datadog APM and dashboards.
- Track request rates, latency, and error rates per route.
- Add custom tags for business-level metrics.
- Integrate with existing Datadog infrastructure.

## Plugin Configuration Reference (Route/Service)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `prefer_name` | boolean | No | `true` | Use route/service name instead of ID in tags |
| `include_path` | boolean | No | `false` | Include HTTP path pattern in tags |
| `include_method` | boolean | No | `false` | Include HTTP method in tags |
| `constant_tags` | array | No | `[]` | Static tags for this route (e.g. `["env:prod"]`) |
| `batch_max_size` | integer | No | `1000` | Max entries per batch |
| `inactive_timeout` | integer | No | `5` | Seconds before flushing batch |
| `buffer_duration` | integer | No | `60` | Max age of oldest entry |
| `max_retry_count` | integer | No | `0` | Retry attempts |

## Plugin Metadata (Global Configuration)

Set the DogStatsD server address for all routes in a gateway group:

```bash
# Get current token and server from context
TOKEN=$(a7 context current -o json | jq -r .token)
SERVER=$(a7 context current -o json | jq -r .server)
GROUP="default"

curl "${SERVER}/apisix/admin/plugin_metadata/datadog?gateway_group=${GROUP}" \
  -X PUT \
  -H "X-API-TOKEN: ${TOKEN}" \
  -d '{
    "host": "127.0.0.1",
    "port": 8125,
    "namespace": "api7ee",
    "constant_tags": ["source:api7-ee"]
  }'
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `host` | string | `"127.0.0.1"` | DogStatsD server host |
| `port` | integer | `8125` | DogStatsD server port |
| `namespace` | string | `"api7ee"` | Metric name prefix |
| `constant_tags` | array | `["source:api7-ee"]` | Global tags for all metrics |

## Metrics Emitted

| Metric | Type | Description |
|--------|------|-------------|
| `{namespace}.request.counter` | counter | Request count |
| `{namespace}.request.latency` | histogram | Total request latency (ms) |
| `{namespace}.upstream.latency` | histogram | Upstream response time (ms) |
| `{namespace}.apisix.latency` | histogram | API7 EE processing time (ms) |
| `{namespace}.ingress.size` | timer | Request body size (bytes) |
| `{namespace}.egress.size` | timer | Response body size (bytes) |

Default namespace is `api7ee`, so metrics appear as `api7ee.request.counter`.

## Automatic Tags

| Tag | Always Present | Description |
|-----|----------------|-------------|
| `route_name` | Yes | Route ID or name |
| `service_name` | If route has service | Service ID or name |
| `consumer` | If authenticated | Consumer username |
| `balancer_ip` | Yes | Upstream IP that handled the request |
| `response_status` | Yes | HTTP status code (e.g. `200`) |
| `response_status_class` | Yes | Status class (e.g. `2xx`, `5xx`) |
| `scheme` | Yes | `http`, `https`, `grpc`, `grpcs` |
| `path` | If `include_path: true` | HTTP path pattern |
| `method` | If `include_method: true` | HTTP method |

## Step-by-Step: Send Metrics to Datadog

### 1. Configure plugin metadata (DogStatsD address)

```bash
a7 plugin-metadata create datadog --gateway-group default -f - <<'EOF'
{
  "host": "127.0.0.1",
  "port": 8125,
  "namespace": "api7ee",
  "constant_tags": ["source:api7-ee", "env:production"]
}
EOF
```

### 2. Enable on a route

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "monitored-api",
  "name": "api-v1",
  "uri": "/api/v1/*",
  "plugins": {
    "datadog": {
      "prefer_name": true,
      "include_path": true,
      "include_method": true
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

### 3. Verify in Datadog

Open Datadog → Metrics Explorer → search for `api7ee.request.counter`.

## Common Patterns

### Custom constant tags per route

```json
{
  "plugins": {
    "datadog": {
      "prefer_name": true,
      "constant_tags": [
        "team:platform",
        "api_version:v2",
        "tier:premium"
      ]
    }
  }
}
```

### Remote Datadog Agent

```bash
a7 plugin-metadata update datadog --gateway-group default -f - <<'EOF'
{
  "host": "datadog-agent.internal",
  "port": 8125,
  "namespace": "mycompany",
  "constant_tags": ["source:api7-ee", "datacenter:us-east-1"]
}
EOF
```

## Datadog Dashboard Queries

```
# Request rate by route
sum:api7ee.request.counter{*} by {route_name}.as_count()

# P95 latency
percentile:api7ee.request.latency{*} by {route_name}, p:95

# Error rate
sum:api7ee.request.counter{response_status_class:5xx}.as_count()

# Upstream health by IP
avg:api7ee.upstream.latency{*} by {balancer_ip}
```

## Config Sync Example

```yaml
version: "1"
gateway_group: default
routes:
  - id: monitored-api
    name: api-v1
    uri: /api/v1/*
    plugins:
      datadog:
        prefer_name: true
        include_path: true
        include_method: true
        constant_tags:
          - "team:platform"
    upstream_id: my-upstream
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| No metrics in Datadog | Agent not receiving UDP | Check `host`/`port` in plugin metadata; verify Agent config |
| Missing consumer tag | No authentication on route | Tag only appears for authenticated requests |
| Wrong metric namespace | Default `api7ee` | Change `namespace` in plugin metadata |
| Tags rejected by Datadog | Invalid tag format | Tags must start with a letter, not end with `:` |
| Metrics delayed | Large `inactive_timeout` | Lower batch settings for faster delivery |
