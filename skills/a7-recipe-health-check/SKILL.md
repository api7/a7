---
name: a7-recipe-health-check
description: >-
  Recipe skill for configuring backend health checks using the a7 CLI in API7 Enterprise Edition.
  Covers active health checks, passive health checks, combining both,
  healthy/unhealthy thresholds, and service-backed route wiring.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: recipe
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 service create
    - a7 service get
    - a7 route create
    - a7 route list
    - a7 config sync
---

# a7-recipe-health-check

## Overview

Health checks monitor backend nodes and remove unhealthy nodes from load
balancing. In current API7 EE usage, define the upstream and health check
configuration on a service, then attach routes to that service with
`service_id`.

API7 EE supports:

- Active checks: gateway probes each node.
- Passive checks: gateway observes real traffic responses.

Use both for production services that need automatic failure detection and
recovery.

## Health Check Configuration Reference

### Active Health Check

| Field | Description |
|-------|-------------|
| `upstream.checks.active.type` | `http`, `https`, or `tcp` |
| `upstream.checks.active.http_path` | HTTP path to probe |
| `upstream.checks.active.healthy.successes` | Consecutive successes to mark healthy |
| `upstream.checks.active.unhealthy.http_failures` | Consecutive HTTP failures to mark unhealthy |
| `upstream.checks.active.unhealthy.timeouts` | Consecutive timeouts to mark unhealthy |

### Passive Health Check

| Field | Description |
|-------|-------------|
| `upstream.checks.passive.type` | `http`, `https`, or `tcp` |
| `upstream.checks.passive.unhealthy.http_statuses` | Status codes treated as unhealthy |
| `upstream.checks.passive.unhealthy.http_failures` | Consecutive failures to mark unhealthy |
| `upstream.checks.passive.healthy.successes` | Consecutive successes to mark healthy |

## Step-by-Step: Configure Health Checks

### 1. Create a service with active HTTP health checks

```bash
a7 service create --gateway-group default -f - <<'EOF'
{
  "id": "backend-service",
  "name": "backend-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "backend-1", "port": 8080, "weight": 1},
      {"host": "backend-2", "port": 8080, "weight": 1},
      {"host": "backend-3", "port": 8080, "weight": 1}
    ],
    "checks": {
      "active": {
        "type": "http",
        "http_path": "/health",
        "healthy": {
          "interval": 5,
          "successes": 2,
          "http_statuses": [200]
        },
        "unhealthy": {
          "interval": 3,
          "http_failures": 3,
          "http_statuses": [500, 502, 503]
        }
      }
    }
  }
}
EOF
```

### 2. Create a route that uses the service

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "api",
  "name": "api",
  "paths": ["/api/*"],
  "service_id": "backend-service"
}
EOF
```

Health checks run for upstream nodes that are reachable through active
configuration. Verify route wiring with:

```bash
a7 service get backend-service --gateway-group default --output json
a7 route list --gateway-group default --service-id backend-service --output json
```

### 3. Passive health check

```bash
a7 service create --gateway-group default -f - <<'EOF'
{
  "id": "backend-passive-service",
  "name": "backend-passive-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "backend-1", "port": 8080, "weight": 1},
      {"host": "backend-2", "port": 8080, "weight": 1}
    ],
    "checks": {
      "passive": {
        "type": "http",
        "unhealthy": {
          "http_failures": 3,
          "http_statuses": [500, 502, 503],
          "timeouts": 3
        },
        "healthy": {
          "successes": 5,
          "http_statuses": [200, 201, 202, 203, 204]
        }
      }
    }
  }
}
EOF
```

Passive-only checks cannot recover a node that receives no traffic. Combine
passive checks with active checks for full recovery coverage.

### 4. Combined active + passive checks

```bash
a7 service create --gateway-group default -f - <<'EOF'
{
  "id": "production-backend-service",
  "name": "production-backend-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "backend-1", "port": 8080, "weight": 1},
      {"host": "backend-2", "port": 8080, "weight": 1},
      {"host": "backend-3", "port": 8080, "weight": 1}
    ],
    "checks": {
      "active": {
        "type": "http",
        "http_path": "/health",
        "healthy": {
          "interval": 5,
          "successes": 2,
          "http_statuses": [200]
        },
        "unhealthy": {
          "interval": 2,
          "http_failures": 3,
          "timeouts": 2,
          "http_statuses": [500, 502, 503, 504]
        }
      },
      "passive": {
        "type": "http",
        "unhealthy": {
          "http_failures": 3,
          "http_statuses": [500, 502, 503],
          "timeouts": 3
        },
        "healthy": {
          "successes": 3,
          "http_statuses": [200, 201, 204]
        }
      }
    }
  }
}
EOF
```

## Config Sync

```yaml
version: "1"
gateway_group: default
services:
  - id: production-backend-service
    name: production-backend-service
    upstream:
      type: roundrobin
      nodes:
        - host: backend-1
          port: 8080
          weight: 1
        - host: backend-2
          port: 8080
          weight: 1
        - host: backend-3
          port: 8080
          weight: 1
      checks:
        active:
          type: http
          http_path: /health
          healthy:
            interval: 5
            successes: 2
            http_statuses: [200]
          unhealthy:
            interval: 2
            http_failures: 3
            timeouts: 2
            http_statuses: [500, 502, 503, 504]
        passive:
          type: http
          unhealthy:
            http_failures: 3
            http_statuses: [500, 502, 503]
            timeouts: 3
          healthy:
            successes: 3
            http_statuses: [200, 201, 204]
routes:
  - id: api
    name: api
    paths:
      - /api/*
    service_id: production-backend-service
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Health checks not running | Route does not reference the service | Verify `service_id` with `a7 route get` |
| All nodes marked unhealthy | Health endpoint returns unexpected status | Verify `http_statuses` includes the response code |
| Node not recovering | Passive-only checks have no traffic to observe | Add active health checks |
| Probe hits wrong endpoint | Default `http_path` is `/` | Set `http_path` to the real health endpoint |
| TLS probe fails | Certificate verification fails | Set `https_verify_certificate: false` or fix certificates |
| No standalone health command | Current a7 does not expose upstream health status | Verify config with `a7 service get` and use gateway observability |
| Command failed with 401 | Invalid token | Refresh your token using `a7 context create` |
| Service not found | Different gateway group | Ensure `--gateway-group` matches where the service was created |
