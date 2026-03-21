---
name: a7-recipe-blue-green
description: >-
  Recipe skill for implementing blue-green deployments using the a7 CLI in API7 Enterprise Edition.
  Covers creating two upstream environments, switching traffic instantly
  via route updates or traffic-split plugin, rollback procedures, and
  config sync workflows with gateway group scoping.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: recipe
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 upstream create
    - a7 upstream update
    - a7 route create
    - a7 route update
    - a7 config sync
    - a7 config diff
---

# a7-recipe-blue-green

## Overview

Blue-green deployment runs two identical production environments (blue and
green). At any time, only one serves live traffic. Deploy the new version to
the idle environment, test it, then switch traffic instantly. If anything
goes wrong, switch back.

This recipe implements blue-green deployment using API7 Enterprise Edition (API7 EE) routes and upstreams
managed by the a7 CLI.

## When to Use

- Zero-downtime deployments with instant rollback.
- You have two identical environments that can be swapped.
- You want to test the new version with internal traffic before switching.
- You need an auditable, scriptable deployment process.

## Prerequisites

- API7 EE Control Plane and at least one Gateway Group.
- a7 CLI configured with a valid token and server address.

## Approach A: Upstream Swap (Simplest)

Switch traffic by updating the route's `upstream_id` to point at the other
environment.

### 1. Create both upstreams

```bash
a7 upstream create --gateway-group default -f - <<'EOF'
{
  "id": "blue",
  "type": "roundrobin",
  "nodes": {
    "blue-backend-1:8080": 1,
    "blue-backend-2:8080": 1
  }
}
EOF

a7 upstream create --gateway-group default -f - <<'EOF'
{
  "id": "green",
  "type": "roundrobin",
  "nodes": {
    "green-backend-1:8080": 1,
    "green-backend-2:8080": 1
  }
}
EOF
```

### 2. Create route pointing to blue

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "api",
  "uri": "/api/*",
  "upstream_id": "blue"
}
EOF
```

### 3. Deploy new version to green, test it

Deploy your new version to the green environment. Test internally using its direct IP/hostname.

### 4. Switch to green

```bash
a7 route update api --gateway-group default -f - <<'EOF'
{
  "upstream_id": "green"
}
EOF
```

Traffic switches instantly across all gateways in the `default` gateway group. No downtime.

### 5. Rollback to blue (if needed)

```bash
a7 route update api --gateway-group default -f - <<'EOF'
{
  "upstream_id": "blue"
}
EOF
```

## Approach B: Traffic-Split Plugin (Header-Based Testing)

Use the `traffic-split` plugin to test the green environment with specific
headers before full switch.

### 1. Create route with traffic-split

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "api",
  "uri": "/api/*",
  "plugins": {
    "traffic-split": {
      "rules": [
        {
          "match": [
            {
              "vars": [["http_x-env", "==", "green"]]
            }
          ],
          "weighted_upstreams": [
            {
              "upstream_id": "green",
              "weight": 1
            }
          ]
        }
      ]
    }
  },
  "upstream_id": "blue"
}
EOF
```

### 2. Test green internally

```bash
curl -H "x-env: green" http://gateway:9080/api/health
```

### 3. Full switch — remove traffic-split, swap upstream

```bash
a7 route update api --gateway-group default -f - <<'EOF'
{
  "plugins": {},
  "upstream_id": "green"
}
EOF
```

## Approach C: Config Sync (Declarative)

### config.yaml — Blue active

```yaml
version: "1"
gateway_group: default
upstreams:
  - id: blue
    type: roundrobin
    nodes:
      "blue-backend-1:8080": 1
      "blue-backend-2:8080": 1
  - id: green
    type: roundrobin
    nodes:
      "green-backend-1:8080": 1
      "green-backend-2:8080": 1
routes:
  - id: api
    uri: /api/*
    upstream_id: blue  # ← change to "green" to switch
```

### Preview changes before switching

```bash
# Edit config.yaml: change upstream_id to "green"
a7 config diff -f config.yaml
```

### Apply the switch

```bash
a7 config sync -f config.yaml
```

## Deployment Script Example

```bash
#!/bin/bash
set -euo pipefail

GROUP="default"
CURRENT=$(a7 route get api --gateway-group $GROUP -o json | jq -r '.upstream_id')
TARGET=$([ "$CURRENT" = "blue" ] && echo "green" || echo "blue")

echo "Current: $CURRENT → Switching to: $TARGET in group: $GROUP"

# Switch
a7 route update api --gateway-group $GROUP -f - <<EOF
{"upstream_id": "$TARGET"}
EOF

echo "Switched to $TARGET. Verifying..."

# Health check (assuming gateway IP is accessible)
if curl -sf http://gateway:9080/api/health > /dev/null; then
  echo "✅ $TARGET is healthy"
else
  echo "❌ $TARGET unhealthy, rolling back to $CURRENT"
  a7 route update api --gateway-group $GROUP -f - <<EOF
{"upstream_id": "$CURRENT"}
EOF
  exit 1
fi
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| 502 after switch | New environment not ready | Test health endpoint before switching; rollback if needed |
| Traffic still going to old env | Route cache or DNS | API7 EE routes update instantly — verify with `a7 route get` |
| Can't rollback | Lost track of previous upstream | Always record current state before switching |
| Connections drop during switch | Long-running requests on old upstream | API7 EE handles in-flight requests gracefully; existing connections complete |
| Command failed with 401 | Invalid token | Refresh your token using `a7 context create` |
| Upstream not found | Different gateway group | Ensure `--gateway-group` matches the group where upstreams were created |
