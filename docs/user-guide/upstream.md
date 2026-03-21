# Upstream Management

The `a7 upstream` command allows you to manage API7 Enterprise Edition (API7 EE) upstreams. You can list, create, update, get, and delete upstreams within a specific gateway group using the CLI.

> **Note:** The `--gateway-group` (or `-g`) flag is required for all upstream commands if not specified in your current context.

## Commands

### `a7 upstream list`

Lists all upstreams in the specified gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--page` | | `1` | Page number for pagination |
| `--page-size` | | `20` | Number of items per page |
| `--name` | | | Filter upstreams by name |
| `--label` | | | Filter upstreams by label |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List all upstreams in the "default" gateway group:
```bash
a7 upstream list -g default
```

Filter upstreams by label:
```bash
a7 upstream list -g default --label team=billing
```

### `a7 upstream get <id>`

Gets detailed information about a specific upstream by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Get upstream by ID:
```bash
a7 upstream get 12345 -g default
```

### `a7 upstream create`

Creates a new upstream from a JSON or YAML file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the upstream configuration file (required) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

Create an upstream from a JSON file:
```bash
a7 upstream create -g default -f upstream.json
```

**Sample `upstream.json`:**
```json
{
  "id": "example-upstream",
  "name": "example-upstream",
  "type": "roundrobin",
  "nodes": {
    "httpbin.org:80": 1
  }
}
```

### `a7 upstream update <id>`

Updates an existing upstream using a configuration file or JSON Patch.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the upstream configuration file or JSON Patch file |
| `--patch` | `-p` | | JSON Patch string (RFC 6902) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

Update upstream with ID `12345` using a file:
```bash
a7 upstream update 12345 -g default -f updated-upstream.json
```

### `a7 upstream delete <id>`

Deletes an upstream by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete upstream without confirmation:
```bash
a7 upstream delete 12345 -g default --force
```

### `a7 upstream export`

Exports upstreams from a gateway group to a file or stdout.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--label` | | | Filter upstreams to export by label |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |
| `--file` | `-f` | | Path to save the exported configuration |

**Examples:**

Export all upstreams to a YAML file:
```bash
a7 upstream export -g default -f all-upstreams.yaml
```

### `a7 upstream health` (Planned)

The `health` subcommand is planned for a future release to show the health check status of upstream nodes.

## Configuration Reference

Key fields in the upstream configuration (sent to `/apisix/admin/upstreams`):

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier for the upstream |
| `name` | string | Human-readable name for the upstream |
| `type` | string | Load balancing algorithm (roundrobin, chash, ewma, least_conn) |
| `nodes` | object | Backend nodes with weight (e.g., `{"host:port": weight}`) |
| `service_name` | string | Service name for service discovery |
| `discovery_type` | string | Service discovery type (e.g., dns, consul, nacos) |
| `scheme` | string | Protocol scheme (http, https, grpc, grpcs) |
| `checks` | object | Health check configuration |
| `timeout` | object | Timeout settings (connect, send, read) |
| `labels` | object | Key-value pairs for filtering and organization |
| `status` | integer | Upstream status (1 for enabled, 0 for disabled) |

## Examples

### Upstream with multiple nodes and health checks

```json
{
  "name": "multi-node-upstream",
  "type": "roundrobin",
  "nodes": {
    "127.0.0.1:8080": 1,
    "127.0.0.1:8081": 1
  },
  "checks": {
    "active": {
      "type": "http",
      "http_path": "/health",
      "healthy": {
        "interval": 2,
        "successes": 2
      },
      "unhealthy": {
        "interval": 1,
        "http_failures": 3
      }
    }
  }
}
```
