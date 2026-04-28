---
name: a7-recipe-api-versioning
description: >-
  Recipe skill for implementing API versioning strategies using API7 Enterprise Edition (API7 EE)
  and the a7 CLI. Covers URI path versioning, header-based versioning, traffic splitting
  for gradual migration, and version lifecycle management.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: recipe
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 service create
    - a7 route create
    - a7 route update
    - a7 route list
    - a7 config sync
    - a7 gateway-group get
---

# a7-recipe-api-versioning

## Overview

API versioning keeps old clients working while new versions are introduced.
In the current API7 EE model, create one service per backend version and attach
routes to those services with `service_id`.

Supported patterns:

1. URI path versioning: `/v1/service`, `/v2/service`.
2. Header-based versioning: `X-API-Version: 2`.
3. Gradual rollout: send a small percentage to the new version.
4. Version deprecation: redirect or return a controlled response.

## Approach A: URI Path Versioning

### 1. Create versioned services

```bash
a7 service create -g production -f - <<'EOF'
{
  "id": "service-v1",
  "name": "Service V1",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "v1-backend", "port": 8080, "weight": 1}
    ]
  }
}
EOF

a7 service create -g production -f - <<'EOF'
{
  "id": "service-v2",
  "name": "Service V2",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "v2-backend", "port": 8080, "weight": 1}
    ]
  }
}
EOF
```

### 2. Create routes with URI rewriting

```bash
a7 route create -g production -f - <<'EOF'
{
  "id": "route-v1",
  "name": "route-v1",
  "paths": ["/v1/*"],
  "service_id": "service-v1",
  "plugins": {
    "proxy-rewrite": {
      "regex_uri": ["^/v1/(.*)", "/$1"]
    }
  }
}
EOF

a7 route create -g production -f - <<'EOF'
{
  "id": "route-v2",
  "name": "route-v2",
  "paths": ["/v2/*"],
  "service_id": "service-v2",
  "plugins": {
    "proxy-rewrite": {
      "regex_uri": ["^/v2/(.*)", "/$1"]
    }
  }
}
EOF
```

## Approach B: Header-Based Versioning

The route defaults to V1 through `service_id`. Matching requests can be routed
to V2 with `traffic-split` and an inline upstream.

```bash
a7 route create -g production -f - <<'EOF'
{
  "id": "versioned-api",
  "name": "versioned-api",
  "paths": ["/api/resource"],
  "service_id": "service-v1",
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "match": [
            {"vars": [["http_x_api_version", "==", "2"]]}
          ],
          "weighted_upstreams": [
            {
              "upstream": {
                "type": "roundrobin",
                "nodes": {"v2-backend:8080": 1}
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

Requests with `X-API-Version: 2` go to V2. Other requests stay on V1.

## Approach C: Gradual Version Rollout

Shift a small percentage of V1 traffic to V2 before making V2 the default.

```bash
a7 route update route-v1 -g production -f - <<'EOF'
{
  "service_id": "service-v1",
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "weighted_upstreams": [
            {
              "upstream": {
                "type": "roundrobin",
                "nodes": {"v2-backend:8080": 1}
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

The entry without `upstream` falls back to the route's default service.

## Version Deprecation with Redirect

```bash
a7 route update route-v1 -g production -f - <<'EOF'
{
  "paths": ["/v1/*"],
  "service_id": "service-v1",
  "plugins": {
    "redirect": {
      "regex_uri": ["^/v1/(.*)", "/v2/$1"],
      "ret_code": 301
    }
  }
}
EOF
```

## Config Sync

```yaml
version: "1"
gateway_group: production
services:
  - id: service-v1
    name: Service V1
    upstream:
      type: roundrobin
      nodes:
        - host: v1-svc
          port: 80
          weight: 1
  - id: service-v2
    name: Service V2
    upstream:
      type: roundrobin
      nodes:
        - host: v2-svc
          port: 80
          weight: 1
routes:
  - id: service-v1-route
    name: service-v1-route
    paths:
      - /v1/*
    service_id: service-v1
    plugins:
      proxy-rewrite:
        regex_uri: ["^/v1/(.*)", "/$1"]
  - id: service-v2-route
    name: service-v2-route
    paths:
      - /v2/*
    service_id: service-v2
    plugins:
      proxy-rewrite:
        regex_uri: ["^/v2/(.*)", "/$1"]
```

Apply the configuration:

```bash
a7 config sync -g production -f versioning-config.yaml
```

## Verification

```bash
curl -i https://gateway.prod.example.com/v1/health
curl -i https://gateway.prod.example.com/v2/health
curl -i -H "X-API-Version: 2" https://gateway.prod.example.com/api/resource
a7 route list -g production --service-id service-v1
```

## Important Considerations

- Always apply services and routes to the intended `--gateway-group`.
- Use route priorities when multiple versioning strategies overlap.
- `regex_uri` follows APISIX/Lua-compatible regex behavior.
- Prefer `service_id` for default routing and reserve inline upstreams for plugin-specific overrides.
