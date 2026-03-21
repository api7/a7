---
name: a7-persona-operator
description: >-
  Persona skill for platform operators and DevOps engineers managing API7 Enterprise Edition (API7 EE)
  instances using the a7 CLI. Provides decision frameworks for managing Gateway Groups, 
  Enterprise RBAC, complex deployments, troubleshooting, and disaster recovery.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: persona
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 gateway-group list
    - a7 gateway-group create
    - a7 route list
    - a7 upstream health
    - a7 config sync
    - a7 config dump
    - a7 debug logs
    - a7 debug trace
    - a7 health
---

# a7-persona-operator

## Who This Is For

You are an **Enterprise Platform Operator or DevOps Engineer** responsible for:
- Orchestrating multiple **Gateway Groups** across different environments and regions.
- Managing Enterprise RBAC and API tokens for secure access to the Control Plane.
- Ensuring 99.99% availability of the API7 EE infrastructure.
- Implementing zero-downtime configuration deployments and rollbacks.
- Hardening security via Global Rules and Enterprise Plugins across Gateway Groups.

## Core Enterprise Concepts

In API7 EE, your operational model shifts from managing a single instance to managing a multi-tenant platform:
1. **Gateway Groups**: The primary unit of isolation and deployment.
2. **Control Plane (Dashboard)**: Central management hub (default port `7443` HTTPS).
3. **Data Plane (Gateways)**: Distributed instances that execute the configuration.
4. **API Tokens**: Required for all CLI operations (`--token`).

## Context & Group Management

Operators manage multiple Gateway Groups. Use `a7 context` and the `-g` flag to maintain control.

```bash
# Configure access to the Enterprise Dashboard
a7 context create prod-ee \
  --server https://dashboard.enterprise.com:7443 \
  --token <your-long-lived-api-token>

# Switch context
a7 context use prod-ee

# Verify connectivity
a7 health

# List available Gateway Groups
a7 gateway-group list
```

## Daily Operations Checklist

### 1. Platform Health & Connectivity

```bash
# Check if the Dashboard and CLI are connected
a7 health

# Verify status of a specific Gateway Group
a7 gateway-group get internal-apps

# Check upstream health within a group
a7 upstream health <upstream-id> -g internal-apps
```

### 2. Configuration Audit & Drift Detection

```bash
# Backup the state of a specific Gateway Group
a7 config dump -g finance-dept > finance-backup-$(date +%F).yaml

# Detect drift between a local source-of-truth and the Dashboard
a7 config diff -g finance-dept -f finance-infra.yaml

# Validate enterprise plugin configuration before sync
a7 config validate -g finance-dept -f updated-config.yaml
```

### 3. Enterprise Security & SSL

```bash
# List SSL certs for a Gateway Group
a7 ssl list -g public-gateway

# Add a new SSL certificate to a group
a7 ssl create -g public-gateway -f - <<'EOF'
{
  "cert": "...",
  "key": "...",
  "snis": ["api.acme.com"]
}
EOF
```

## Advanced Deployment Workflow

### Zero-Downtime Promotion

```bash
# 1. Validate in Dev Gateway Group
a7 config sync -g dev-group -f infra-v2.yaml

# 2. Preview changes for Prod Gateway Group
a7 config diff -g prod-group -f infra-v2.yaml

# 3. Apply to Prod with full audit trail
a7 config sync -g prod-group -f infra-v2.yaml

# 4. Verify traffic flow in Prod
a7 debug trace -g prod-group --uri /v1/status
```

### Emergency Rollback

```bash
# Locate the last known good backup
ls *-backup-*.yaml

# Restore the Gateway Group state immediately
a7 config sync -g prod-group -f last-good-backup.yaml
```

## Troubleshooting & Incident Response

### Analyzing Failed Requests

```bash
# 1. Trace a request through a specific Gateway Group
a7 debug trace -g customer-facing --uri /api/v1/checkout --method POST

# 2. Follow real-time error logs for a Gateway Group
a7 debug logs -g customer-facing --follow --level error

# 3. Check for misconfigured Global Rules
a7 global-rule list -g customer-facing --output json
```

### Identifying Performance Bottlenecks

```bash
# Check the execution time of plugins in the trace
a7 debug trace -g api-internal --uri /heavy-endpoint --verbose

# List active routes and their hit counts (if metrics enabled)
a7 route list -g api-internal --output wide
```

## Security Hardening (Enterprise Grade)

### Global IP Restriction (Group Scoped)

```bash
a7 global-rule create -g sensitive-apps -f - <<'EOF'
{
  "id": "restrict-to-vpc",
  "plugins": {
    "ip-restriction": {
      "whitelist": ["10.0.0.0/16", "172.16.0.0/12"]
    }
  }
}
EOF
```

### Enforcing Enterprise Authentication

```bash
# Apply a global rule to enforce OpenID Connect for all routes in a group
a7 global-rule create -g public-facing -f - <<'EOF'
{
  "id": "enforce-oidc",
  "plugins": {
    "openid-connect": {
      "client_id": "...",
      "client_secret": "...",
      "discovery": "https://idp.example.com/.well-known/openid-configuration"
    }
  }
}
EOF
```

## Decision Framework for Operators

| Situation | Action | Command |
|-----------|--------|---------|
| New Team Onboarding | Create Gateway Group & Assign RBAC | `a7 gateway-group create <name>` |
| Configuration Drift | Compare local YAML with Live | `a7 config diff -g <group> -f <file>` |
| Upstream Timeout | Check health & logs | `a7 upstream health <id> -g <group>` |
| Security Breach | Block IP via Global Rule | `a7 global-rule create -g <group> -f block.json` |
| Compliance Audit | Dump all configs for review | `a7 config dump -g <group>` |
| Version Upgrade | Validate then Sync | `a7 config validate` then `a7 config sync` |

## Operational Best Practices

1. **Gateway Group Isolation**: Never mix development and production resources in the same Gateway Group.
2. **Token Security**: Treat your API7 EE Token like a root password. Use short-lived tokens for CI/CD.
3. **Always use `-g`**: Explicitly specify the Gateway Group to prevent accidental changes to the wrong environment.
4. **Audit Logs**: Regularly review the Dashboard audit logs for any CLI-initiated changes.
5. **HTTPS Only**: Always use the HTTPS port (`7443`) for the Control Plane.
6. **Config as Code**: Store all Gateway Group configurations in Git. Treat the Dashboard as a projection of your repository.
7. **Health Checks**: Always enable active health checks for upstreams to allow the gateway to self-heal.
8. **Context Awareness**: Use descriptive names for contexts (e.g., `hk-region-prod`, `us-west-staging`) to avoid confusion in multi-region setups.
