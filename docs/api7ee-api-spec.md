# API7 Enterprise Edition Admin API Specification

This is the authoritative reference for the API7 EE Admin API, covering both control-plane and runtime endpoints.

## Dual-API Architecture

API7 EE distinguishes between design-time (control-plane) and runtime (gateway) resources.

- **Control-plane Prefix**: `/api` (e.g., `/api/gateway-groups`)
- **Runtime Prefix**: `/apisix/admin` (e.g., `/apisix/admin/routes`)
- **Default Port**: 7443 (HTTPS)

## Authentication

Authentication is required for all requests via the `X-API-KEY` header. API7 EE uses prefixed tokens to distinguish access levels:

| Prefix | Type | Description |
|--------|------|-------------|
| `a7ee` | Access Token | General API access (RBAC-controlled) |
| `a7adm` | Admin Key | Gateway-level administrative access |
| `a7prt` | Portal Token | Developer portal specific access |

## Gateway Group Scoping

Most runtime resources are scoped by a **Gateway Group**. When performing operations on runtime resources (`/apisix/admin/*`), you must provide the `gateway_group_id` as a query parameter.

Example: `GET /apisix/admin/routes?gateway_group_id=default`

## Response Formats

### Single Resource
```json
{
  "value": {
    "id": "resource-id",
    "name": "resource-name",
    ...
  }
}
```

### List of Resources (Paginated)
```json
{
  "total": 100,
  "list": [
    {
      "id": "id-1",
      "name": "name-1",
      ...
    },
    ...
  ]
}
```

### Error Response
```json
{
  "message": "Detailed error message"
}
```

## Patch Operations (RFC 6902)

API7 EE uses **JSON Patch** (RFC 6902) for `PATCH` operations. The request body must be an array of patch objects.

```json
[
  { "op": "replace", "path": "/status", "value": 0 },
  { "op": "add", "path": "/labels/env", "value": "prod" }
]
```

## Control-plane Resources (/api)

### 1. Gateway Group (/api/gateway-groups)
Manage logical groupings of gateway instances.
- **Methods**: `GET` (list), `GET /:id`, `POST` (create), `PUT /:id` (update), `DELETE /:id`

### 2. Service Template (/api/services/template)
Design-time service definitions used to generate runtime services.
- **Methods**: `GET` (list), `GET /:id`, `POST`, `PUT /:id`, `DELETE /:id`

## Runtime Resources (/apisix/admin)

All runtime resources require `gateway_group_id` query parameter.

### 1. Route (/apisix/admin/routes)
- **Methods**: `GET` (list), `GET /:id`, `POST` (create), `PUT /:id` (create/update), `PATCH /:id`, `DELETE /:id`
- **Fields**: `id`, `name`, `uris`, `methods`, `host`, `hosts`, `plugins`, `upstream_id`, `service_id`, `status`.

### 2. Upstream (/apisix/admin/upstreams)
- **Methods**: `GET`, `GET /:id`, `POST`, `PUT /:id`, `PATCH /:id`, `DELETE /:id`
- **Fields**: `id`, `name`, `type`, `nodes` (object/array), `checks`, `scheme`.

### 3. Service (/apisix/admin/services)
- **Methods**: `GET`, `GET /:id`, `POST`, `PUT /:id`, `PATCH /:id`, `DELETE /:id`
- **Fields**: `id`, `name`, `upstream_id`, `plugins`.

### 4. Consumer (/apisix/admin/consumers)
- **Methods**: `GET`, `GET /:username`, `PUT` (idempotent create/update), `DELETE /:username`
- **Note**: Identified by `username`.

### 5. SSL (/apisix/admin/ssls)
- **Methods**: `GET`, `GET /:id`, `POST`, `PUT /:id`, `PATCH /:id`, `DELETE /:id`
- **Fields**: `cert`, `key`, `snis`, `status`.

### 6. Global Rule (/apisix/admin/global_rules)
- **Methods**: `GET`, `GET /:id`, `PUT /:id`, `DELETE /:id`

### 7. Stream Route (/apisix/admin/stream_routes)
- **Methods**: `GET`, `GET /:id`, `POST`, `DELETE /:id`

### 8. Plugin Config (/apisix/admin/plugin_configs)
- **Methods**: `GET`, `GET /:id`, `PUT /:id`, `DELETE /:id`

### 9. Plugin Metadata (/apisix/admin/plugin_metadata)
- **Methods**: `GET /:plugin_name`, `PUT /:plugin_name`, `DELETE /:plugin_name`

### 10. Consumer Group (/apisix/admin/consumer_groups)
- **Methods**: `GET`, `GET /:id`, `PUT /:id`, `DELETE /:id`

### 11. Credential (/apisix/admin/consumers/:username/credentials)
- **Methods**: `GET`, `GET /:id`, `PUT /:id`, `DELETE /:id`

### 12. Secret (/apisix/admin/secrets)
- **Methods**: `GET`, `GET /:manager/:id`, `PUT /:manager/:id`, `DELETE /:manager/:id`

### 13. Proto (/apisix/admin/protos)
- **Methods**: `GET`, `GET /:id`, `POST`, `DELETE /:id`

### 14. Plugin (/apisix/admin/plugins)
- **Methods**: 
  - `GET /apisix/admin/plugins/list`: List plugin names.
  - `GET /apisix/admin/plugins/:name`: Get plugin schema.

## Summary of 16 Resource Types

| Resource | Prefix | Identifier | Scope |
|----------|--------|------------|-------|
| Gateway Group | `/api` | `id` | Global |
| Service Template | `/api` | `id` | Global |
| Route | `/apisix/admin` | `id` | Gateway Group |
| Upstream | `/apisix/admin` | `id` | Gateway Group |
| Service | `/apisix/admin` | `id` | Gateway Group |
| Consumer | `/apisix/admin` | `username` | Gateway Group |
| SSL | `/apisix/admin` | `id` | Gateway Group |
| Global Rule | `/apisix/admin` | `id` | Gateway Group |
| Stream Route | `/apisix/admin` | `id` | Gateway Group |
| Plugin Config | `/apisix/admin` | `id` | Gateway Group |
| Plugin Metadata | `/apisix/admin` | `plugin_name`| Gateway Group |
| Consumer Group | `/apisix/admin` | `id` | Gateway Group |
| Credential | `/apisix/admin` | `id` | Consumer |
| Secret | `/apisix/admin` | `manager/id`| Gateway Group |
| Proto | `/apisix/admin` | `id` | Gateway Group |
| Plugin | `/apisix/admin` | `name` | Gateway Group |