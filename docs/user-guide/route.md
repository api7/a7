# Route Management

The `a7 route` command allows you to manage API7 Enterprise Edition (API7 EE) routes. You can list, create, update, get, and delete routes within a specific gateway group using the CLI.

> **Note:** The `--gateway-group` (or `-g`) flag is required for all route commands if not specified in your current context.

## Commands

### `a7 route list`

Lists all routes in the specified gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--page` | | `1` | Page number for pagination |
| `--page-size` | | `20` | Number of items per page |
| `--name` | | | Filter routes by name |
| `--label` | | | Filter routes by label |
| `--uri` | | | Filter routes by URI |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List all routes in the "default" gateway group:
```bash
a7 route list -g default
```

Filter routes by label:
```bash
a7 route list -g default --label env=prod
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

Create a route from a JSON file:
```bash
a7 route create -g default -f route.json
```

**Sample `route.json`:**
```json
{
  "id": "getting-started",
  "name": "example-route",
  "uri": "/get",
  "methods": ["GET"],
  "upstream": {
    "type": "roundrobin",
    "nodes": {
      "httpbin.org:80": 1
    }
  }
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

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--label` | | | Filter routes to export by label |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |
| `--file` | `-f` | | Path to save the exported configuration |

**Examples:**

Export all routes to a YAML file:
```bash
a7 route export -g default -f all-routes.yaml
```

## Configuration Reference

Key fields in the route configuration (sent to `/apisix/admin/routes`):

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier for the route |
| `name` | string | Human-readable name for the route |
| `uri` | string | The URI pattern to match |
| `methods` | array | HTTP methods allowed (e.g., ["GET", "POST"]) |
| `upstream` | object | Inline upstream configuration |
| `upstream_id` | string | Reference to an existing upstream ID |
| `status` | integer | Route status (1 for enabled, 0 for disabled) |
| `plugins` | object | Plugin configurations for the route |
| `labels` | object | Key-value pairs for filtering and organization |

## Examples

### Route with multiple methods and labels

```json
{
  "uri": "/api/v1/*",
  "name": "api-v1-route",
  "methods": ["GET", "POST"],
  "labels": {
    "env": "production",
    "team": "billing"
  },
  "upstream": {
    "type": "roundrobin",
    "nodes": {
      "billing.internal:8080": 1
    }
  }
}
```
