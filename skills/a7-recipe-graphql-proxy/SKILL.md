---
name: a7-recipe-graphql-proxy
description: >-
  Recipe skill for implementing GraphQL proxying patterns using API7 Enterprise Edition (API7 EE)
  and the a7 CLI. Covers operation-based routing, per-operation rate limiting, 
  REST-to-GraphQL conversion, and enterprise security for GraphQL APIs.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: recipe
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 config sync
    - a7 consumer-restriction create
---

# a7-recipe-graphql-proxy

## Overview

API7 Enterprise Edition (API7 EE) provides advanced support for GraphQL by exposing internal variables that allow for routing and policy enforcement based on the GraphQL query structure. This is all managed through the **Gateway Group** scoped architecture of the a7 CLI.

Key GraphQL Variables:
- `graphql_name`: The operation name (e.g., `"getUser"`).
- `graphql_operation`: The type of operation (`"query"`, `"mutation"`, or `"subscription"`).
- `graphql_root_fields`: The top-level fields requested (e.g., `["user", "profile"]`).

## When to Use

- **Separation of Concerns**: Routing read-only queries to one backend and mutations to another.
- **Granular Rate Limiting**: Applying stricter limits on expensive mutations than on simple queries.
- **Access Control**: Restricting sensitive GraphQL operations to specific consumer groups.
- **Modernization**: Using `degraphql` to expose a legacy REST API for a GraphQL-first frontend.
- **Enterprise Security**: Adding auth and WAF layers to GraphQL endpoints in a specific **Gateway Group**.

## Approach A: Operation-Based Routing (by Gateway Group)

In an enterprise environment, you may want to route different GraphQL operations to specific microservices or database clusters within your **Gateway Group**.

### 1. Route Queries to Read-Only Backend

```bash
a7 route create -g prod-group -f - <<'EOF'
{
  "id": "gql-queries",
  "uri": "/graphql",
  "vars": [["graphql_operation", "==", "query"]],
  "upstream": {
    "type": "roundrobin",
    "nodes": { "gql-read-replica:4000": 1 }
  }
}
EOF
```

### 2. Route Mutations to Primary Backend

```bash
a7 route create -g prod-group -f - <<'EOF'
{
  "id": "gql-mutations",
  "uri": "/graphql",
  "vars": [["graphql_operation", "==", "mutation"]],
  "upstream": {
    "type": "roundrobin",
    "nodes": { "gql-primary-db:4000": 1 }
  }
}
EOF
```

## Approach B: Per-Operation Rate Limiting

Differentiate between heavy queries and frequent mutations by applying separate `limit-count` plugins.

```bash
# Expensive Mutations: 50 req/min
a7 route update gql-mutations -g prod-group -f - <<'EOF'
{
  "plugins": {
    "key-auth": {},
    "limit-count": {
      "count": 50,
      "time_window": 60,
      "key_type": "var",
      "key": "consumer_name",
      "rejected_code": 429,
      "rejected_msg": "Mutation quota exceeded"
    }
  }
}
EOF
```

## Approach C: REST-to-GraphQL with degraphql

The `degraphql` plugin allows you to map a RESTful URI to a specific GraphQL query on the backend. This is useful for providing a stable REST interface while migrating to a GraphQL backend.

```bash
a7 route create -g prod-group -f - <<'EOF'
{
  "id": "rest-bridge-user",
  "uri": "/api/users/:id",
  "methods": ["GET"],
  "plugins": {
    "degraphql": {
      "query": "query getUser($id: ID!) { user(id: $id) { name email profile { bio } } }",
      "variables": ["id"]
    }
  },
  "upstream": {
    "type": "roundrobin",
    "nodes": { "graphql-engine:8080": 1 }
  }
}
EOF
```

## Approach D: Restricting Operations by Consumer Group

Use the `consumer-restriction` plugin within your **Gateway Group** to limit who can perform specific GraphQL mutations.

```bash
a7 route update gql-mutations -g prod-group -f - <<'EOF'
{
  "plugins": {
    "key-auth": {},
    "consumer-restriction": {
      "type": "consumer_group",
      "whitelist": ["admin-group", "system-accounts"],
      "rejected_code": 403,
      "rejected_msg": "Only admins can perform mutations"
    }
  }
}
EOF
```

## Declarative GraphQL Management

Manage your GraphQL infrastructure across multiple Gateway Groups using `a7 config sync`.

```yaml
# graphql-infra.yaml
gateway_groups:
  - id: prod-group
    routes:
      - id: gql-main
        uri: "/graphql"
        upstream:
          nodes: { "gql-svc:4000": 1 }
        plugins:
          key-auth: {}
          # Global security for all GQL traffic
          ip-restriction:
            whitelist: ["10.0.0.0/8"]
      - id: gql-analytics
        uri: "/graphql"
        priority: 10
        vars: [["graphql_name", "==", "getDailyStats"]]
        upstream:
          nodes: { "analytics-svc:4000": 1 }
```

Apply the configuration:
```bash
a7 config sync -g prod-group -f graphql-infra.yaml
```

## Important Considerations

- **Gateway Group Scope**: All GraphQL routes, upstreams, and consumers MUST be scoped to a `--gateway-group`.
- **Admin API Port**: Use `7443` (HTTPS) for all `a7` CLI operations.
- **Authentication**: Use the `--token` flag to authenticate with the API7 EE dashboard.
- **Body Parsing**: API7 EE parses GraphQL from the request body. Large query strings may hit the `client_max_body_size` limit.
- **Priority**: Use the `priority` field when you have overlapping URIs (e.g., a generic `/graphql` and a specific `/graphql` for one operation name).
- **Batched Requests**: The current GraphQL variable extraction logic operates on the first operation in a batched request.

## Verification

```bash
# Test basic query
curl -X POST https://gateway.prod.example.com/graphql \
  -H "Content-Type: application/json" \
  -H "X-API-TOKEN: your-token" \
  -d '{"query": "query { health { status } }"}'

# Test mutation access control
curl -X POST https://gateway.prod.example.com/graphql \
  -H "Content-Type: application/json" \
  -H "X-API-TOKEN: user-token" \
  -d '{"query": "mutation { deleteUser(id: 1) { success } }"}'
# Should return 403 if user is not in the whitelist

# Test REST-to-GraphQL
curl -i https://gateway.prod.example.com/api/users/123
# Returns the mapped GraphQL response
```
