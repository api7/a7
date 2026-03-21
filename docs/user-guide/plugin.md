# Plugin Management

The `a7 plugin` command allows you to inspect API7 Enterprise Edition (API7 EE) plugins. Plugin commands are read-only and are used to list available plugins and retrieve their configuration schemas.

> **Note:** The `--gateway-group` (or `-g`) flag is required for all plugin commands if not specified in your current context.

## Commands

### `a7 plugin list`

Lists available plugins in the specified gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--subsystem` | | | Filter by subsystem (`http` or `stream`) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List all plugins in the "default" gateway group:
```bash
a7 plugin list -g default
```

List available AI Gateway plugins:
```bash
# AI Gateway plugins are categorized under the HTTP subsystem
a7 plugin list -g default --subsystem http
```

### `a7 plugin get <name>`

Gets the configuration schema for a specific plugin by name.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `json` | Output format (json, yaml) |

**Examples:**

Get schema for the `ai-proxy` plugin:
```bash
a7 plugin get ai-proxy -g default
```

Get schema for `key-auth` in YAML format:
```bash
a7 plugin get key-auth -g default -o yaml
```

## Plugin Categories

API7 Enterprise Edition includes several categories of plugins:

- **Authentication:** `key-auth`, `jwt-auth`, `openid-connect`, etc.
- **Security:** `ip-restriction`, `cors`, `uri-blocker`, etc.
- **Traffic Control:** `limit-count`, `limit-req`, `proxy-cache`, etc.
- **AI Gateway (EE-specific):** `ai-proxy`, `ai-rag`, `ai-token-ratelimit`, etc.
- **Observability:** `prometheus`, `zipkin`, `skywalking`, etc.

## Notes

- `a7 plugin list` and `a7 plugin get` are read-only commands.
- The plugin list returns a direct array of available plugin names for the specified gateway group.
- The schema returned by `a7 plugin get` can be used as a reference when configuring plugins in routes, services, or global rules.
