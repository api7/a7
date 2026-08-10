---
name: a7-plugin-key-auth
description: >-
  Skill for configuring the API7 Enterprise Edition (API7 EE) key-auth plugin via the a7 CLI.
  Covers API key authentication setup on routes, consumer credential binding,
  key lookup from header/query/cookie, hide_credentials, anonymous consumer
  fallback, and common operational patterns.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: plugin
  apisix_version: ">=3.0.0"
  plugin_name: key-auth
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 consumer create
    - a7 consumer update
---

# a7-plugin-key-auth

## Overview

The `key-auth` plugin authenticates requests using API keys. Clients include a
key in a header, query parameter, or cookie. API7 EE looks up the key against
consumer credentials and, on match, forwards the request with consumer identity
headers. On failure it returns `401 Unauthorized`.

## When to Use

- Protect routes with simple API-key authentication
- Identify which consumer is calling an API
- Combine with rate-limiting for tiered access (authenticated vs anonymous)
- Hide credentials from upstream services

## Plugin Configuration Reference (Route/Service)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `header` | string | No | `"apikey"` | Header name to extract API key from |
| `query` | string | No | `"apikey"` | Query parameter name (lower priority than header) |
| `hide_credentials` | boolean | No | `false` | Remove key from request before forwarding upstream |
| `anonymous_consumer` | string | No | — | Consumer username for unauthenticated requests |
| `realm` | string | No | `"key"` | Realm in `WWW-Authenticate` response header on 401 |

## Consumer Credential Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `key` | string | **Yes** | Unique API key for the consumer. Auto-encrypted in the database. |

## Key Lookup Priority

1. **Header** (default: `apikey`) — checked first
2. **Query parameter** (default: `apikey`) — checked if header absent
3. If both absent → `401 Unauthorized` with `"Missing API key in request"`

## Step-by-Step: Enable key-auth on a Route

Replace `<gateway-group-id>` with the ID returned by
`a7 gateway-group list -o json`.

### 1. Create a consumer

```bash
a7 consumer create -g <gateway-group-id> -f - <<'EOF'
{
  "username": "alice"
}
EOF
```

### 2. Add key-auth credential to the consumer

```bash
a7 credential create cred-alice-key-auth -g <gateway-group-id> \
  --consumer alice \
  --plugins-json '{"key-auth":{"key":"alice-secret-key-001"}}'
```

### 3. Create a service and route with key-auth enabled

```bash
a7 service create -g <gateway-group-id> -f - <<'EOF'
{
  "id": "protected-api-service",
  "name": "Protected API",
  "upstream": {
    "type": "roundrobin",
    "nodes": [{"host": "backend", "port": 8080, "weight": 1}]
  }
}
EOF

a7 route create -g <gateway-group-id> -f - <<'EOF'
{
  "id": "protected-api",
  "paths": ["/api/*"],
  "service_id": "protected-api-service",
  "plugins": {
    "key-auth": {}
  }
}
EOF
```

### 4. Verify authentication

```bash
# Should succeed (200)
curl -i http://127.0.0.1:9080/api/users -H "apikey: alice-secret-key-001"

# Should fail (401)
curl -i http://127.0.0.1:9080/api/users
```

## Common Patterns

### Custom header name

```json
{
  "plugins": {
    "key-auth": {
      "header": "X-API-Token"
    }
  }
}
```

Client sends: `curl -H "X-API-Token: alice-secret-key-001" ...`

### Query parameter authentication

```json
{
  "plugins": {
    "key-auth": {
      "query": "token"
    }
  }
}
```

Client sends: `curl "http://127.0.0.1:9080/api/users?token=alice-secret-key-001"`

### Hide credentials from upstream

```json
{
  "plugins": {
    "key-auth": {
      "hide_credentials": true
    }
  }
}
```

The `apikey` header or query param is stripped before reaching the backend.
Always enable this in production.

### Anonymous consumer with rate limiting

```bash
# Create anonymous consumer with strict limits
a7 consumer create -g <gateway-group-id> -f - <<'EOF'
{
  "username": "anonymous",
  "plugins": {
    "limit-count": {
      "count": 10,
      "time_window": 60,
      "rejected_code": 429
    }
  }
}
EOF
```

```json
{
  "plugins": {
    "key-auth": {
      "anonymous_consumer": "anonymous"
    }
  }
}
```

Requests with valid keys → authenticated consumer. Requests without keys →
anonymous consumer with rate limits.

## Headers Added to Upstream

On successful authentication, API7 EE adds:

| Header | Value |
|--------|-------|
| `X-Consumer-Username` | Consumer's username |
| `X-Credential-Identifier` | Credential ID |
| `X-Consumer-Custom-Id` | Consumer's `labels.custom_id` (if set) |

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `401 "Missing API key in request"` | No key in header or query | Add `apikey` header or query param |
| `401 "Invalid API key in request"` | Key does not match any consumer | Verify the key value in consumer credentials |
| Key visible in upstream logs | `hide_credentials` is false | Set `hide_credentials: true` |
| Anonymous users not working | `anonymous_consumer` not set or consumer missing | Create the consumer and set the field |

## Config Sync Example

Save the following as `key-auth.yaml`:

```yaml
version: "1"
services:
  - id: protected-api-service
    name: Protected API service
    upstream:
      type: roundrobin
      nodes:
        - host: backend
          port: 8080
          weight: 1
routes:
  - id: protected-api
    name: Protected API route
    paths:
      - /api/*
    service_id: protected-api-service
    plugins:
      key-auth: {}
```

Validate and apply this partial configuration to the target gateway group:

```bash
a7 config validate -f key-auth.yaml
a7 config sync -g <gateway-group-id> -f key-auth.yaml --delete=false
```

> **Note**: Create the consumer and credential separately with
> `a7 consumer create` and `a7 credential create`. Config Sync manages only the
> service and route in this example. Disabling deletion preserves other
> resources that are not included in this partial configuration.
