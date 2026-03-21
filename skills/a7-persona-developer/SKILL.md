---
name: a7-persona-developer
description: >-
  Persona skill for API developers building and testing APIs on API7 Enterprise Edition (API7 EE)
  using the a7 CLI. Provides decision frameworks for API design, Service Template 
  lifecycle, Portal publishing, plugin configuration, and local-to-cloud development workflows.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: persona
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 service-template create
    - a7 service-template publish
    - a7 route create
    - a7 upstream create
    - a7 consumer create
    - a7 portal list
    - a7 config sync
    - a7 config validate
    - a7 debug trace
---

# a7-persona-developer

## Who This Is For

You are an **API Developer** responsible for:
- Designing API schemas and configuring routes within a **Gateway Group**.
- Leveraging **Service Templates** to standardize API deployments across environments.
- Publishing APIs to the **Developer Portal** for internal or external consumption.
- Configuring advanced enterprise plugins (OIDC, Canary, Request/Response Transformation).
- Debugging complex request flows using built-in enterprise tracing tools.

## Core Enterprise Concepts

In API7 EE, developers work within a structured lifecycle:
1. **Gateway Groups**: Your assigned workspace (e.g., `ecommerce-dev`).
2. **Service Templates**: Blueprints for services (e.g., `payment-service-v1`).
3. **Publication**: Promoting a Service Template to a live Gateway Group.
4. **Developer Portal**: The consumer-facing documentation and self-service hub.

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

### Step 1: Create a Service Template

Standardize your service configuration before deploying it to a gateway.

```bash
a7 service-template create -f - <<'EOF'
{
  "name": "user-service-template",
  "desc": "Template for User Management API",
  "upstream": {
    "type": "roundrobin",
    "nodes": { "user-backend.internal:8080": 1 }
  },
  "plugins": {
    "key-auth": {}
  }
}
EOF
```

### Step 2: Publish to a Gateway Group

```bash
# Publish the template to the 'staging-group'
a7 service-template publish user-service-template --group staging-group
```

### Step 3: Configure a Route within the Group

```bash
a7 route create -g staging-group -f - <<'EOF'
{
  "id": "user-v1-get",
  "uri": "/v1/users/*",
  "methods": ["GET"],
  "service_id": "user-service-template",
  "plugins": {
    "proxy-rewrite": {
      "regex_uri": ["^/v1/users/(.*)", "/users/$1"]
    }
  }
}
EOF
```

### Step 4: Add to Developer Portal

```bash
# Link your service to the Portal for documentation
a7 portal publish -g staging-group --service user-service-template --portal public-portal
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
- name: Publish Service Template
  run: |
    a7 service-template publish ${{ env.SERVICE_NAME }} \
      --group ${{ env.TARGET_GROUP }} \
      --token ${{ secrets.A7_TOKEN }}
```

## Decision Framework for Developers

| Situation | Action | Command |
|-----------|--------|---------|
| Standardizing multiple APIs | Use a Service Template | `a7 service-template create` |
| Promoting to production | Publish Template to Group | `a7 service-template publish` |
| Exposing to external devs | Add to Developer Portal | `a7 portal publish` |
| Backend URI mismatch | Use `proxy-rewrite` | `a7 route update ...` |
| Testing Canary version | Use `traffic-split` | `a7 route update ...` |
| Auth failure (401) | Check Trace & Logs | `a7 debug trace` & `a7 debug logs` |

## Best Practices

1. **Templates First**: Always create a **Service Template** for reusable service logic.
2. **Group Scoping**: Always use the `-g` flag to target the correct environment.
3. **Port & Protocol**: Ensure you are connecting to the Dashboard via HTTPS on port `7443`.
4. **Token Security**: Do not hardcode your `--token` in scripts; use environment variables or secrets.
5. **Declarative Sync**: Prefer `a7 config sync` for complex multi-route deployments.
6. **Documentation**: Always provide a description (`--desc`) for routes and templates for colleagues.
7. **Trace Verbosity**: Use `--verbose` in `debug trace` to inspect plugin input/output headers.
8. **Portal Sync**: Keep your Swagger/OpenAPI docs in sync with your Portal publication.
