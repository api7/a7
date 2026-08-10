# Gateway Group

The `a7 gateway-group` command allows you to manage API7 Enterprise Edition (EE) gateway groups. Gateway groups are logical groupings of gateway instances (APISIX nodes) that share the same configuration.

Unlike runtime resources (like routes or services), gateway groups are Control Plane resources. Therefore, you do not need to specify a `--gateway-group` flag when managing the gateway groups themselves.

## Commands

### `a7 gateway-group list`

Lists all gateway groups configured in the API7 EE Control Plane.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--label` | | | Filter by label (`key=value`) |
| `--output` | `-o` | `table` | Output format (`table`, `json`, `yaml`) |

**Examples:**

List all gateway groups:
```bash
a7 gateway-group list
```

Filter by environment label:
```bash
a7 gateway-group list --label env=prod
```

### `a7 gateway-group get`

Gets detailed information about a specific gateway group by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `table` | Output format (`table`, `json`, `yaml`) |

**Examples:**

```bash
a7 gateway-group get <gateway-group-id>
```

### `a7 gateway-group create`

Creates a new gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--name` | | | Name of the gateway group (required) |
| `--description` | | | Description |
| `--file` | `-f` | | Path to a configuration file |
| `--labels` | | | Label in `key=value` format (repeatable) |

**Examples:**

Create a gateway group via flags:
```bash
a7 gateway-group create --name staging --description "Staging environment"
```

### `a7 gateway-group update`

Updates an existing gateway group's configuration.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--name` | | | New name |
| `--description` | | | New description |
| `--file` | `-f` | | Path to a configuration file |
| `--labels` | | | Label in `key=value` format (repeatable) |

**Examples:**

```bash
a7 gateway-group update <gateway-group-id> --description "Updated staging description"
```

### `a7 gateway-group delete`

Deletes a gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

```bash
a7 gateway-group delete <gateway-group-id> --force
```

## Scoping Runtime Resources

Gateway groups are used to scope runtime resources. Most other `a7` commands require the `--gateway-group` flag to specify which group the operation should apply to.

```bash
# Get a route in a gateway group
a7 route get route-1 --gateway-group <gateway-group-id>
```

You can also set a default gateway group in your context configuration or via the `A7_GATEWAY_GROUP` environment variable.
