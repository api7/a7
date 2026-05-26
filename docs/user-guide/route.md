# Route Management

The `a7 route` command allows you to manage API7 Enterprise Edition (API7 EE) routes. You can list, create, update, get, and delete routes within a specific gateway group using the CLI.

> **Note:** The `--gateway-group` (or `-g`) flag is required for all route commands if not specified in your current context.

## Commands

### `a7 route list`

Lists routes in the specified gateway group. By default, every route across
every service in the gateway group is returned; pass `--service-id` to
narrow the result to a single service.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--label` | | | Filter routes by label |
| `--service-id` | | | Filter routes to a single service |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List every route in the "default" gateway group:
```bash
a7 route list -g default
```

List routes for a single service:
```bash
a7 route list -g default --service-id example-service
```

Filter routes by label within a service:
```bash
a7 route list -g default --service-id example-service --label env=prod
```

### `a7 route get <id>`

Gets detailed information about a specific route by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Get route by ID:
```bash
a7 route get 12345 -g default
```

### `a7 route create`

Creates a new route from a JSON or YAML file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the route configuration file (required) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

Create a service first:
```bash
a7 service create -g default -f service.json
```

Create a route from a JSON file:
```bash
a7 route create -g default -f route.json
```

**Sample `route.json`:**
```json
{
  "id": "getting-started",
  "name": "example-route",
  "paths": ["/get"],
  "methods": ["GET"],
  "service_id": "example-service"
}
```

### `a7 route update <id>`

Updates an existing route using a configuration file or JSON Patch.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the route configuration file or JSON Patch file |
| `--patch` | `-p` | | JSON Patch string (RFC 6902) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

Update route with ID `12345` using a file:
```bash
a7 route update 12345 -g default -f updated-route.json
```

Update route using JSON Patch:
```bash
a7 route update 12345 -g default --patch '[{"op": "replace", "path": "/status", "value": 0}]'
```

### `a7 route delete <id>`

Deletes a route by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete route without confirmation:
```bash
a7 route delete 12345 -g default --force
```

### `a7 route export`

Exports routes from a gateway group to a file or stdout.
API7 EE requires `--service-id` for route export because routes are scoped by
service.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--service-id` | | | Service ID whose routes should be exported (required by API7 EE) |
| `--label` | | | Filter routes to export by label |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |
| `--file` | `-f` | | Path to save the exported configuration |

**Examples:**

Export routes for a service to a YAML file:
```bash
a7 route export -g default --service-id example-service -f routes.yaml
```

## Configuration Reference

Key fields in the route configuration (sent to `/apisix/admin/routes`):

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier for the route |
| `name` | string | Human-readable name for the route |
| `paths` | array | Path patterns to match |
| `uri` | string | Legacy URI pattern accepted by some APISIX-compatible payloads |
| `methods` | array | HTTP methods allowed (e.g., ["GET", "POST"]) |
| `service_id` | string | Required by current API7 EE; reference to the service that owns the upstream configuration |
| `status` | integer | Route status (1 for enabled, 0 for disabled) |
| `plugins` | object | Plugin configurations for the route |
| `labels` | object | Key-value pairs for filtering and organization |

## Examples

### Route with multiple methods and labels

```json
{
  "paths": ["/api/v1/*"],
  "name": "api-v1-route",
  "service_id": "billing-service",
  "methods": ["GET", "POST"],
  "labels": {
    "env": "production",
    "team": "billing"
  }
}
```
