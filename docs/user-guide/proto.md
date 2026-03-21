# Proto Management

The `a7 proto` command allows you to manage API7 Enterprise Edition (API7 EE) proto definitions used by gRPC-related integrations. You can list, create, update, get, and delete proto resources within a specific gateway group using the CLI.

> **Note:** The `--gateway-group` (or `-g`) flag is required for all proto commands if not specified in your current context.

## Commands

### `a7 proto list`

Lists proto definitions in the specified gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--page` | | `1` | Page number for pagination |
| `--page-size` | | `20` | Number of items per page |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List proto definitions in the "default" gateway group:
```bash
a7 proto list -g default
```

### `a7 proto get <id>`

Gets detailed information about a proto definition by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Get a proto definition:
```bash
a7 proto get helloworld-proto -g default
```

### `a7 proto create`

Creates a proto definition from a JSON or YAML file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the proto configuration file (required) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

Create a proto definition from a JSON file:
```bash
a7 proto create -g default -f proto.json
```

**Sample `proto.json`:**
```json
{
  "id": "helloworld-proto",
  "name": "helloworld",
  "content": "syntax = \"proto3\";\npackage helloworld;\nservice Greeter {\n  rpc SayHello (HelloRequest) returns (HelloReply) {}\n}\nmessage HelloRequest { string name = 1; }\nmessage HelloReply { string message = 1; }"
}
```

### `a7 proto update <id>`

Updates an existing proto definition using a configuration file or JSON Patch.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the proto configuration file or JSON Patch file |
| `--patch` | `-p` | | JSON Patch string (RFC 6902) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

Update proto `helloworld-proto` using a file:
```bash
a7 proto update helloworld-proto -g default -f updated-proto.json
```

### `a7 proto delete <id>`

Deletes a proto definition by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete proto without confirmation:
```bash
a7 proto delete helloworld-proto -g default --force
```

### `a7 proto export`

Exports proto definitions from a gateway group to a file or stdout.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |
| `--file` | `-f` | | Path to save the exported configuration |

**Examples:**

Export all proto definitions to a YAML file:
```bash
a7 proto export -g default -f all-protos.yaml
```

## Configuration Reference

Key fields in the proto configuration (sent to `/apisix/admin/protos`):

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier for the proto definition |
| `name` | string | Human-readable name |
| `content` | string | Full Protocol Buffer definition text |
| `desc` | string | Description of the proto resource |
| `labels` | object | Key-value pairs for filtering and organization |

## Examples

### Proto and gRPC Transcoding

Proto resources are commonly referenced by the `grpc-transcode` plugin. Once defined, they can be used in route configurations:

```json
{
  "uri": "/hello",
  "plugins": {
    "grpc-transcode": {
      "proto_id": "helloworld-proto",
      "service": "helloworld.Greeter",
      "method": "SayHello"
    }
  },
  "upstream": {
    "scheme": "grpc",
    "nodes": {
      "127.0.0.1:50051": 1
    }
  }
}
```
