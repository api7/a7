# Plugin Metadata

The `a7 plugin-metadata` command allows you to manage API7 Enterprise Edition (EE) plugin metadata.

Plugin metadata is configured per plugin name and is scoped to a specific gateway group. It is stored at `/apisix/admin/plugin_metadata/{plugin_name}` within the targeted gateway group.

Unlike other resources, plugin metadata does not have a list endpoint. Operations are performed by specifying the plugin name directly.

## Commands

All plugin metadata commands require the `--gateway-group` (or `-g`) flag to specify the target gateway group.

### `a7 plugin-metadata get`

Gets metadata for a specific plugin by name.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--token` | `-t` | | API token (can be set via A7_TOKEN) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Get metadata for the `syslog` plugin in the `default` gateway group:
```bash
a7 plugin-metadata get syslog -g default
```

Get metadata in JSON format:
```bash
a7 plugin-metadata get syslog -g default -o json
```

### `a7 plugin-metadata create`

Creates or sets plugin metadata for a plugin from a JSON or YAML file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the plugin metadata file (required) |
| `--token` | `-t` | | API token |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Create metadata for `syslog` from a JSON file:
```bash
a7 plugin-metadata create syslog -g default -f metadata.json
```

Create metadata for `syslog` from a YAML file:
```bash
a7 plugin-metadata create syslog -g prod -f metadata.yaml
```

### `a7 plugin-metadata update`

Updates existing plugin metadata for a plugin using a JSON or YAML file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the plugin metadata file (required) |
| `--token` | `-t` | | API token |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Update metadata for `syslog`:
```bash
a7 plugin-metadata update syslog -g default -f metadata-updated.json
```

### `a7 plugin-metadata delete`

Deletes plugin metadata for a specific plugin name.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--token` | `-t` | | API token |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete metadata for `syslog` with confirmation:
```bash
a7 plugin-metadata delete syslog -g default
```

Delete metadata for `syslog` without confirmation:
```bash
a7 plugin-metadata delete syslog -g default --force
```

## Configuration Reference

The following environment variables can be used to configure the `a7` CLI for plugin metadata operations:

| Environment Variable | Description |
|----------------------|-------------|
| `A7_SERVER` | API7 EE Control Plane URL (e.g., `https://localhost:7443`) |
| `A7_TOKEN` | API token for authentication |
| `A7_GATEWAY_GROUP` | Default gateway group to use if `--gateway-group` is omitted |

## Examples

### Configuring Log Format for Syslog

Create a `syslog-metadata.yaml` file:

```yaml
log_format:
  host: "$host"
  request_id: "$request_id"
  client_ip: "$remote_addr"
```

Apply the metadata to the `default` gateway group:

```bash
a7 plugin-metadata create syslog -g default -f syslog-metadata.yaml
```
