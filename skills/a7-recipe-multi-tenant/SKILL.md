---
name: a7-recipe-multi-tenant
description: >-
  Recipe skill for implementing multi-tenant patterns using API7 Enterprise Edition (API7 EE)
  and the a7 CLI. Covers physical isolation via Gateway Groups, logical isolation via 
  Consumer Groups, and enterprise-grade tenant management.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: recipe
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 consumer create
    - a7 consumer-group create
    - a7 route create
    - a7 gateway-group create
    - a7 config sync
    - a7 config dump
---

# a7-recipe-multi-tenant

## Overview

Multi-tenancy in API7 Enterprise Edition (API7 EE) provides a robust framework for serving multiple isolated tenants (customers, business units, or environment tiers) through a unified control plane.

API7 EE enhances standard multi-tenancy with:
1. **Gateway Groups (Physical/Logical Isolation)** — The primary unit of isolation. Each group has its own configuration, plugins, and can be deployed to separate gateway instances.
2. **Consumer Groups (Soft Isolation)** — Group consumers within a Gateway Group for shared rate limits and authentication policies.
3. **Gateway Group Scoping** — Every resource (routes, upstreams, consumers) is scoped to a specific `--gateway-group`.
4. **Enterprise RBAC** — Control which teams can manage which Gateway Groups.

## When to Use

- **Multi-Environment**: Separating `dev`, `staging`, and `prod` within one API7 instance.
- **SaaS/B2B**: Providing dedicated gateway configurations for different enterprise customers.
- **Internal Platform**: Serving multiple internal product teams with strict quota and configuration isolation.
- **Compliance**: Keeping configuration and traffic for regulated data separate from general traffic.

## Approach A: Gateway Groups for Tiered Isolation

In API7 EE, Gateway Groups are the most effective way to separate "Free" vs "Premium" tiers or different business units.

### 1. Define Gateway Groups

```bash
# Create a Premium Gateway Group
a7 gateway-group create premium-tier --desc "High-performance tier for paid customers"

# Create a Standard Gateway Group
a7 gateway-group create standard-tier --desc "Standard tier for free/trial users"
```

### 2. Configure Tier-Specific Policies

Each group can have its own global rules or specific routes.

```bash
# Configure global rate limit for the standard tier
a7 global-rule create -g standard-tier -f - <<'EOF'
{
  "id": "global-throttle",
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

## Approach B: Consumer Groups within a Gateway Group

Use Consumer Groups to manage different tenants sharing the same Gateway Group infrastructure.

### 1. Create Consumer Groups (scoped to a Gateway Group)

```bash
# Free tier consumers in the "platform" group
a7 consumer-group create -g platform -f - <<'EOF'
{
  "id": "tenant-free",
  "desc": "Free tier tenant group",
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

# Pro tier consumers in the "platform" group
a7 consumer-group create -g platform -f - <<'EOF'
{
  "id": "tenant-pro",
  "desc": "Pro tier tenant group",
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

### 2. Assign Consumers to Tenants

```bash
# Acme Corp (Pro Tenant)
a7 consumer create -g platform -f - <<'EOF'
{
  "username": "acme-corp",
  "group_id": "tenant-pro",
  "plugins": {
    "key-auth": { "key": "acme-secret-key" }
  }
}
EOF

# Startup XYZ (Free Tenant)
a7 consumer create -g platform -f - <<'EOF'
{
  "username": "startup-xyz",
  "group_id": "tenant-free",
  "plugins": {
    "key-auth": { "key": "startup-xyz-key" }
  }
}
EOF
```

## Approach C: Forwarding Tenant Context

Inject tenant metadata into request headers using `proxy-rewrite`. This allows backends to identify the tenant without re-parsing auth tokens.

```bash
a7 route create -g platform -f - <<'EOF'
{
  "id": "multi-tenant-api",
  "uri": "/service/*",
  "upstream": {
    "type": "roundrobin",
    "nodes": { "internal-service:8080": 1 }
  },
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

## Declarative Multi-Tenant Management

Manage multiple Gateway Groups and their respective tenants in a single configuration file.

```yaml
# a7-enterprise-tenants.yaml
gateway_groups:
  - id: platform
    consumer_groups:
      - id: tenant-free
        plugins:
          limit-count:
            count: 100
            time_window: 86400
      - id: tenant-pro
        plugins:
          limit-count:
            count: 10000
            time_window: 86400
    consumers:
      - username: acme-corp
        group_id: tenant-pro
        plugins:
          key-auth:
            key: "acme-secret-key"
    routes:
      - id: api-v1
        uri: "/v1/*"
        upstream:
          nodes:
            "api-backend:8080": 1
        plugins:
          key-auth: {}
          proxy-rewrite:
            headers:
              set:
                X-Tenant: "$consumer_group_id"
```

Apply the configuration:
```bash
# Apply to the specific gateway group
a7 config sync -g platform -f a7-enterprise-tenants.yaml
```

## Important Considerations

- **Physical Isolation**: If tenants require strict performance isolation, deploy them to different **Gateway Groups** which are in turn bound to different physical gateway instances.
- **Token Management**: In API7 EE, use the `--token` flag for a7 CLI commands to authenticate with the dashboard.
- **Priority**: Authentication plugins (like `key-auth`) MUST run before `proxy-rewrite` for `$consumer_name` and `$consumer_group_id` variables to be available.
- **Port**: The default API7 EE Admin API port is `7443` (HTTPS). Ensure your CLI is configured correctly.

## Verification

```bash
# Check consumers in a specific group
a7 consumer list -g platform

# Verify Gateway Group status
a7 gateway-group get platform

# Test Tenant A (Free)
curl -i -H "apikey: startup-xyz-key" https://gateway.example.com/v1/resource
# Check headers at backend for X-Tenant: tenant-free

# Test Tenant B (Pro)
curl -i -H "apikey: acme-secret-key" https://gateway.example.com/v1/resource
# Check headers at backend for X-Tenant: tenant-pro
```
