---
name: a7-recipe-canary
description: >-
  Recipe skill for implementing canary releases using the a7 CLI in API7 Enterprise Edition.
  Covers gradual traffic shifting with the traffic-split plugin, header-based canary routing,
  monitoring checkpoints, and full promotion or rollback workflows.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: recipe
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 service create
    - a7 service update
    - a7 route create
    - a7 route update
    - a7 route get
    - a7 config sync
---

# a7-recipe-canary

## Overview

A canary release gradually shifts traffic from a stable version to a new
version. Start with a small percentage, monitor, then increase until the new
version receives all traffic. If errors spike, roll back to the stable service.

This recipe uses:

- `service_id` for the route's default stable backend.
- `traffic-split` with an inline upstream for temporary canary traffic.
- `service update` or `route update` for promotion and rollback.

## Prerequisites

- API7 EE Control Plane and at least one Gateway Group.
- a7 CLI configured with a valid token and server address.
- Stable and canary backend deployments.

## Step-by-Step Canary Release

### 1. Create stable and canary services

```bash
a7 service create --gateway-group default -f - <<'EOF'
{
  "id": "stable-service",
  "name": "stable-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "stable-v1", "port": 8080, "weight": 1}
    ]
  }
}
EOF

a7 service create --gateway-group default -f - <<'EOF'
{
  "id": "canary-service",
  "name": "canary-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "canary-v2", "port": 8080, "weight": 1}
    ]
  }
}
EOF
```

### 2. Start canary at 5%

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "api",
  "name": "api",
  "paths": ["/api/*"],
  "service_id": "stable-service",
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "weighted_upstreams": [
            {
              "upstream": {
                "type": "roundrobin",
                "nodes": [{"host": "canary-v2", "port": 8080, "weight": 1}]
              },
              "weight": 5
            },
            {
              "weight": 95
            }
          ]
        }
      ]
    }
  }
}
EOF
```

### 3. Increase to 25%

```bash
a7 route update api --gateway-group default -f - <<'EOF'
{
  "service_id": "stable-service",
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "weighted_upstreams": [
            {
              "upstream": {
                "type": "roundrobin",
                "nodes": [{"host": "canary-v2", "port": 8080, "weight": 1}]
              },
              "weight": 25
            },
            {
              "weight": 75
            }
          ]
        }
      ]
    }
  }
}
EOF
```

### 4. Promote to 100%

Remove `traffic-split` and switch the route's default service to canary.

```bash
a7 route update api --gateway-group default -f - <<'EOF'
{
  "plugins": {},
  "service_id": "canary-service"
}
EOF
```

Optionally update the stable service to point to the promoted backend so the
same service IDs can be reused in the next release:

```bash
a7 service update stable-service --gateway-group default -f - <<'EOF'
{
  "id": "stable-service",
  "name": "stable-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "canary-v2", "port": 8080, "weight": 1}
    ]
  }
}
EOF
```

## Rollback

Remove the plugin and send all traffic back to the stable service:

```bash
a7 route update api --gateway-group default -f - <<'EOF'
{
  "plugins": {},
  "service_id": "stable-service"
}
EOF
```

## Header-Based Canary

```bash
a7 route update api --gateway-group default -f - <<'EOF'
{
  "service_id": "stable-service",
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "match": [
            {"vars": [["http_x_canary", "==", "true"]]}
          ],
          "weighted_upstreams": [
            {
              "upstream": {
                "type": "roundrobin",
                "nodes": [{"host": "canary-v2", "port": 8080, "weight": 1}]
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

Only requests with `x-canary: true` go to canary. All others use the route's
default `stable-service`.

## Canary Progression Script

```bash
#!/bin/bash
set -euo pipefail

GROUP="default"
ROUTE_ID="api"
WEIGHTS=(5 25 50 75 100)
HEALTH_URL="http://gateway:9080/api/health"
WAIT_SECONDS=300

for w in "${WEIGHTS[@]}"; do
  if [ "$w" -eq 100 ]; then
    a7 route update "$ROUTE_ID" --gateway-group "$GROUP" -f - <<EOF
{"plugins": {}, "service_id": "canary-service"}
EOF
  else
    stable_w=$((100 - w))
    a7 route update "$ROUTE_ID" --gateway-group "$GROUP" -f - <<EOF
{
  "service_id": "stable-service",
  "plugins": {
    "traffic-split": {
      "rules": [{
        "weighted_upstreams": [
          {"upstream": {"type": "roundrobin", "nodes": [{"host": "canary-v2", "port": 8080, "weight": 1}]}, "weight": $w},
          {"weight": $stable_w}
        ]
      }]
    }
  }
}
EOF
  fi

  sleep "$WAIT_SECONDS"
  if ! curl -sf "$HEALTH_URL" > /dev/null; then
    a7 route update "$ROUTE_ID" --gateway-group "$GROUP" -f - <<EOF
{"plugins": {}, "service_id": "stable-service"}
EOF
    exit 1
  fi
done
```

## Config Sync

```yaml
version: "1"
services:
  - id: stable-service
    name: stable-service
    upstream:
      type: roundrobin
      nodes:
        - host: stable-v1
          port: 8080
          weight: 1
  - id: canary-service
    name: canary-service
    upstream:
      type: roundrobin
      nodes:
        - host: canary-v2
          port: 8080
          weight: 1
routes:
  - id: api
    name: api
    paths:
      - /api/*
    service_id: stable-service
    plugins:
      traffic-split:
        rules:
          - weighted_upstreams:
              - upstream:
                  type: roundrobin
                  nodes:
                    - host: canary-v2
                      port: 8080
                      weight: 1
                weight: 10
              - weight: 90
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Traffic ratio is not exact | Weighted routing is probabilistic | Measure over enough requests |
| Canary not receiving traffic | Match condition never true | Check header/cookie name and `a7 route get api --gateway-group default` output |
| Rollback not instant | Config has not propagated yet | Verify with `a7 route get api --gateway-group default` and retry after propagation |
| 502 from canary | Canary backend is unhealthy | Check canary service health before increasing weight |
| Weight changes have no effect | Editing wrong route | Verify route ID with `a7 route list --gateway-group default --service-id stable-service` |
| Command failed with 403 | RBAC permission issue | Ensure your token can modify routes in the gateway group |
