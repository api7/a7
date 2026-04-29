---
name: a7-recipe-multi-tenant
description: >-
  Recipe skill for implementing multi-tenant patterns using API7 Enterprise Edition (API7 EE)
  and the a7 CLI. Covers gateway-group isolation, consumer-group policies,
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
    - a7 consumer-group create
    - a7 consumer-group list
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
2. Consumer groups for shared tenant policies.
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

Gateway groups are the primary isolation boundary.

```bash
a7 gateway-group create premium-tier --desc "High-performance tier for paid customers"
a7 gateway-group create standard-tier --desc "Standard tier for free and trial users"
a7 gateway-group create platform --desc "Shared platform gateway group for tenant consumers and routes"
```

Each group can have its own global policies:

```bash
a7 global-rule create -g standard-tier -f - <<'EOF'
{
  "id": "standard-global-throttle",
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

## Approach B: Consumer Groups and Credentials

Create consumer groups in the shared gateway group:

```bash
a7 consumer-group create -g platform -f - <<'EOF'
{
  "id": "tenant-free",
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

a7 consumer-group create -g platform -f - <<'EOF'
{
  "id": "tenant-pro",
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
```

Create consumers with raw payloads when assigning them to consumer groups, then
create key-auth credentials with `a7 credential create`.

```bash
a7 consumer create -g platform -f - <<'EOF'
{
  "username": "acme-corp",
  "group_id": "tenant-pro"
}
EOF

a7 credential create -g platform --consumer acme-corp --plugins-json '{"key-auth":{"key":"acme-secret-key"}}'

a7 consumer create -g platform -f - <<'EOF'
{
  "username": "startup-xyz",
  "group_id": "tenant-free"
}
EOF

a7 credential create -g platform --consumer startup-xyz --plugins-json '{"key-auth":{"key":"startup-xyz-key"}}'
```

## Approach C: Tenant-Aware Service Route

Create the backend service first:

```bash
a7 service create -g platform -f - <<'EOF'
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
a7 route create -g platform -f - <<'EOF'
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
          "X-Tenant-ID": "$consumer_group_id",
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
consumer_groups:
  - id: tenant-free
    plugins:
      limit-count:
        count: 100
        time_window: 86400
        key_type: var
        key: consumer_name
  - id: tenant-pro
    plugins:
      limit-count:
        count: 10000
        time_window: 86400
        key_type: var
        key: consumer_name
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
            X-Tenant-ID: "$consumer_group_id"
            X-User-ID: "$consumer_name"
            X-Gateway-Group: platform
```

Apply it:

```bash
a7 config sync -g platform -f platform-tenants.yaml
```

Use `a7 consumer create -f` and `a7 credential create` for tenant identities and
credentials when consumer group assignment or key material is required.

## Verification

```bash
a7 consumer-group list -g platform
a7 consumer list -g platform
a7 service get tenant-api-service -g platform -o json
a7 route get multi-tenant-api -g platform -o json
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
- Use consumer groups for shared tenant policies inside one gateway group.
- Keep credentials under `a7 credential`, not embedded directly in consumers.
- `a7 config sync -g` manages one gateway group at a time.
- Use raw consumer payloads for fields that do not have first-class CLI flags.
