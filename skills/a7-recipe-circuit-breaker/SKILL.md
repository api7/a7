---
name: a7-recipe-circuit-breaker
description: >-
  Recipe skill for implementing circuit breaker patterns using the a7 CLI in API7 Enterprise Edition.
  Covers the api-breaker plugin, unhealthy thresholds, healthy recovery,
  response code classification, and integration with service health checks.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: recipe
  apisix_version: ">=3.0.0"
  plugin_name: api-breaker
  a7_commands:
    - a7 service create
    - a7 route create
    - a7 route update
    - a7 route get
    - a7 config sync
---

# a7-recipe-circuit-breaker

## Overview

A circuit breaker prevents cascading failures by detecting unhealthy backend
responses and temporarily stopping requests to the failing service. API7 EE
implements this through the `api-breaker` plugin on routes.

Use the current service-backed route model:

1. Create a service that owns the upstream backend.
2. Create a route with `service_id`.
3. Enable `api-breaker` on the route.

## Plugin Configuration Reference

| Field | Required | Description |
|-------|----------|-------------|
| `break_response_code` | Yes | HTTP status returned when the circuit is open |
| `break_response_body` | No | Response body returned when open |
| `break_response_headers` | No | Headers returned when open |
| `unhealthy.http_statuses` | No | Upstream status codes counted as unhealthy |
| `unhealthy.failures` | No | Consecutive unhealthy responses before opening |
| `healthy.http_statuses` | No | Status codes counted as healthy for recovery |
| `healthy.successes` | No | Consecutive healthy responses before closing |
| `max_breaker_sec` | No | Maximum circuit-open duration |

## Step-by-Step: Enable Circuit Breaker

### 1. Create a protected service

```bash
a7 service create --gateway-group default -f - <<'EOF'
{
  "id": "backend-service",
  "name": "backend-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "backend", "port": 8080, "weight": 1}
    ]
  }
}
EOF
```

### 2. Create a route with `api-breaker`

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "protected-api",
  "name": "protected-api",
  "paths": ["/api/*"],
  "service_id": "backend-service",
  "plugins": {
    "api-breaker": {
      "break_response_code": 502,
      "unhealthy": {
        "http_statuses": [500, 502, 503],
        "failures": 3
      },
      "healthy": {
        "http_statuses": [200],
        "successes": 3
      },
      "max_breaker_sec": 300
    }
  }
}
EOF
```

After three consecutive 500/502/503 responses, the circuit opens and returns
502 immediately. After cooldown, API7 EE tests recovery and closes the circuit
after enough healthy responses.

### 3. Custom error response

```bash
a7 route update protected-api --gateway-group default -f - <<'EOF'
{
  "id": "protected-api",
  "name": "protected-api",
  "paths": ["/api/*"],
  "service_id": "backend-service",
  "plugins": {
    "api-breaker": {
      "break_response_code": 503,
      "break_response_body": "{\"error\": \"service temporarily unavailable\", \"retry_after\": 30}",
      "break_response_headers": [
        {"key": "Content-Type", "value": "application/json"},
        {"key": "Retry-After", "value": "30"}
      ],
      "unhealthy": {
        "http_statuses": [500, 502, 503, 504],
        "failures": 5
      },
      "healthy": {
        "http_statuses": [200, 201, 204],
        "successes": 2
      },
      "max_breaker_sec": 60
    }
  }
}
EOF
```

### 4. Combine with health checks

For production, define health checks on the service upstream and keep
`api-breaker` on the route. Health checks manage node health; the circuit
breaker protects this route from repeated upstream failures.

```bash
a7 service create --gateway-group default -f - <<'EOF'
{
  "id": "monitored-backend-service",
  "name": "monitored-backend-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "backend-1", "port": 8080, "weight": 1},
      {"host": "backend-2", "port": 8080, "weight": 1}
    ],
    "checks": {
      "active": {
        "type": "http",
        "http_path": "/health",
        "healthy": {"interval": 5, "successes": 2},
        "unhealthy": {"interval": 3, "http_failures": 3}
      }
    }
  }
}
EOF

a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "api",
  "name": "api",
  "paths": ["/api/*"],
  "service_id": "monitored-backend-service",
  "plugins": {
    "api-breaker": {
      "break_response_code": 503,
      "unhealthy": {
        "http_statuses": [500, 502, 503],
        "failures": 3
      },
      "healthy": {
        "http_statuses": [200],
        "successes": 3
      }
    }
  }
}
EOF
```

## Config Sync

```yaml
version: "1"
services:
  - id: backend-service
    name: backend-service
    upstream:
      type: roundrobin
      nodes:
        - host: backend
          port: 8080
          weight: 1
routes:
  - id: protected-api
    name: protected-api
    paths:
      - /api/*
    service_id: backend-service
    plugins:
      api-breaker:
        break_response_code: 503
        break_response_body: '{"error": "service unavailable"}'
        break_response_headers:
          - key: Content-Type
            value: application/json
          - key: Retry-After
            value: "30"
        unhealthy:
          http_statuses: [500, 502, 503]
          failures: 3
        healthy:
          http_statuses: [200]
          successes: 3
        max_breaker_sec: 300
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Circuit never opens | `unhealthy.http_statuses` misses the real error code | Add the actual upstream error codes |
| Circuit stays open too long | `max_breaker_sec` too high | Lower `max_breaker_sec` |
| Circuit flaps | Threshold too low for intermittent errors | Increase `unhealthy.failures` |
| API7 returns 502 outside breaker response | Backend is unreachable | Connection errors also count toward unhealthy thresholds |
| Recovery too slow | `healthy.successes` too high | Lower `healthy.successes` |
| Route not using breaker | Plugin attached to the wrong route | Verify with `a7 route get <id> -o json` |
| Command failed with 403 | RBAC permission issue | Ensure your token can modify routes in the gateway group |
