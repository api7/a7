---
name: a7-recipe-blue-green
description: >-
  Recipe skill for implementing blue-green deployments using the a7 CLI in API7 Enterprise Edition.
  Covers creating two service-backed environments, switching traffic via route service_id updates,
  rollback procedures, and config sync workflows with gateway group scoping.
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
    - a7 route get
    - a7 config sync
    - a7 config diff
---

# a7-recipe-blue-green

## Overview

Blue-green deployment runs two identical production environments. Only one
environment serves live traffic at a time. Deploy the new version to the idle
environment, test it, then switch the route to the new service. If anything
goes wrong, switch the route back.

This recipe uses the current API7 EE service-backed route model:

1. Create one service for blue.
2. Create one service for green.
3. Point the route to the active service with `service_id`.
4. Switch or roll back by updating the route's `service_id`.

## Prerequisites

- API7 EE Control Plane and at least one Gateway Group.
- a7 CLI configured with a valid token and server address.
- Two deployable backend environments, such as blue and green.

## Approach A: Service Swap

### 1. Create both services

```bash
a7 service create --gateway-group default -f - <<'EOF'
{
  "id": "blue-service",
  "name": "blue-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "blue-backend-1", "port": 8080, "weight": 1},
      {"host": "blue-backend-2", "port": 8080, "weight": 1}
    ]
  }
}
EOF

a7 service create --gateway-group default -f - <<'EOF'
{
  "id": "green-service",
  "name": "green-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "green-backend-1", "port": 8080, "weight": 1},
      {"host": "green-backend-2", "port": 8080, "weight": 1}
    ]
  }
}
EOF
```

### 2. Create the route pointing to blue

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "api",
  "name": "api",
  "paths": ["/api/*"],
  "service_id": "blue-service"
}
EOF
```

### 3. Deploy and test green

Deploy the new version to the green environment, then test it directly through
its internal hostname or a temporary test route before switching production
traffic.

### 4. Switch to green

```bash
a7 route update api --gateway-group default -f - <<'EOF'
{
  "service_id": "green-service"
}
EOF
```

Traffic switches across all gateways in the `default` gateway group after the
configuration propagates.

### 5. Roll back to blue

```bash
a7 route update api --gateway-group default -f - <<'EOF'
{
  "service_id": "blue-service"
}
EOF
```

## Approach B: Header-Based Green Testing

Use `traffic-split` for targeted green testing while the route defaults to blue.
The route still uses `service_id` for the default backend; the plugin contains
an inline upstream only for matched green requests.

```bash
a7 route update api --gateway-group default -f - <<'EOF'
{
  "service_id": "blue-service",
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "match": [
            {"vars": [["http_x_env", "==", "green"]]}
          ],
          "weighted_upstreams": [
            {
              "upstream": {
                "type": "roundrobin",
                "nodes": [{"host": "green-backend-1", "port": 8080, "weight": 1}]
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

Test green with:

```bash
curl -H "x-env: green" http://gateway:9080/api/health
```

When ready, remove the plugin and switch the route to `green-service`:

```bash
a7 route update api --gateway-group default -f - <<'EOF'
{
  "plugins": {},
  "service_id": "green-service"
}
EOF
```

## Config Sync

```yaml
version: "1"
services:
  - id: blue-service
    name: blue-service
    upstream:
      type: roundrobin
      nodes:
        - host: blue-backend-1
          port: 8080
          weight: 1
        - host: blue-backend-2
          port: 8080
          weight: 1
  - id: green-service
    name: green-service
    upstream:
      type: roundrobin
      nodes:
        - host: green-backend-1
          port: 8080
          weight: 1
        - host: green-backend-2
          port: 8080
          weight: 1
routes:
  - id: api
    name: api
    paths:
      - /api/*
    service_id: blue-service # change to green-service to switch
```

Preview and apply:

```bash
a7 config diff -f config.yaml
a7 config sync -f config.yaml
```

## Deployment Script

```bash
#!/bin/bash
set -euo pipefail

GROUP="default"
ROUTE_ID="api"
CURRENT=$(a7 route get "$ROUTE_ID" --gateway-group "$GROUP" -o json | jq -r '.service_id')
TARGET=$([ "$CURRENT" = "blue-service" ] && echo "green-service" || echo "blue-service")

echo "Current: $CURRENT; switching to: $TARGET"

a7 route update "$ROUTE_ID" --gateway-group "$GROUP" -f - <<EOF
{"service_id": "$TARGET"}
EOF

if curl -sf http://gateway:9080/api/health > /dev/null; then
  echo "$TARGET is healthy"
else
  echo "$TARGET unhealthy; rolling back to $CURRENT"
  a7 route update "$ROUTE_ID" --gateway-group "$GROUP" -f - <<EOF
{"service_id": "$CURRENT"}
EOF
  exit 1
fi
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| 502 after switch | New environment not ready | Test health endpoint before switching; roll back if needed |
| Traffic still going to old env | Route update has not propagated yet | Verify with `a7 route get api --gateway-group "$GROUP" -o json` and retry after propagation |
| Cannot roll back | Previous service ID was not recorded | Record `service_id` before switching |
| Command failed with 401 | Invalid token | Refresh your token using `a7 context create` |
| Service not found | Different gateway group | Ensure `--gateway-group` matches where services and routes were created |
