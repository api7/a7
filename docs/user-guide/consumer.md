# Consumer Management

The `a7 consumer` command allows you to manage API7 Enterprise Edition (API7 EE) consumers. You can list, create, update, get, and delete consumers within a specific gateway group using the CLI.

> **Note:** The `--gateway-group` (or `-g`) flag is required for all consumer commands if not specified in your current context.

## Commands

### `a7 consumer list`

Lists all consumers in the specified gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--page` | | `1` | Page number for pagination |
| `--page-size` | | `20` | Number of items per page |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List all consumers in the "default" gateway group:
```bash
a7 consumer list -g default
```

### `a7 consumer get <username>`

Gets detailed information about a specific consumer by its username.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Get consumer by username:
```bash
a7 consumer get my-consumer -g default
```

### `a7 consumer create`

Creates a new consumer from a JSON or YAML file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the consumer configuration file (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Create a consumer from a JSON file:
```bash
a7 consumer create -g default -f consumer.json
```

**Sample `consumer.json`:**
```json
{
  "username": "my-consumer",
  "desc": "My API consumer",
  "plugins": {
    "key-auth": {
      "key": "my-secret-key"
    }
  }
}
```

### `a7 consumer update <username>`

Updates an existing consumer using a configuration file or JSON Patch.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the consumer configuration file or JSON Patch file |
| `--patch` | `-p` | | JSON Patch string (RFC 6902) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Update consumer "my-consumer" using a file:
```bash
a7 consumer update my-consumer -g default -f updated-consumer.json
```

### `a7 consumer delete <username>`

Deletes a consumer by its username.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete consumer without confirmation:
```bash
a7 consumer delete my-consumer -g default --force
```

### `a7 consumer export`

Exports consumers from a gateway group to a file or stdout.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |
| `--file` | `-f` | | Path to save the exported configuration |

**Examples:**

Export all consumers to a YAML file:
```bash
a7 consumer export -g default -f all-consumers.yaml
```

### `a7 credential`

Credential management is handled via the separate `credential` command, nested under consumers in API7 EE.

**Usage:** `a7 consumer credential [command] -g <gateway-group>`

Commands: `list`, `get`, `create`, `update`, `delete`.

## Configuration Reference

Key fields in the consumer configuration (sent to `/apisix/admin/consumers`):

| Field | Type | Description |
|-------|------|-------------|
| `username` | string | Unique identifier for the consumer |
| `desc` | string | Human-readable description |
| `plugins` | object | Plugin configurations (e.g., key-auth, jwt-auth) |
| `group_id` | string | Reference to a consumer group |
| `labels` | object | Key-value labels for the consumer |

## Examples

### Consumer with key-auth

```json
{
  "username": "api-user",
  "desc": "API key authenticated user",
  "plugins": {
    "key-auth": {
      "key": "secret-api-key-123"
    }
  }
}
```
