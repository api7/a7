---
name: a7-plugin-zipkin
description: >-
  Skill for configuring the API7 Enterprise Edition (API7 EE) zipkin plugin via the a7 CLI.
  Covers distributed tracing with Zipkin, Jaeger, or any Zipkin-compatible
  collector, B3 propagation headers, sampling, and gateway group scoping.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: plugin
  apisix_version: ">=3.0.0"
  plugin_name: zipkin
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 config sync
---

# a7-plugin-zipkin

## Overview

The `zipkin` plugin sends distributed traces to Zipkin-compatible collectors
using the Zipkin v2 HTTP API. It supports B3 propagation headers for trace
context across services. Compatible backends include Zipkin, Jaeger, and
SkyWalking (via Zipkin receiver).

## When to Use

- Distributed tracing with Zipkin, Jaeger, or compatible collectors
- B3 header propagation across microservices
- Per-request sampling control via headers
- Trace ID injection into access logs

## Plugin Configuration Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `endpoint` | string | **Yes** | — | Zipkin collector URL (e.g. `http://zipkin:9411/api/v2/spans`) |
| `sample_ratio` | number | **Yes** | — | Sampling rate from 0.00001 to 1 |
| `service_name` | string | No | `"APISIX"` | Service name in Zipkin UI |
| `server_addr` | string | No | `$server_addr` | IPv4 address for span reporting |
| `span_version` | integer | No | `2` | Span format: 1 (legacy) or 2 (default) |

## B3 Propagation Headers

The plugin uses B3 propagation format:

### Injected to upstream

| Header | Description |
|--------|-------------|
| `x-b3-traceid` | Trace ID (16 or 32 hex chars) |
| `x-b3-spanid` | Span ID (16 hex chars) |
| `x-b3-parentspanid` | Parent span ID |
| `x-b3-sampled` | Sampling decision (1 or 0) |

### Extracted from client

| Header | Description |
|--------|-------------|
| `b3` | Single-header format: `{traceid}-{spanid}-{sampled}-{parentspanid}` |
| `x-b3-sampled` | `1` = force sample, `0` = skip, `d` = debug |
| `x-b3-flags` | `1` = force debug sampling |

Clients can override sampling per-request by setting `x-b3-sampled: 1`.

## Step-by-Step: Enable Zipkin Tracing

### 1. Create a route with zipkin

Enable tracing for gateway group `default`:

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "traced-api",
  "uri": "/api/*",
  "plugins": {
    "zipkin": {
      "endpoint": "http://zipkin:9411/api/v2/spans",
      "sample_ratio": 1,
      "service_name": "my-gateway",
      "span_version": 2
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

### 2. Send a request

```bash
curl http://localhost:9080/api/hello
```

### 3. View traces in Zipkin UI

Open the Zipkin UI and search for service `my-gateway`.

## Common Patterns

### Send traces to Jaeger

Jaeger supports the Zipkin v2 API:

```json
{
  "plugins": {
    "zipkin": {
      "endpoint": "http://jaeger-collector:9411/api/v2/spans",
      "sample_ratio": 1,
      "service_name": "my-gateway"
    }
  }
}
```

### Production sampling (10%)

```json
{
  "plugins": {
    "zipkin": {
      "endpoint": "http://zipkin:9411/api/v2/spans",
      "sample_ratio": 0.1,
      "service_name": "production-gateway"
    }
  }
}
```

### Enable globally via Global Rule

```bash
a7 global_rule create --gateway-group prod -f - <<'EOF'
{
  "id": "zipkin-global",
  "plugins": {
    "zipkin": {
      "endpoint": "http://zipkin:9411/api/v2/spans",
      "sample_ratio": 0.5,
      "service_name": "prod-gateway"
    }
  }
}
EOF
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| No traces in Zipkin UI | Wrong `endpoint` URL | Verify collector is reachable; must include `/api/v2/spans` |
| Traces not connected | B3 headers stripped | Ensure intermediate proxies forward `x-b3-*` headers |
| All requests sampled | `sample_ratio: 1` | Lower for production (e.g. 0.01-0.1) |
| 400 from collector | Span version mismatch | Try `span_version: 1` if collector only supports v1 |
| Config not applied | Wrong gateway group specified | Ensure `--gateway-group` matches the desired cluster |

## Config Sync Example

```yaml
version: "1"
gateway_group: default
routes:
  - id: traced-api
    uri: /api/*
    plugins:
      zipkin:
        endpoint: http://zipkin:9411/api/v2/spans
        sample_ratio: 1
        service_name: my-gateway
        span_version: 2
    upstream_id: my-upstream
upstreams:
  - id: my-upstream
    type: roundrobin
    nodes:
      "backend:8080": 1
```
