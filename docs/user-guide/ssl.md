# SSL Certificate Management

The `a7 ssl` command allows you to manage API7 Enterprise Edition (API7 EE) SSL certificates. You can list, create, update, get, and delete SSL certificates within a specific gateway group using the CLI.

> **Note:** The `--gateway-group` (or `-g`) flag is required for all SSL commands if not specified in your current context.

## Commands

### `a7 ssl list`

Lists all SSL certificates in the specified gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--page` | | `1` | Page number for pagination |
| `--page-size` | | `20` | Number of items per page |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List all SSL certificates in the "default" gateway group:
```bash
a7 ssl list -g default
```

### `a7 ssl get <id>`

Gets detailed information about a specific SSL certificate by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Get SSL certificate by ID:
```bash
a7 ssl get 12345 -g default
```

### `a7 ssl create`

Creates a new SSL certificate from a JSON or YAML file.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the SSL configuration file (required) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

Create an SSL certificate from a JSON file:
```bash
a7 ssl create -g default -f ssl.json
```

**Sample `ssl.json`:**
```json
{
  "id": "example-ssl",
  "cert": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
  "key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
  "snis": ["example.com", "*.example.com"]
}
```

### `a7 ssl update <id>`

Updates an existing SSL certificate using a configuration file or JSON Patch.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the SSL configuration file or JSON Patch file |
| `--patch` | `-p` | | JSON Patch string (RFC 6902) |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

Update SSL certificate with ID `12345` using a file:
```bash
a7 ssl update 12345 -g default -f updated-ssl.json
```

### `a7 ssl delete <id>`

Deletes an SSL certificate by its ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete SSL certificate without confirmation:
```bash
a7 ssl delete 12345 -g default --force
```

### `a7 ssl export`

Exports SSL certificates from a gateway group to a file or stdout.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |
| `--file` | `-f` | | Path to save the exported configuration |

**Examples:**

Export all SSL certificates to a YAML file:
```bash
a7 ssl export -g default -f all-ssls.yaml
```

## Configuration Reference

Key fields in the SSL configuration (sent to `/apisix/admin/ssls`):

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier for the SSL certificate |
| `cert` | string | PEM-encoded server certificate |
| `key` | string | PEM-encoded private key |
| `snis` | array | Array of Server Name Indications |
| `client` | object | mTLS client verification settings (`ca`, `depth`) |
| `type` | string | Certificate type: `server` (default) or `client` |
| `status` | integer | Certificate status: `1` for enabled, `0` for disabled |
| `labels` | object | Key-value labels for the certificate |

## Examples

### SSL with mTLS client verification

```json
{
  "cert": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
  "key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
  "snis": ["secure.example.com"],
  "client": {
    "ca": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
    "depth": 2
  }
}
```
