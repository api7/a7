---
name: a7-plugin-skywalking
description: >-
  Skill for configuring the API7 Enterprise Edition (API7 EE) skywalking plugin via the a7 CLI.
  Covers distributed tracing with Apache SkyWalking OAP, sampling
  configuration, service topology, and gateway group scoping.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: plugin
  apisix_version: ">=3.0.0"
  plugin_name: skywalking
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 config sync
---

# a7-plugin-skywalking

## Overview

The `skywalking` plugin integrates API7 EE with Apache SkyWalking for
distributed tracing. It creates entry and exit spans for each request,
reports them to SkyWalking OAP via HTTP, and enables service topology
visualization and performance analysis.

## When to Use

- Trace requests across microservices via SkyWalking
- Visualize service topology and dependency maps
- Analyze per-route and per-service latency
- Correlate traces with logs using `skywalking-logger`

## Plugin Configuration Reference (Route/Service)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `sample_ratio` | number | No | `1` | Sampling rate from 0.00001 to 1 (1 = trace all) |

## Global Configuration (Gateway Group)

In API7 EE, global settings like the SkyWalking endpoint are typically
configured at the gateway group level.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `service_name` | string | `"APISIX"` | Service name in SkyWalking UI |
| `service_instance_name` | string | `"APISIX Instance Name"` | Instance name (use `$hostname` for dynamic) |
| `endpoint_addr` | string | `http://127.0.0.1:12800` | SkyWalking OAP HTTP endpoint |
| `report_interval` | integer | `3` | Reporting interval in seconds |

## Step-by-Step: Enable SkyWalking Tracing

### 1. Ensure SkyWalking OAP is reachable

Verify your SkyWalking OAP server is running and accessible from the API7 EE
gateway nodes.

### 2. Configure gateway group settings

Configure the `skywalking` plugin attributes in your API7 EE gateway group.

### 3. Enable on a route

Enable tracing for gateway group `default`:

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "traced-api",
  "uri": "/api/*",
  "plugins": {
    "skywalking": {
      "sample_ratio": 1
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

### 4. Send a request and view traces

```bash
curl http://localhost:9080/api/hello
```

View traces in SkyWalking UI at the configured address.

## Common Patterns

### Partial sampling (production)

```json
{
  "plugins": {
    "skywalking": {
      "sample_ratio": 0.1
    }
  }
}
```

Traces 10% of requests. Sufficient for production traffic analysis without
excessive overhead.

### Trace-log correlation with skywalking-logger

```json
{
  "plugins": {
    "skywalking": {
      "sample_ratio": 1
    },
    "skywalking-logger": {
      "endpoint_addr": "http://skywalking-oap:12800"
    }
  }
}
```

Associates access logs with trace IDs in the SkyWalking UI.

### Enable globally via Global Rule

```bash
a7 global_rule create --gateway-group default -f - <<'EOF'
{
  "id": "skywalking-global",
  "plugins": {
    "skywalking": {
      "sample_ratio": 0.5
    }
  }
}
EOF
```

## Span Structure

The plugin creates two spans per request:

- **entrySpan**: From request arrival to response completion
- **exitSpan**: From upstream call start to response received

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| No traces in SkyWalking UI | Wrong `endpoint_addr` | Verify OAP is reachable from gateway nodes |
| Missing service in topology | `service_name` mismatch | Check service name in gateway group config |
| High overhead | `sample_ratio: 1` in production | Lower to 0.01-0.1 for high-traffic routes |
| Traces not correlated | Backend not instrumented | Install SkyWalking agent in upstream services |
| Config not applied | Wrong gateway group specified | Ensure `--gateway-group` matches the desired cluster |

## Config Sync Example

```yaml
version: "1"
gateway_group: default
routes:
  - id: traced-api
    uri: /api/*
    plugins:
      skywalking:
        sample_ratio: 1
    upstream_id: my-upstream
upstreams:
  - id: my-upstream
    type: roundrobin
    nodes:
      "backend:8080": 1
```
