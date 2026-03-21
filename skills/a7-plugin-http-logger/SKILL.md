---
name: a7-plugin-http-logger
description: >-
  Skill for configuring the API7 Enterprise Edition (API7 EE) http-logger plugin via the a7 CLI.
  Covers pushing access logs to HTTP/HTTPS endpoints in batches, custom log
  formats, and gateway group scoping.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: plugin
  apisix_version: ">=3.0.0"
  plugin_name: http-logger
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 config sync
---

# a7-plugin-http-logger

## Overview

The `http-logger` plugin pushes request/response logs as JSON to HTTP or
HTTPS endpoints. Logs are batched for efficiency and support custom formats
using NGINX variables. Use it to send structured logs to any HTTP-based
logging backend (Elasticsearch, Loki, custom APIs, etc.).

## When to Use

- Ship access logs to an HTTP-based logging backend
- Custom log formats with selected fields only
- Conditional request/response body capture
- Batch log delivery with retry on failure

## Plugin Configuration Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `uri` | string | **Yes** | — | HTTP/HTTPS endpoint for log delivery |
| `auth_header` | string | No | — | Authorization header value |
| `timeout` | integer | No | `3` | Connection timeout in seconds |
| `log_format` | object | No | — | Custom log format (supports `$variable` syntax) |
| `include_req_body` | boolean | No | `false` | Include request body in logs |
| `include_req_body_expr` | array | No | — | Conditional expression for request body logging |
| `include_resp_body` | boolean | No | `false` | Include response body in logs |
| `include_resp_body_expr` | array | No | — | Conditional expression for response body logging |
| `concat_method` | string | No | `"json"` | Batch format: `json` (array) or `new_line` (newline-separated) |
| `ssl_verify` | boolean | No | `false` | Verify SSL certificate for HTTPS endpoints |

### Batch Processing Parameters

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `batch_max_size` | integer | `1000` | Max entries per batch |
| `inactive_timeout` | integer | `5` | Seconds before flushing incomplete batch |
| `buffer_duration` | integer | `60` | Max age of oldest entry before forced flush |
| `max_retry_count` | integer | `0` | Retry attempts on failure |
| `retry_delay` | integer | `1` | Seconds between retries |

## Default Log Entry Format

When no custom `log_format` is set, each log entry contains:

```json
{
  "client_ip": "127.0.0.1",
  "route_id": "1",
  "start_time": 1703907485819,
  "latency": 101.9,
  "apisix_latency": 100.9,
  "upstream_latency": 1,
  "upstream": "127.0.0.1:8080",
  "request": {
    "method": "GET",
    "uri": "/api/users",
    "url": "http://127.0.0.1:9080/api/users",
    "size": 194,
    "headers": { "host": "...", "user-agent": "..." }
  },
  "response": {
    "status": 200,
    "size": 123,
    "headers": { "content-type": "...", "content-length": "..." }
  }
}
```

## Step-by-Step: Ship Logs to an HTTP Endpoint

### 1. Create a route with http-logger

Enable logging for gateway group `default`:

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "logged-api",
  "uri": "/api/*",
  "plugins": {
    "http-logger": {
      "uri": "http://log-collector:8080/logs",
      "batch_max_size": 100,
      "inactive_timeout": 10
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

### 2. Global logging for a gateway group

Apply a Global Rule to log all traffic in the `prod` group:

```bash
a7 global_rule create --gateway-group prod -f - <<'EOF'
{
  "id": "http-logger-global",
  "plugins": {
    "http-logger": {
      "uri": "http://log-collector:8080/global-logs",
      "batch_max_size": 500,
      "inactive_timeout": 30
    }
  }
}
EOF
```

## Common Patterns

### Custom log format with NGINX variables

```json
{
  "plugins": {
    "http-logger": {
      "uri": "http://log-collector:8080/logs",
      "log_format": {
        "@timestamp": "$time_iso8601",
        "client_ip": "$remote_addr",
        "host": "$host",
        "method": "$request_method",
        "uri": "$request_uri",
        "status": "$status",
        "latency": "$request_time"
      }
    }
  }
}
```

### Authenticated endpoint

```json
{
  "plugins": {
    "http-logger": {
      "uri": "https://log-service.example.com/api/v1/logs",
      "auth_header": "Bearer eyJhbGciOiJIUzI1NiIs...",
      "ssl_verify": true,
      "timeout": 5
    }
  }
}
```

### Conditional request body logging

Log request bodies only when a query parameter is present:

```json
{
  "plugins": {
    "http-logger": {
      "uri": "http://log-collector:8080/logs",
      "include_req_body": true,
      "include_req_body_expr": [
        ["arg_debug", "==", "true"]
      ]
    }
  }
}
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| No logs arriving | Wrong `uri` or endpoint down | Verify endpoint is reachable from gateway nodes |
| SSL handshake failure | Certificate not trusted | Set `ssl_verify: false` for self-signed certs |
| Logs delayed | Large `inactive_timeout` | Lower `inactive_timeout` for faster delivery |
| Logs dropped | Buffer overflow | Increase `batch_max_size`; reduce delivery latency |
| Auth rejected | Wrong `auth_header` value | Include full header value (e.g. `Bearer <token>`) |
| Config not applied | Wrong gateway group specified | Ensure `--gateway-group` matches the desired cluster |

## Config Sync Example

```yaml
version: "1"
gateway_group: default
routes:
  - id: logged-api
    uri: /api/*
    plugins:
      http-logger:
        uri: http://log-collector:8080/logs
        batch_max_size: 200
        inactive_timeout: 10
        log_format:
          timestamp: "$time_iso8601"
          client_ip: "$remote_addr"
          method: "$request_method"
          status: "$status"
    upstream_id: my-upstream
```
