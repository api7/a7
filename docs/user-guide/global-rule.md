# Global Rule Management

The `a7 global-rule` command allows you to manage API7 Enterprise Edition (API7 EE) global rules. You can list, create, update, get, and delete global rules within a specific gateway group using the CLI.

> **Note:** The `--gateway-group` (or `-g`) flag is required for all global-rule commands if not specified in your current context.

## Commands

### `a7 global-rule list`

Lists all global rules in the specified gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--page` | | `1` | Page number for pagination |
| `--page-size` | | `20` | Number of items per page |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List all global rules in the "default" gateway group:
```bash
a7 global-rule list -g default
```

### `a7 global-rule get <id>`

Gets detailed information about a specific global rule by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Get global rule by ID:
```bash
a7 global-rule get 1 -g default
```

### `a7 global-rule create`

Creates a new global rule. Each global rule wraps exactly one plugin, and the
rule's `id` is derived from the plugin name (e.g. a rule for the `cors` plugin
gets `id: "cors"`). Setting `id` in the payload is rejected; use `a7
global-rule update` to modify an existing rule.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the global rule configuration file |
| `--plugins-json` | | | Plugins as JSON map |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

One of `--file` or `--plugins-json` must be provided.

**Examples:**

Create a global rule from a JSON file:
```bash
a7 global-rule create -g default -f global-rule.json
```

Create a global rule inline:
```bash
a7 global-rule create -g default --plugins-json '{"prometheus":{}}'
```

**Sample `global-rule.json`:**
```json
{
  "plugins": {
    "prometheus": {}
  }
}
```

### `a7 global-rule update <id>`

Updates an existing global rule using a configuration file or JSON Patch.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the global rule configuration file or JSON Patch file |
| `--patch` | `-p` | | JSON Patch string (RFC 6902) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Update global rule with ID `1` using a file:
```bash
a7 global-rule update 1 -g default -f updated-global-rule.json
```

### `a7 global-rule delete <id>`

Deletes a global rule by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete global rule without confirmation:
```bash
a7 global-rule delete 1 -g default --force
```

### `a7 global-rule export`

Exports global rules from a gateway group to a file or stdout.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |
| `--file` | `-f` | | Path to save the exported configuration |

**Examples:**

Export all global rules to a YAML file:
```bash
a7 global-rule export -g default -f all-global-rules.yaml
```

## Configuration Reference

Key fields in the global rule configuration (sent to `/apisix/admin/global_rules`):

| Field | Type | Description |
|-------|------|-------------|
| `plugins` | object | Plugin configuration. Must contain exactly one plugin; the plugin's key becomes the rule's `id`. |

## Examples

### Global rule for observability

Apply the Prometheus plugin globally to all requests in the gateway group.

```json
{
  "plugins": {
    "prometheus": {}
  }
}
```
