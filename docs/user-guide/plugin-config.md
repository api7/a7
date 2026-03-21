# Plugin Config

The `a7 plugin-config` command allows you to manage API7 Enterprise Edition (EE) plugin configs. Plugin configs are reusable sets of plugins that can be referenced by routes, services, or consumers within a gateway group.

All plugin config commands require the `--gateway-group` (or `-g`) flag.

## Commands

### `a7 plugin-config list`

Lists all plugin configs in the specified gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--page` | | `1` | Page number for pagination |
| `--page-size` | | `20` | Number of items per page |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List all plugin configs in the `default` gateway group:
```bash
a7 plugin-config list -g default
```

Output in JSON format:
```bash
a7 plugin-config list -g default -o json
```

### `a7 plugin-config get`

Gets detailed information about a specific plugin config by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Get plugin config by ID:
```bash
a7 plugin-config get pc-1 -g default
```

### `a7 plugin-config create`

Creates a new plugin config from a JSON or YAML file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the plugin config file (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Create a plugin config from YAML:
```bash
a7 plugin-config create -g default -f plugin-config.yaml
```

### `a7 plugin-config update`

Updates an existing plugin config by ID using a JSON or YAML file. API7 EE uses JSON Patch (RFC 6902) for partial updates when possible, or full replacement.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the plugin config file (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Update plugin config with ID `pc-1`:
```bash
a7 plugin-config update pc-1 -g default -f updated.json
```

### `a7 plugin-config delete`

Deletes a plugin config by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete plugin config without confirmation:
```bash
a7 plugin-config delete pc-1 -g default --force
```

### `a7 plugin-config export`

Exports the plugin config to a file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Output file path |
| `--output` | `-o` | `yaml` | Export format (json, yaml) |

**Examples:**

Export all plugin configs to a YAML file:
```bash
a7 plugin-config export -g default -f all-configs.yaml
```

## Configuration Reference

Key fields in plugin config:

| Field | Description |
|-------|-------------|
| `id` | Unique identifier (string) |
| `name` | Human-readable name |
| `desc` | Description |
| `plugins` | Map of plugin configurations |
| `labels` | User-defined labels |

## Examples

### Reusable Rate Limit Configuration

Create a `rate-limit.yaml` file:

```yaml
id: "global-rate-limit"
name: "global-rate-limit"
desc: "Standard rate limiting for all internal services"
plugins:
  limit-count:
    count: 1000
    time_window: 60
    rejected_code: 429
    key_type: var
    key: remote_addr
labels:
  env: production
  tier: internal
```

Apply it to the `prod` gateway group:

```bash
a7 plugin-config create -g prod -f rate-limit.yaml
```
