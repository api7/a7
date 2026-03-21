# Credential

The `a7 credential` command manages API7 Enterprise Edition (EE) consumer credentials. Credentials are nested under a consumer, providing authentication data (like API keys) for that specific user.

All credential operations require specifying both the owner consumer via the `--consumer` flag and the target gateway group via the `--gateway-group` (or `-g`) flag.

## Required Flags

Every `a7 credential` subcommand requires the following flags:

| Flag | Short | Description |
|------|-------|-------------|
| `--consumer` | | Consumer username that owns the credential (required) |
| `--gateway-group` | `-g` | Target gateway group name (required) |

The API path for credentials is:
`/apisix/admin/consumers/{username}/credentials`

## Commands

### `a7 credential list`

Lists all credentials for a specific consumer within a gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--consumer` | | | Consumer username (required) |
| `--gateway-group` | `-g` | | Target gateway group (required) |
| `--page` | | `1` | Page number |
| `--page-size` | | `20` | Page size |
| `--output` | `-o` | `table` | Output format (`table`, `json`, `yaml`) |

**Examples:**

List credentials for consumer `jack` in the `default` group:
```bash
a7 credential list --consumer jack -g default
```

### `a7 credential get`

Gets a specific credential by ID for a consumer.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--consumer` | | | Consumer username (required) |
| `--gateway-group` | `-g` | | Target gateway group (required) |
| `--output` | `-o` | `yaml` | Output format (`json`, `yaml`) |

**Examples:**

Get credential `key-1` for consumer `jack`:
```bash
a7 credential get key-1 --consumer jack -g default
```

### `a7 credential create`

Creates a new credential for a consumer from a JSON or YAML file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--consumer` | | | Consumer username (required) |
| `--gateway-group` | `-g` | | Target gateway group (required) |
| `--file` | `-f` | | Path to credential config file (required) |
| `--output` | `-o` | `yaml` | Output format (`json`, `yaml`) |

**Examples:**

Create a credential from YAML:
```bash
a7 credential create --consumer jack -g default -f key-auth.yaml
```

### `a7 credential update`

Updates an existing credential by ID using a JSON or YAML file. API7 EE uses JSON Patch (RFC 6902) for partial updates.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--consumer` | | | Consumer username (required) |
| `--gateway-group` | `-g` | | Target gateway group (required) |
| `--file` | `-f` | | Path to credential config file (required) |
| `--output` | `-o` | `yaml` | Output format (`json`, `yaml`) |

**Examples:**

Update credential `key-1`:
```bash
a7 credential update key-1 --consumer jack -g default -f updated-key.json
```

### `a7 credential delete`

Deletes a credential by ID for a consumer.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--consumer` | | | Consumer username (required) |
| `--gateway-group` | `-g` | | Target gateway group (required) |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete a credential without confirmation:
```bash
a7 credential delete key-1 --consumer jack -g default --force
```

## Configuration Reference

A credential configuration typically specifies the authentication plugin and its associated data.

### Example: Key Auth Credential

```yaml
id: "key-1"
plugins:
  key-auth:
    key: "my-secret-api-key-123"
```

### Example: JWT Auth Credential

```yaml
id: "jwt-1"
plugins:
  jwt-auth:
    key: "user-jwt-key"
    secret: "super-secret-passphrase"
```

## Examples

### Adding an API Key to a Consumer

1. Create a credential file `key.yaml`:
   ```yaml
   id: "my-api-key"
   plugins:
     key-auth:
       key: "secret-token-val"
   ```

2. Add the credential to consumer `alice` in the `prod` gateway group:
   ```bash
   a7 credential create --consumer alice -g prod -f key.yaml
   ```
