---
name: a7-plugin-traffic-split
description: >-
  Skill for configuring the API7 Enterprise Edition (API7 EE) traffic-split plugin via the a7 CLI.
  Covers weighted traffic splitting between upstreams with conditional match
  rules. Includes canary release, blue-green deployment, A/B testing patterns,
  and gateway group scoping.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: plugin
  apisix_version: ">=3.0.0"
  plugin_name: traffic-split
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 route get
---

# a7-plugin-traffic-split

## Overview

The `traffic-split` plugin dynamically directs portions of traffic to
different upstream services based on custom rules (`match`) and weighted
distributions (`weighted_upstreams`). Use it for canary releases, blue-green
deployments, and A/B testing — all without modifying DNS or load balancers.

## When to Use

- Canary release: gradually shift traffic to a new version (10% → 50% → 100%)
- Blue-green deployment: switch traffic based on request headers or cookies
- A/B testing: split traffic by user attributes (headers, query params, cookies)
- Feature flags: route specific users to feature branches
- Multi-version API: run multiple backend versions simultaneously

## Plugin Configuration Reference (Route/Service)

### Top-level

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `rules` | array[object] | Yes | — | List of traffic splitting rules. Each rule has optional `match` and required `weighted_upstreams`. |

### rules[].match

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `match` | array[object] | No | `[]` | Conditions to activate this rule. Empty = unconditional (all traffic uses weights). |
| `match[].vars` | array[array] | No | — | Variable expressions: `["variable", "operator", "value"]`. Uses Nginx variables. Multiple vars in one object = AND. Multiple objects in match = OR. |

**Operators**: `==`, `~=`, `>`, `<`, `>=`, `<=`, `~~` (regex match), `!~~`, `in`, `has`, `!` — see [lua-resty-expr](https://github.com/api7/lua-resty-expr#operator-list).

**Common variables**: `arg_name` (query param), `http_header-name` (request header), `cookie_name` (cookie value).

### rules[].weighted_upstreams[]

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `upstream` | object | No | — | Inline upstream configuration (see below). |
| `weight` | integer | No | `1` | Traffic weight for this upstream. |

**If only `weight` is set** (no inline `upstream`), traffic goes to the route's service upstream.

### Inline upstream object

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | No | `"roundrobin"` | Load balancing: `"roundrobin"` or `"chash"`. |
| `nodes` | array | Yes | — | Backend nodes as `[{host, port, weight}]`. |
| `timeout` | object | No | `15` (seconds) | `{"connect": N, "send": N, "read": N}` |
| `pass_host` | string | No | `"pass"` | `"pass"` = client host, `"node"` = upstream node, `"rewrite"` = use `upstream_host`. |
| `upstream_host` | string | No | — | Custom Host header. Only works with `pass_host: "rewrite"`. |
| `name` | string | No | — | Human-readable name for the upstream. |

**Not supported in inline weighted upstreams**: `service_name`, `discovery_type`, `checks`, `retries`, and `retry_timeout`. Keep shared upstream behavior on the bound service whenever possible.

## Step-by-Step: Enable traffic-split on a Route

### 1. Canary release — 20% to new version

Configure a 20/80 split for gateway group `default`:

```bash
a7 service create --gateway-group default -f - <<'EOF'
{
  "id": "stable-api-service",
  "name": "stable-api-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [{"host": "backend-v1", "port": 8080, "weight": 1}]
  }
}
EOF

a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "canary-release",
  "uri": "/api/*",
  "service_id": "stable-api-service",
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "weighted_upstreams": [
            {
              "upstream": {
                "name": "new-version-v2",
                "type": "roundrobin",
                "nodes": [{"host": "backend-v2", "port": 8080, "weight": 1}]
              },
              "weight": 2
            },
            {
              "weight": 8
            }
          ]
        }
      ]
    }
  }
}
EOF
```

Result: 20% traffic → `backend-v2`, 80% → `backend-v1` (route default).

### 2. Blue-green deployment — header-based switching

```bash
a7 service create --gateway-group prod -f - <<'EOF'
{
  "id": "blue-api-service",
  "name": "blue-api-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [{"host": "blue-backend", "port": 8080, "weight": 1}]
  }
}
EOF

a7 route create --gateway-group prod -f - <<'EOF'
{
  "id": "blue-green",
  "uri": "/api/*",
  "service_id": "blue-api-service",
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "match": [
            {
              "vars": [
                ["http_x-canary", "==", "true"]
              ]
            }
          ],
          "weighted_upstreams": [
            {
              "upstream": {
                "name": "green-env",
                "type": "roundrobin",
                "nodes": [{"host": "green-backend", "port": 8080, "weight": 1}]
              },
              "weight": 1
            }
          ]
        }
      ]
    }
  }
}
EOF
```

Result: Requests with header `x-canary: true` → green, all others → blue.

### 3. Increase canary to 50%

```bash
a7 route update canary-release --gateway-group default -f - <<'EOF'
{
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "weighted_upstreams": [
            {
              "upstream": {
                "name": "new-version-v2",
                "type": "roundrobin",
                "nodes": [{"host": "backend-v2", "port": 8080, "weight": 1}]
              },
              "weight": 5
            },
            {
              "weight": 5
            }
          ]
        }
      ]
    }
  }
}
EOF
```

## Common Patterns

### A/B testing by query parameter

```json
{
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "match": [
            {
              "vars": [
                ["arg_variant", "==", "B"]
              ]
            }
          ],
          "weighted_upstreams": [
            {
              "upstream": {
                "name": "variant-B",
                "type": "roundrobin",
                "nodes": [{"host": "variant-b", "port": 8080, "weight": 1}]
              }
            }
          ]
        }
      ]
    }
  }
}
```

Requests with `?variant=B` → variant B backend.

### Multi-rule routing (OR logic)

```json
{
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "match": [{"vars": [["http_x-api-id", "==", "1"]]}],
          "weighted_upstreams": [
            {"upstream": {"type": "roundrobin", "nodes": [{"host": "svc-a", "port": 8080, "weight": 1}]}}
          ]
        },
        {
          "match": [{"vars": [["http_x-api-id", "==", "2"]]}],
          "weighted_upstreams": [
            {"upstream": {"type": "roundrobin", "nodes": [{"host": "svc-b", "port": 8080, "weight": 1}]}}
          ]
        }
      ]
    }
  }
}
```

### Health Check Considerations

```json
{
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "weighted_upstreams": [
            {
              "upstream": {"type": "roundrobin", "nodes": [{"host": "canary", "port": 8080, "weight": 1}]},
              "weight": 2
            },
            {
              "weight": 8
            }
          ]
        }
      ]
    }
  }
}
```

Current API7 EE does not expose standalone upstream CRUD through `a7`. Prefer
inline weighted upstreams for traffic splitting, and keep health-check settings
on service upstream configuration when needed.

## Match Logic Reference

| Structure | Logic |
|-----------|-------|
| Multiple entries in one `vars` array | **AND** — all must match |
| Multiple objects in `match` array | **OR** — any can match |
| Empty `match` or no `match` | **Unconditional** — always applies weights |

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Traffic ratio inaccurate | Round-robin algorithm causes slight deviation | Expected behavior; ratios converge over many requests |
| Match rule not triggering | Variable name wrong or operator mismatch | Use `http_header-name` for headers, `arg_name` for query params |
| Health checks not working | Health checks are not configured on the service upstream | Move stable backend health-check settings to the service upstream configuration |
| All traffic going to default | Match conditions never true | Debug with `a7 route get` and verify header/param names |
| Weight 0 not blocking traffic | Weight 0 means "never forward" to that upstream | Correct — set weight to 0 to exclude an upstream |
| Config not applied | Wrong gateway group specified | Ensure `--gateway-group` matches the desired cluster |

## Config Sync Example

```yaml
version: "1"
services:
  - id: stable-api-service
    name: stable-api-service
    upstream:
      type: roundrobin
      nodes:
        - host: stable-backend
          port: 8080
          weight: 1
routes:
  - id: canary-api
    uri: /api/*
    service_id: stable-api-service
    plugins:
      traffic-split:
        rules:
          - weighted_upstreams:
              - upstream:
                  type: roundrobin
                  nodes:
                    - host: canary-backend
                      port: 8080
                      weight: 1
                weight: 2
              - weight: 8
```
