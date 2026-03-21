# Service Management

The `a7 service` command allows you to manage API7 Enterprise Edition (API7 EE) runtime services. You can list, create, update, get, and delete services within a specific gateway group using the CLI.

> **Note:** This command manages **runtime services** via the `/apisix/admin/services` endpoint. These are different from **service templates** which are used for control-plane design-time configurations.
>
> The `--gateway-group` (or `-g`) flag is required for all service commands if not specified in your current context.

## Commands

### `a7 service list`

Lists all services in the specified gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--page` | | `1` | Page number for pagination |
| `--page-size` | | `20` | Number of items per page |
| `--name` | | | Filter services by name |
| `--label` | | | Filter services by label |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List all services in the "default" gateway group:
```bash
a7 service list -g default
```

Filter services by label:
```bash
a7 service list -g default --label env=staging
```

### `a7 service get <id>`

Gets detailed information about a specific service by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Get service by ID:
```bash
a7 service get 12345 -g default
```

### `a7 service create`

Creates a new service from a JSON or YAML file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the service configuration file (required) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

Create a service from a JSON file:
```bash
a7 service create -g default -f service.json
```

**Sample `service.json`:**
```json
{
  "id": "example-service",
  "name": "example-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": {
      "httpbin.org:80": 1
    }
  }
}
```

### `a7 service update <id>`

Updates an existing service using a configuration file or JSON Patch.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the service configuration file or JSON Patch file |
| `--patch` | `-p` | | JSON Patch string (RFC 6902) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

Update service with ID `12345` using a file:
```bash
a7 service update 12345 -g default -f updated-service.json
```

### `a7 service delete <id>`

Deletes a service by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete service without confirmation:
```bash
a7 service delete 12345 -g default --force
```

### `a7 service export`

Exports services from a gateway group to a file or stdout.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--label` | | | Filter services to export by label |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |
| `--file` | `-f` | | Path to save the exported configuration |

**Examples:**

Export all services to a YAML file:
```bash
a7 service export -g default -f all-services.yaml
```

## Configuration Reference

Key fields in the service configuration (sent to `/apisix/admin/services`):

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier for the service |
| `name` | string | Human-readable name for the service |
| `desc` | string | Description of the service |
| `upstream` | object | Inline upstream configuration |
| `upstream_id` | string | Reference to an existing upstream ID |
| `status` | integer | Service status (1 for enabled, 0 for disabled) |
| `plugins` | object | Plugin configurations for the service |
| `hosts` | array | List of hostnames the service handles |
| `labels` | object | Key-value pairs for filtering and organization |

## Examples

### Service with plugins

```json
{
  "name": "protected-service",
  "upstream": {
    "type": "roundrobin",
    "nodes": {
      "127.0.0.1:8080": 1
    }
  },
  "plugins": {
    "limit-count": {
      "count": 100,
      "time_window": 60,
      "rejected_code": 429
    },
    "key-auth": {}
  }
}
```
