# Declarative Configuration

The `a7 config` command group provides tools to manage API7 Enterprise Edition (EE) resources using declarative configuration files. This allows you to define your infrastructure as code and synchronize it across different gateway groups.

## Config File Format

The declarative config file supports YAML and JSON. It is organized into sections corresponding to each resource type.

### EE Resource Sections

A7 supports the following resource types in its declarative configuration:

- `routes`
- `services`
- `consumers`
- `ssl`
- `global_rules`
- `stream_routes`
- `protos`
- `secrets`
- `plugin_metadata`
- `gateway_groups` (Control Plane resource)
- `service_templates` (Control Plane resource)
- `credentials` (Nested under consumers)
- `canary_release`

> **Note:** `upstreams`, `plugin_configs`, and `consumer_groups` are not exposed via the API7 EE Admin API and are excluded from declarative config operations.

### Structure with Gateway Groups

When managing runtime resources (like routes or services), the configuration can be scoped to specific gateway groups.

```yaml
version: "1"
gateway_groups:
  - name: "default"
    routes:
      - id: "route-1"
        paths:
          - "/api/v1/*"
        service_id: "service-1"
    services:
      - id: "service-1"
        name: "my-service"
        upstream:
          type: roundrobin
          nodes:
            - host: "127.0.0.1"
              port: 8080
              weight: 1
  - name: "prod"
    routes:
      - id: "route-prod"
        paths:
          - "/prod/*"
```

## Commands

### `a7 config dump`

Dumps resources from the API7 EE Control Plane into a declarative config file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Scope dump to a specific gateway group |
| `--all-groups` | | `false` | Dump resources from ALL gateway groups |
| `--output` | `-o` | `yaml` | Output format (`yaml`, `json`) |
| `--file` | `-f` | | Write output to a file |

**Examples:**

Dump the `default` gateway group to a file:
```bash
a7 config dump -g default -f default-config.yaml
```

Dump everything from the Control Plane:
```bash
a7 config dump --all-groups -f full-backup.yaml
```

### `a7 config validate`

Validates the structure and syntax of a declarative config file.

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--file` | `-f` | Yes | Path to the config file |

**Examples:**

```bash
a7 config validate -f my-config.yaml
```

### `a7 config diff`

Compares a local config file with the current state on the Control Plane.

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--file` | `-f` | Yes | Path to the local config file |
| `--gateway-group` | `-g` | | Scope comparison to a gateway group |
| `--output` | `-o` | | Output format (`json` for machine-readable) |

**Examples:**

```bash
a7 config diff -f updated-config.yaml -g prod
```

### `a7 config sync`

Synchronizes the Control Plane state to match the local config file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | | Path to the local config file (required) |
| `--gateway-group` | `-g` | | Target gateway group |
| `--dry-run` | | `false` | Preview changes without applying |
| `--delete` | | `true` | Delete remote resources not in the local file |

**Examples:**

Preview a sync operation:
```bash
a7 config sync -f new-policy.yaml -g staging --dry-run
```

Apply a full sync:
```bash
a7 config sync -f production.yaml -g prod
```

## Sync Ordering

To ensure referential integrity, `a7 config sync` applies changes in a specific order:

1. `services`
2. `consumers`
3. `credentials`
4. `routes`

When deleting resources (`--delete=true`), the order is reversed to prevent dependency violations.

## Configuration Reference

- `version`: Must be `"1"`.
- `A7_GATEWAY_GROUP`: Environment variable can set the default group for `dump`, `diff`, and `sync`.
- `id`: Most resources use a unique string ID.
- `username`: Consumers are identified by their username.
- `plugin_name`: Plugin metadata is identified by the plugin name.
