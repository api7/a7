# Configuration

The a7 CLI manages connections to API7 Enterprise Edition (API7 EE) instances through a flexible configuration system that supports multiple contexts, environment variables, and command line overrides.

## Config File Location

The a7 CLI stores its configuration in a YAML file. It searches for this file in several locations, following this precedence order:

1. `A7_CONFIG_DIR`: If this environment variable is set, a7 looks for `config.yaml` inside that directory.
2. `XDG_CONFIG_HOME`: If set, a7 uses `$XDG_CONFIG_HOME/a7/config.yaml`.
3. Default: `~/.config/a7/config.yaml` on most systems.

The configuration file is created lazily. It won't exist until you run your first `a7 context create` command.

## Config File Format

The `config.yaml` file stores multiple connection profiles and tracks which one is currently active.

```yaml
current-context: local
contexts:
  - name: local
    server: https://localhost:7443
    token: your-api7-token
    gateway-group: default
    tls-skip-verify: true
  - name: production
    server: https://api7.example.com:7443
    token: prod-token-here
    gateway-group: prod-group
    ca-cert: /path/to/ca.crt
```

## Environment Variables

You can control a7 behavior using environment variables. These are useful for CI/CD pipelines or temporary overrides.

| Variable | Description |
|----------|-------------|
| `A7_SERVER` | The URL of the API7 EE Admin API (e.g., `https://localhost:7443`) |
| `A7_TOKEN` | Your API7 EE token for authentication |
| `A7_GATEWAY_GROUP` | The default gateway group for runtime commands |
| `A7_CONFIG_DIR` | Custom directory for the config file |
| `NO_COLOR` | Disable colored output if set |

## Override Precedence

When multiple configuration sources exist, a7 determines the final value using this priority:

1. Command line flags (e.g., `--server`, `--token`, `--gateway-group`)
2. Environment variables
3. Current context in the config file

For example, to run a one-off command against a specific gateway group:

```bash
a7 route list --gateway-group special-group
```

## Context Management

Contexts allow you to switch between different API7 EE environments or gateway groups quickly.

### create

Create a new context profile.

**Usage:** `a7 context create <name> --server <url> --token <token> --gateway-group <group> [flags]`

**Flags:**

| Flag | Description |
|------|-------------|
| `--server` | API7 EE Admin API server address |
| `--token` | API7 EE token |
| `--gateway-group`, `-g` | Default gateway group name |
| `--tls-skip-verify` | Skip TLS certificate verification |
| `--ca-cert` | Path to CA certificate file |

The first context you create is automatically set as the current context.

```bash
a7 context create prod \
  --server https://1.2.3.4:7443 \
  --token secret-token \
  --gateway-group prod-group
```

### use

Switch to a different context.

**Usage:** `a7 context use <name>`

```bash
a7 context use production
# Output: Switched to context "production".
```

### list

List all available contexts.

**Usage:** `a7 context list` (alias: `ls`)

```bash
a7 context list
```

### delete

Remove a context from the configuration.

**Usage:** `a7 context delete <name> [--force]` (alias: `rm`)

```bash
a7 context delete staging --force
# Output: Context "staging" deleted.
```

### current

Display the name of the active context.

**Usage:** `a7 context current`

```bash
a7 context current
# Output: local
```

## Examples

### Single Instance Setup

For a simple local setup, create one context and start managing resources.

```bash
a7 context create local \
  --server https://localhost:7443 \
  --token your-token \
  --gateway-group default \
  --tls-skip-verify
a7 route list
```

### CI/CD Usage

In automated environments, use environment variables to avoid creating configuration files on disk.

```bash
export A7_SERVER="https://api7-prod:7443"
export A7_TOKEN="prod-secret-token"
export A7_GATEWAY_GROUP="prod-group"
a7 route list
```
