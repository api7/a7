# Consumer Group

The `a7 consumer-group` command allows you to manage API7 Enterprise Edition (EE) consumer groups. Consumer groups are reusable sets of plugins that can be referenced by multiple consumers through the consumer `group_id` field.

All consumer group commands require the `--gateway-group` (or `-g`) flag.

## Commands

### `a7 consumer-group list`

Lists all consumer groups in the specified gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--page` | | `1` | Page number for pagination |
| `--page-size` | | `20` | Number of items per page |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List all consumer groups in the `default` gateway group:
```bash
a7 consumer-group list -g default
```

### `a7 consumer-group get`

Gets detailed information about a specific consumer group by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Get consumer group by ID:
```bash
a7 consumer-group get cg-gold -g default
```

### `a7 consumer-group create`

Creates a new consumer group from a JSON or YAML file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the consumer group configuration file (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Create a consumer group from a YAML file:
```bash
a7 consumer-group create -g default -f gold-tier.yaml
```

### `a7 consumer-group update`

Updates an existing consumer group by ID using a configuration file. API7 EE uses JSON Patch (RFC 6902) for partial updates.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the consumer group configuration file (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Update consumer group with ID `cg-gold`:
```bash
a7 consumer-group update cg-gold -g default -f updated-gold.json
```

### `a7 consumer-group delete`

Deletes a consumer group by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete consumer group:
```bash
a7 consumer-group delete cg-gold -g default --force
```

### `a7 consumer-group export`

Exports the consumer group to a file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Output file path |
| `--output` | `-o` | `yaml` | Export format (json, yaml) |

**Examples:**

Export all consumer groups to a YAML file:
```bash
a7 consumer-group export -g default -f consumer-groups-backup.yaml
```

## Configuration Reference

Key fields in the consumer group configuration:

| Field | Description |
|-------|-------------|
| `id` | Unique identifier (string) |
| `name` | Human-readable name |
| `desc` | Description |
| `plugins` | Plugin configurations shared by consumers in the group |
| `labels` | Optional key/value labels |

## Examples

### Creating a Tiered Consumer Group

Create a `silver-tier.yaml` file:

```yaml
id: cg-silver
name: silver
desc: Consumer group for silver tier users with moderate rate limits
plugins:
  limit-count:
    count: 500
    time_window: 60
    rejected_code: 429
    key_type: var
    key: remote_addr
labels:
  tier: silver
```

Apply it to the `prod` gateway group:

```bash
a7 consumer-group create -g prod -f silver-tier.yaml
```

### Binding a Consumer to a Group

To bind a consumer to a consumer group, set the `group_id` field in the consumer configuration:

```yaml
username: app-user-1
group_id: cg-silver
```

Then apply it with the `a7 consumer` command:

```bash
a7 consumer update app-user-1 -g prod -f consumer-config.yaml
```
