# a7 — CLI for API7 Enterprise Edition

A command-line tool for managing [API7 Enterprise Edition](https://api7.ai) API Gateway resources. Supports both control-plane (`/api/*`) and runtime (`/apisix/admin/*`) operations.

## Installation

### From Source

```bash
go install github.com/api7/a7/cmd/a7@latest
```

### Build Locally

```bash
git clone https://github.com/api7/a7.git
cd a7
make build
# Binary at ./bin/a7
```

### Releases

Download pre-built binaries from the [Releases](https://github.com/api7/a7/releases) page.

## Quick Start

### 1. Create a context

```bash
a7 context create dev \
  --server https://api7ee.example.com \
  --token a7ee.your-access-token \
  --gateway-group default
```

### 2. List gateway groups

```bash
a7 gateway-group list
```

### 3. List routes in a gateway group

```bash
a7 route list --gateway-group default
```

### 4. Create a route

```bash
a7 route create \
  --name my-route \
  --uri /api/v1/* \
  --methods GET,POST \
  --upstream-id my-upstream
```

### 5. Export as JSON

```bash
a7 route list -o json
```

## Commands

### Utility

| Command | Alias | Description |
|---------|-------|-------------|
| `a7 version` | | Print CLI version |
| `a7 completion <shell>` | | Generate shell completions |
| `a7 context create` | | Create a connection context |
| `a7 context list` | | List all contexts |
| `a7 context use <name>` | | Switch active context |
| `a7 context current` | | Show current context |
| `a7 context delete <name>` | | Delete a context |

### Control Plane Resources

| Command | Alias | Description |
|---------|-------|-------------|
| `a7 gateway-group <action>` | `gg` | Manage gateway groups |
| `a7 service-template <action>` | `st` | Manage service templates |

### Runtime Resources

| Command | Alias | Description |
|---------|-------|-------------|
| `a7 route <action>` | `rt` | Manage routes |
| `a7 upstream <action>` | `us` | Manage upstreams |
| `a7 consumer <action>` | `c` | Manage consumers |
| `a7 ssl <action>` | | Manage SSL certificates |
| `a7 plugin list` | `pl` | List available plugins |

Each resource command supports: `list`, `get`, `create`, `update`, `delete`.

## Authentication

a7 supports API7 EE access tokens via the `X-API-KEY` header.

**Precedence** (highest to lowest):
1. `--token` flag
2. `A7_TOKEN` environment variable
3. Context config (`~/.config/a7/config.yaml`)

### Token Types

| Prefix | Type | Scope |
|--------|------|-------|
| `a7ee` | Access Token | Full control-plane + admin API |
| `a7adm` | Gateway Admin Key | Runtime API only (gateway-group scoped) |

## Configuration

Config file: `~/.config/a7/config.yaml`

```yaml
current-context: dev
contexts:
  - name: dev
    server: https://api7ee.dev.example.com
    token: a7ee.xxxxx
    gateway-group: default
  - name: prod
    server: https://api7ee.prod.example.com
    token: a7ee.yyyyy
    gateway-group: production
    tls-skip-verify: false
    ca-cert: /path/to/ca.pem
```

Override the config directory with `A7_CONFIG_DIR` or `XDG_CONFIG_HOME`.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `A7_TOKEN` | API access token |
| `A7_SERVER` | Control-plane server URL |
| `A7_GATEWAY_GROUP` | Default gateway group |
| `A7_CONFIG_DIR` | Config directory path |
| `NO_COLOR` | Disable color output |

## Global Flags

```
--server          API7 EE server URL (overrides context)
--token           API access token (overrides context)
--gateway-group   Default gateway group (overrides context)
-o, --output      Output format: json, yaml (default: table)
```

## Development

```bash
make build          # Build binary
make test           # Run tests
make test-verbose   # Tests with verbose output
make lint           # Run linter
make fmt            # Format code
make vet            # Run go vet
make check          # fmt + vet + lint + test
```

See [AGENTS.md](AGENTS.md) for the full development guide.

## License

Apache 2.0 — see [LICENSE](LICENSE).
