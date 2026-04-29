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
    - a7 service create
    - a7 service get
    - a7 route create
    - a7 route update
    - a7 route get
    - a7 consumer create
    - a7 credential create
    - a7 config sync
---

# a7-recipe-graphql-proxy

## Overview

API7 EE can route and protect GraphQL traffic by using GraphQL variables from
the parsed request body:

- `graphql_name`: operation name, such as `getUser`.
- `graphql_operation`: operation type, such as `query` or `mutation`.
- `graphql_root_fields`: top-level fields requested by the operation.

Use the current a7 service-backed route model:

1. Create services for the GraphQL backends.
2. Create routes with `paths` and `service_id`.
3. Add GraphQL `vars` and plugins through `a7 route create/update -f` payloads.

## When to Use

- Route read-only GraphQL queries and mutations to different backends.
- Apply tighter limits to expensive operations.
- Expose REST-style paths that are translated to GraphQL with `degraphql`.
- Protect sensitive GraphQL operations with auth and access-control plugins.

## Approach A: Operation-Based Routing

Create separate services for read and write GraphQL traffic.

```bash
a7 service create -g prod-group -f - <<'EOF'
{
  "id": "gql-read-service",
  "name": "gql-read-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "gql-read-replica", "port": 4000, "weight": 1}
    ]
  }
}
EOF

a7 service create -g prod-group -f - <<'EOF'
{
  "id": "gql-write-service",
  "name": "gql-write-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "gql-primary-db", "port": 4000, "weight": 1}
    ]
  }
}
EOF
```

Route GraphQL queries to the read service:

```bash
a7 route create -g prod-group -f - <<'EOF'
{
  "id": "gql-queries",
  "name": "gql-queries",
  "paths": ["/graphql"],
  "service_id": "gql-read-service",
  "vars": [["graphql_operation", "==", "query"]]
}
EOF
```

Route GraphQL mutations to the write service:

```bash
a7 route create -g prod-group -f - <<'EOF'
{
  "id": "gql-mutations",
  "name": "gql-mutations",
  "paths": ["/graphql"],
  "service_id": "gql-write-service",
  "vars": [["graphql_operation", "==", "mutation"]]
}
EOF
```

Use route priorities if a generic `/graphql` route overlaps with more specific
GraphQL operation routes.

## Approach B: Per-Operation Rate Limiting

Apply stricter limits to mutation traffic. Include the `service_id` in update
payloads so the route remains bound to the intended service.

```bash
a7 route update gql-mutations -g prod-group -f - <<'EOF'
{
  "service_id": "gql-write-service",
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

Create consumers and credentials separately:

```bash
a7 consumer create -g prod-group --username frontend-app
a7 credential create -g prod-group --consumer frontend-app --plugins-json '{"key-auth":{"key":"frontend-secret"}}'
```

## Approach C: REST-to-GraphQL with degraphql

The `degraphql` plugin maps a REST-style path to a GraphQL query on the backend.

```bash
a7 service create -g prod-group -f - <<'EOF'
{
  "id": "graphql-engine-service",
  "name": "graphql-engine-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": [
      {"host": "graphql-engine", "port": 8080, "weight": 1}
    ]
  }
}
EOF

a7 route create -g prod-group -f - <<'EOF'
{
  "id": "rest-bridge-user",
  "name": "rest-bridge-user",
  "paths": ["/api/users/:id"],
  "methods": ["GET"],
  "service_id": "graphql-engine-service",
  "plugins": {
    "degraphql": {
      "query": "query getUser($id: ID!) { user(id: $id) { name email profile { bio } } }",
      "variables": ["id"]
    }
  }
}
EOF
```

## Approach D: Restrict Mutations

Use `consumer-restriction` to limit mutation routes to selected consumers or
consumer groups.

```bash
a7 route update gql-mutations -g prod-group -f - <<'EOF'
{
  "service_id": "gql-write-service",
  "plugins": {
    "key-auth": {},
    "consumer-restriction": {
      "type": "consumer_name",
      "whitelist": ["admin-client", "system-client"],
      "rejected_code": 403,
      "rejected_msg": "Only approved clients can perform mutations"
    }
  }
}
EOF
```

## Declarative Management Notes

`a7 config sync` can manage services and normal service-backed routes. GraphQL
operation matching currently requires raw route payload fields such as `vars`,
so use `a7 route create/update -f` for those operation-specific routes.

```yaml
version: "1"
services:
  - id: graphql-engine-service
    name: graphql-engine-service
    upstream:
      type: roundrobin
      nodes:
        - host: graphql-engine
          port: 8080
          weight: 1
routes:
  - id: rest-bridge-user
    name: rest-bridge-user
    paths:
      - /api/users/:id
    methods:
      - GET
    service_id: graphql-engine-service
    plugins:
      degraphql:
        query: "query getUser($id: ID!) { user(id: $id) { name email profile { bio } } }"
        variables:
          - id
```

Apply it to one gateway group:

```bash
a7 config sync -g prod-group -f graphql-rest-bridge.yaml
```

## Verification

```bash
a7 service get gql-read-service -g prod-group -o json
a7 route get gql-queries -g prod-group -o json
a7 route get gql-mutations -g prod-group -o json
```

Traffic verification requires a deployed gateway and GraphQL backend:

```bash
curl -X POST https://gateway.prod.example.com/graphql \
  -H "Content-Type: application/json" \
  -H "apikey: frontend-secret" \
  -d '{"query": "query getUser { user(id: 1) { name } }"}'

curl -X POST https://gateway.prod.example.com/graphql \
  -H "Content-Type: application/json" \
  -H "apikey: frontend-secret" \
  -d '{"query": "mutation deleteUser { deleteUser(id: 1) { success } }"}'
```

## Important Considerations

- Body parsing is required for GraphQL variables to be available.
- Batched requests may only expose variables for the first operation.
- Use route priorities when multiple routes share `/graphql`.
- Keep auth credentials under `a7 credential`, not embedded directly in the consumer.
