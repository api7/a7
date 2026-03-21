# Secret Manager

The `a7 secret` command allows you to manage API7 Enterprise Edition (EE) secret manager resources. These resources store configurations for external secret backends like HashiCorp Vault, AWS Secrets Manager, or Google Cloud Secret Manager.

All secret commands require the `--gateway-group` (or `-g`) flag.

## Supported Providers

API7 EE supports several secret providers, including:

- `vault`: HashiCorp Vault
- `aws`: AWS Secrets Manager
- `gcp`: Google Cloud Secret Manager
- `azure`: Azure Key Vault

Secret manager IDs use a compound format: `<provider>/<id>`. For example: `vault/my-vault-1`.

## Commands

### `a7 secret list`

Lists all secret managers in the specified gateway group.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--page` | | `1` | Page number for pagination |
| `--page-size` | | `20` | Number of items per page |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |

**Examples:**

List all secrets in the `default` gateway group:
```bash
a7 secret list -g default
```

### `a7 secret get`

Gets a secret manager configuration by its compound ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Get a Vault secret manager:
```bash
a7 secret get vault/prod-vault -g default
```

### `a7 secret create`

Creates a new secret manager from a JSON or YAML file using a compound ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the secret configuration file (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Create an AWS secret manager:
```bash
a7 secret create aws/my-aws -g default -f aws-config.yaml
```

### `a7 secret update`

Updates an existing secret manager by compound ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--file` | `-f` | | Path to the secret configuration file (required) |
| `--output` | `-o` | `yaml` | Output format (json, yaml) |

**Examples:**

Update a Vault secret manager:
```bash
a7 secret update vault/prod-vault -g default -f updated-vault.json
```

### `a7 secret delete`

Deletes a secret manager by its compound ID.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group name (required) |
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

Delete a secret manager:
```bash
a7 secret delete vault/prod-vault -g default --force
```

## Configuration Reference

The content of the configuration file depends on the provider type.

### Vault Configuration

```yaml
uri: "https://vault.example.com:8200"
prefix: "apisix/prod"
token: "hvs.CAES..."
namespace: "eng-team"
```

### AWS Configuration

```yaml
region: "us-west-2"
access_key_id: "AKIA..."
secret_access_key: "wJalrXU..."
```

## Examples

### Referencing Secrets in Plugins

Once a secret manager is configured, you can reference secrets in plugin configurations using the `$secret://<manager-id>/<secret-key>` syntax:

```yaml
# In a route or service plugin config
plugins:
  jwt-auth:
    secret: "$secret://vault/prod-vault/jwt-secret"
```
