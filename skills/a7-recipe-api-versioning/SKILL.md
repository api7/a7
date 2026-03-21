---
name: a7-recipe-api-versioning
description: >-
  Recipe skill for implementing API versioning strategies using API7 Enterprise Edition (API7 EE)
  and the a7 CLI. Covers URI path versioning, header-based versioning, traffic splitting 
  for gradual migration, and enterprise version lifecycle management.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: recipe
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 upstream create
    - a7 config sync
    - a7 gateway-group get
---

# a7-recipe-api-versioning

## Overview

API versioning is critical for maintaining backward compatibility while evolving your services. API7 Enterprise Edition (API7 EE) provides sophisticated traffic control mechanisms to manage multiple API versions across different **Gateway Groups**.

Strategies supported:
1. **URI Path Versioning**: `/v1/service`, `/v2/service` (most common)
2. **Header-Based Versioning**: `X-API-Version: 2` or `Accept` header
3. **Query Parameter Versioning**: `?v=2`
4. **Gradual Rollout (Canary/Blue-Green)**: Shift traffic between versions based on weights
5. **Version Deprecation**: Graceful retirement with redirects or custom error messages

## When to Use

- Launching a new version of an API with breaking changes.
- Managing "Legacy", "Stable", and "Beta" versions simultaneously.
- Phasing out old versions by gradually shifting traffic.
- Providing specific versions to different **Gateway Groups** or client tiers.

## Approach A: URI Path Versioning (with Gateway Groups)

Each version is routed to its respective upstream, and the version prefix is stripped before reaching the backend.

### 1. Create Versioned Upstreams

```bash
# Upstream for v1 in the "production" gateway group
a7 upstream create -g production -f - <<'EOF'
{
  "id": "upstream-v1",
  "name": "Service V1",
  "type": "roundrobin",
  "nodes": { "v1-backend:8080": 1 }
}
EOF

# Upstream for v2 in the "production" gateway group
a7 upstream create -g production -f - <<'EOF'
{
  "id": "upstream-v2",
  "name": "Service V2",
  "type": "roundrobin",
  "nodes": { "v2-backend:8080": 1 }
}
EOF
```

### 2. Create Routes with URI Rewriting

```bash
# Route for v1: /v1/users -> /users on backend
a7 route create -g production -f - <<'EOF'
{
  "id": "route-v1",
  "uri": "/v1/*",
  "upstream_id": "upstream-v1",
  "plugins": {
    "proxy-rewrite": {
      "regex_uri": ["^/v1/(.*)", "/$1"]
    }
  }
}
EOF

# Route for v2: /v2/users -> /users on backend
a7 route create -g production -f - <<'EOF'
{
  "id": "route-v2",
  "uri": "/v2/*",
  "upstream_id": "upstream-v2",
  "plugins": {
    "proxy-rewrite": {
      "regex_uri": ["^/v2/(.*)", "/$1"]
    }
  }
}
EOF
```

## Approach B: Header-Based Versioning

A single URI serves multiple versions based on the `X-API-Version` header using the `traffic-split` plugin.

```bash
a7 route create -g production -f - <<'EOF'
{
  "uri": "/api/resource",
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "match": [
            { "vars": [["http_x_api_version", "==", "2"]] }
          ],
          "weighted_upstreams": [
            {
              "upstream": {
                "type": "roundrobin",
                "nodes": { "v2-backend:8080": 1 }
              },
              "weight": 1
            }
          ]
        }
      ]
    }
  },
  "upstream": {
    "type": "roundrobin",
    "nodes": { "v1-backend:8080": 1 }
  }
}
EOF
```

- Header `X-API-Version: 2` routes to V2.
- Missing header or other values route to V1 (default upstream).

## Approach C: Gradual Version Rollout

Shift 10% of traffic to V2 to monitor stability before a full rollout.

```bash
a7 route update route-v1 -g production -f - <<'EOF'
{
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "weighted_upstreams": [
            {
              "upstream": {
                "type": "roundrobin",
                "nodes": { "v2-backend:8080": 1 }
              },
              "weight": 1
            },
            {
              "weight": 9
            }
          ]
        }
      ]
    }
  }
}
EOF
```
*Note: A `weighted_upstreams` entry without an `upstream` field refers back to the route's default upstream (V1).*

## Version Deprecation with Redirect

When V1 is deprecated, redirect clients to the V2 equivalent with a `301` status code.

```bash
a7 route update route-v1 -g production -f - <<'EOF'
{
  "uri": "/v1/*",
  "plugins": {
    "redirect": {
      "regex_uri": ["^/v1/(.*)", "/v2/$1"],
      "ret_code": 301
    }
  }
}
EOF
```

## Declarative Versioning with Config Sync

Manage the lifecycle of versions across Gateway Groups in a single YAML file.

```yaml
# versioning-config.yaml
gateway_groups:
  - id: production
    upstreams:
      - id: v1-backend
        nodes: { "v1-svc:80": 1 }
      - id: v2-backend
        nodes: { "v2-svc:80": 1 }
    routes:
      - id: service-v1
        uri: "/v1/*"
        upstream_id: v1-backend
        plugins:
          proxy-rewrite:
            regex_uri: ["^/v1/(.*)", "/$1"]
      - id: service-v2
        uri: "/v2/*"
        upstream_id: v2-backend
        plugins:
          proxy-rewrite:
            regex_uri: ["^/v2/(.*)", "/$1"]
```

Apply the configuration:
```bash
a7 config sync -g production -f versioning-config.yaml
```

## Important Considerations

- **Gateway Group Scoping**: Always ensure your versioned routes are applied to the correct `--gateway-group`.
- **Admin API Port**: Use port `7443` (HTTPS) for API7 EE communication.
- **Authentication**: Use the `--token` flag for CLI authentication.
- **Priority**: When using multiple versioning strategies on the same URI, the order of rules in `traffic-split` is critical.
- **Regex Syntax**: `regex_uri` follows PCRE-compatible Lua regex. Double-escape special characters in JSON if necessary.

## Verification

```bash
# Verify V1 path
curl -i https://gateway.prod.example.com/v1/health

# Verify V2 path
curl -i https://gateway.prod.example.com/v2/health

# Verify Header-based versioning
curl -i -H "X-API-Version: 2" https://gateway.prod.example.com/api/resource

# Check route status in Gateway Group
a7 route list -g production
```
