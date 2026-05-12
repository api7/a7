---
name: a7-persona-developer
description: >-
  Persona skill for API developers building and testing APIs on API7 Enterprise Edition (API7 EE)
  using the a7 CLI. Provides decision frameworks for service-backed API design,
  route configuration, plugin configuration, and local-to-cloud development workflows.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: persona
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 route create
    - a7 service create
    - a7 consumer create
    - a7 config sync
    - a7 config validate
    - a7 debug trace
---

# a7-persona-developer

## Who This Is For

You are an **API Developer** responsible for:
- Designing API schemas and configuring routes within a **Gateway Group**.
- Defining **Services** with inline upstreams and attaching routes with `service_id`.
- Publishing APIs to gateway groups with service-backed routes.
- Configuring advanced enterprise plugins (OIDC, Canary, Request/Response Transformation).
- Debugging complex request flows using built-in enterprise tracing tools.

## Core Enterprise Concepts

In API7 EE, developers work within a structured lifecycle:
1. **Gateway Groups**: Your assigned workspace (e.g., `ecommerce-dev`).
2. **Services**: Runtime service definitions with inline upstreams (e.g., `payment-service-v1`).
3. **Service-backed Routes**: Routes should reference a service with `service_id` in current API7 EE.

## Getting Started

### 1. Connect to the Enterprise Dashboard

```bash
# Set up your development context
a7 context create dev-ee \
  --server https://dashboard.enterprise.com:7443 \
  --token <your-personal-api-token>

# Use the context
a7 context use dev-ee

# Verify your access to assigned groups
a7 gateway-group list
```

### 2. Explore Enterprise Plugins

```bash
# List all plugins available in your group
a7 plugin list -g my-group

# View the schema and required fields for an enterprise plugin
a7 plugin get openid-connect -g my-group --output json
```

## Building & Publishing Your API

### Step 1: Create a Service

Create the service configuration in the target gateway group.

```bash
a7 service create -g staging-group -f - <<'EOF'
{
  "id": "user-service",
  "name": "user-service",
  "desc": "User Management API",
  "upstream": {
    "type": "roundrobin",
    "nodes": [{"host": "user-backend.internal", "port": 8080, "weight": 1}]
  },
  "plugins": {
    "key-auth": {}
  }
}
EOF
```

### Step 2: Configure a Route within the Group

```bash
a7 route create -g staging-group -f - <<'EOF'
{
  "id": "user-v1-get",
  "uri": "/v1/users/*",
  "methods": ["GET"],
  "service_id": "user-service",
  "plugins": {
    "proxy-rewrite": {
      "regex_uri": ["^/v1/users/(.*)", "/users/$1"]
    }
  }
}
EOF
```

## Plugin Selection Guide (Enterprise Edition)

### Identity & Security

| Need | Plugin | Enterprise Benefit |
|------|--------|--------------------|
| SSO / OIDC | `openid-connect` | Native integration with Okta, Azure AD, Ping |
| LDAP Auth | `ldap-auth` | Connect to enterprise directory services |
| mTLS | `mtls` | Enforce client certificate validation at the group level |
| WAF / Shield | `api-breaker` | Protect backends from cascading failures |

### Traffic & Resilience

| Need | Plugin | Enterprise Benefit |
|------|--------|--------------------|
| Canary Rollout | `traffic-split` | Weighted routing for zero-downtime testing |
| Fault Injection | `fault-injection` | Chaos engineering directly in the gateway |
| Data Masking | `response-rewrite` | Mask PII in response bodies for compliance |

## Local to Cloud Workflow

### 1. Develop Locally
Run a local APISIX instance via Docker and test your routes using generic `a7` commands.

### 2. Validate for Enterprise
Before pushing to the Dashboard, validate your config against the enterprise schema.

```bash
a7 config validate -g dev-group -f my-api.yaml
```

### 3. Sync to Dashboard
```bash
a7 config sync -g dev-group -f my-api.yaml
```

## Debugging Enterprise APIs

### Request Tracing

Use `debug trace` to see exactly which plugins are executed and how the URI is transformed within your **Gateway Group**.

```bash
# Trace a request with an API Key
a7 debug trace -g dev-group \
  --uri /v1/users/123 \
  --method GET \
  --header "X-API-KEY: my-dev-key" \
  --verbose
```

### Live Log Streaming

```bash
# Stream error logs for your specific group
a7 debug logs -g dev-group --follow --level error
```

## CI/CD Integration

Automate your API lifecycle using `a7` in your pipelines.

```yaml
# Example GitHub Action Step
- name: Sync Service Config
  run: |
    a7 config sync -f api7.yaml \
      --gateway-group ${{ env.TARGET_GROUP }} \
      --token ${{ secrets.A7_TOKEN }}
```

## Decision Framework for Developers

| Situation | Action | Command |
|-----------|--------|---------|
| Standardizing multiple APIs | Use services with inline upstreams and shared plugins | `a7 service create` |
| Promoting to production | Sync service and route config to target group | `a7 config sync` |
| Exposing an API path | Create or update a service-backed route | `a7 route create -f route.yaml` |
| Backend URI mismatch | Use `proxy-rewrite` | `a7 route update ...` |
| Testing Canary version | Use `traffic-split` | `a7 route update ...` |
| Auth failure (401) | Check Trace & Logs | `a7 debug trace` & `a7 debug logs` |

## Best Practices

1. **Services First**: Put reusable upstream and plugin configuration on services.
2. **Group Scoping**: Always use the `-g` flag to target the correct environment.
3. **Port & Protocol**: Ensure you are connecting to the Dashboard via HTTPS on port `7443`.
4. **Token Security**: Do not hardcode your `--token` in scripts; use environment variables or secrets.
5. **Declarative Sync**: Prefer `a7 config sync` for complex multi-route deployments.
6. **Documentation**: Always provide a description (`--desc`) for routes and services for colleagues.
7. **Trace Verbosity**: Use `--verbose` in `debug trace` to inspect plugin input/output headers.
8. **Route Model**: Prefer `service create` plus `route create` with `service_id`; avoid standalone upstream workflows for API7 EE.
