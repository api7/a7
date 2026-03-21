# Stream Route Management

The `a7 stream-route` command allows you to manage API7 Enterprise Edition (API7 EE) stream routes (L4 TCP/UDP). You can list, create, update, get, and delete stream routes within a specific gateway group using the CLI.

> **Note:** The `--gateway-group` (or `-g`) flag is required for all stream-route commands if not specified in your current context.

## Commands

### `a7 stream-route list`

Lists all stream routes in the specified gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--page` | | `1` | Page number for pagination |
| `--page-size` | | `20` | Number of items per page |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List all stream routes in the "default" gateway group:
```bash
a7 stream-route list -g default
```

### `a7 stream-route get <id>`

Gets detailed information about a specific stream route by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Get stream route by ID:
```bash
a7 stream-route get 1 -g default
```

### `a7 stream-route create`

Creates a new stream route from a JSON or YAML file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the stream route configuration file (required) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

Create a stream route from a JSON file:
```bash
a7 stream-route create -g default -f stream-route.json
```

**Sample `stream-route.json`:**
```json
{
  "id": "tcp-proxy",
  "name": "tcp-proxy",
  "server_port": 9100,
  "upstream": {
    "type": "roundrobin",
    "nodes": {
      "127.0.0.1:8080": 1
    }
  }
}
```

### `a7 stream-route update <id>`

Updates an existing stream route using a configuration file or JSON Patch.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the stream route configuration file or JSON Patch file |
| `--patch` | `-p` | | JSON Patch string (RFC 6902) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

Update stream route with ID `1` using a file:
```bash
a7 stream-route update 1 -g default -f updated-stream-route.json
```

### `a7 stream-route delete <id>`

Deletes a stream route by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete stream route without confirmation:
```bash
a7 stream-route delete 1 -g default --force
```

### `a7 stream-route export`

Exports stream routes from a gateway group to a file or stdout.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |
| `--file` | `-f` | | Path to save the exported configuration |

**Examples:**

Export all stream routes to a YAML file:
```bash
a7 stream-route export -g default -f all-stream-routes.yaml
```

## Configuration Reference

Key fields in the stream route configuration (sent to `/apisix/admin/stream_routes`):

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier for the stream route |
| `name` | string | Human-readable name for the stream route |
| `remote_addr` | string | Match source client address |
| `server_addr` | string | Match destination server address |
| `server_port` | integer | Match destination server port |
| `sni` | string | Match TLS SNI |
| `upstream` | object | Inline upstream configuration |
| `upstream_id` | string | Reference to an existing upstream ID |
| `plugins` | object | Plugin configurations |
| `labels` | object | Key-value pairs for filtering and organization |

## Examples

### Stream route with SNI matching

```json
{
  "name": "sni-route",
  "sni": "tcp.example.com",
  "server_port": 9443,
  "upstream": {
    "type": "roundrobin",
    "nodes": {
      "127.0.0.1:9000": 1
    }
  }
}
```
