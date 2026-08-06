---
name: a7-recipe-multi-tenant
description: >-
  Recipe skill for implementing multi-tenant patterns using API7 Enterprise Edition (API7 EE)
  and the a7 CLI. Covers gateway-group isolation, consumer policies,
  service-backed tenant routes, and credential-based tenant access.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: recipe
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 gateway-group create
    - a7 global-rule create
    - a7 consumer create
    - a7 consumer list
    - a7 credential create
    - a7 service create
    - a7 service get
    - a7 route create
    - a7 route get
    - a7 config sync
---

# a7-recipe-multi-tenant

## Overview

Multi-tenancy in API7 EE is built from three layers:

1. Gateway groups for runtime isolation.
2. Consumers and credentials for tenant identity.
3. Service-backed routes for tenant APIs.

For route traffic, use the current a7 model:

1. Create a service with upstream nodes.
2. Create routes with `paths` and `service_id`.
3. Create consumers and credentials separately.

## When to Use

- Separate `dev`, `staging`, and `prod` gateway configurations.
- Serve SaaS tenants with different limits or auth policies.
- Isolate regulated or high-priority tenants by gateway group.
- Let platform teams own shared routing while app teams own service targets.

## Approach A: Gateway Groups for Isolation

Gateway groups are the primary isolation boundary. With `jq` installed, create
the groups and capture the IDs returned by API7 EE:

```bash
PREMIUM_GROUP_ID=$(a7 gateway-group create --name premium-tier --description "High-performance tier for paid customers" --output json | jq -r '.id')
STANDARD_GROUP_ID=$(a7 gateway-group create --name standard-tier --description "Standard tier for free and trial users" --output json | jq -r '.id')
PLATFORM_GROUP_ID=$(a7 gateway-group create --name platform --description "Shared platform gateway group for tenant consumers and routes" --output json | jq -r '.id')
```

API7 EE generates gateway-group IDs. Keep these variables in the current shell
and pass the generated IDs, rather than the display names, to runtime-resource
commands.

Each group can have its own global policies:

```bash
a7 global-rule create -g "$STANDARD_GROUP_ID" -f - <<'EOF'
{
  "plugins": {
    "limit-count": {
      "count": 5000,
      "time_window": 3600,
      "rejected_code": 429
    }
  }
}
EOF
```

## Approach B: Consumers and Credentials

Current API7 EE does not expose consumer group management through the Admin API.
Model tenants as consumers, attach per-consumer plugins when needed, and create
credentials with `a7 credential create`.

```bash
a7 consumer create -g "$PLATFORM_GROUP_ID" -f - <<'EOF'
{
  "username": "startup-xyz",
  "desc": "Free tier tenants",
  "plugins": {
    "limit-count": {
      "count": 100,
      "time_window": 86400,
      "key_type": "var",
      "key": "consumer_name",
      "rejected_code": 429,
      "rejected_msg": "Free tier quota exceeded"
    }
  }
}
EOF

a7 credential create -g "$PLATFORM_GROUP_ID" --consumer startup-xyz --plugins-json '{"key-auth":{"key":"startup-xyz-key"}}'

a7 consumer create -g "$PLATFORM_GROUP_ID" -f - <<'EOF'
{
  "username": "acme-corp",
  "desc": "Pro tier tenants",
  "plugins": {
    "limit-count": {
      "count": 10000,
      "time_window": 86400,
      "key_type": "var",
      "key": "consumer_name",
      "rejected_code": 429,
      "rejected_msg": "Pro tier quota exceeded"
    }
  }
}
EOF

a7 credential create -g "$PLATFORM_GROUP_ID" --consumer acme-corp --plugins-json '{"key-auth":{"key":"acme-secret-key"}}'
```

## Approach C: Tenant-Aware Service Route

Create the backend service first:

```bash
a7 service create -g "$PLATFORM_GROUP_ID" -f - <<'EOF'
{
  "id": "tenant-api-service",
  "name": "tenant-api-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "internal-service", "port": 8080, "weight": 1}
    ]
  }
}
EOF
```

Create the tenant route with `paths` and `service_id`:

```bash
a7 route create -g "$PLATFORM_GROUP_ID" -f - <<'EOF'
{
  "id": "multi-tenant-api",
  "name": "multi-tenant-api",
  "paths": ["/service/*"],
  "service_id": "tenant-api-service",
  "plugins": {
    "key-auth": {},
    "proxy-rewrite": {
      "headers": {
        "set": {
          "X-Tenant-ID": "$consumer_name",
          "X-User-ID": "$consumer_name",
          "X-Gateway-Group": "platform"
        }
      }
    }
  }
}
EOF
```

## Declarative Per-Group Management

Use one declarative file per gateway group and apply it with `-g`. This matches
the current `a7 config sync` workflow.

```yaml
version: "1"
services:
  - id: tenant-api-service
    name: tenant-api-service
    upstream:
      type: roundrobin
      nodes:
        - host: internal-service
          port: 8080
          weight: 1
routes:
  - id: multi-tenant-api
    name: multi-tenant-api
    paths:
      - /service/*
    service_id: tenant-api-service
    plugins:
      key-auth: {}
      proxy-rewrite:
        headers:
          set:
            X-Tenant-ID: "$consumer_name"
            X-User-ID: "$consumer_name"
            X-Gateway-Group: platform
```

Apply it:

```bash
a7 config sync -g "$PLATFORM_GROUP_ID" -f platform-tenants.yaml
```

Use `a7 consumer create -f` and `a7 credential create` for tenant identities and
key material.

## Verification

```bash
a7 consumer list -g "$PLATFORM_GROUP_ID"
a7 credential list -g "$PLATFORM_GROUP_ID" --consumer startup-xyz
a7 service get tenant-api-service -g "$PLATFORM_GROUP_ID" -o json
a7 route get multi-tenant-api -g "$PLATFORM_GROUP_ID" -o json
```

Traffic verification requires a deployed gateway:

```bash
curl -i -H "apikey: startup-xyz-key" https://gateway.example.com/service/resource
curl -i -H "apikey: acme-secret-key" https://gateway.example.com/service/resource
```

The backend should receive `X-Tenant-ID`, `X-User-ID`, and `X-Gateway-Group`
headers after successful authentication.

## Important Considerations

- Use different gateway groups for strict runtime isolation.
- Use consumer-level plugins for tenant-specific policies inside one gateway group.
- Keep credentials under `a7 credential`, not embedded directly in consumers.
- `a7 config sync -g` manages one gateway group at a time.
- Use raw consumer payloads for fields that do not have first-class CLI flags.
