# a7 — CLI for API7 Enterprise Edition

> **⚠️ This project is under active development and is NOT production-ready.** APIs, commands, and output formats may change without notice.

`a7` is a command-line tool for managing [API7 Enterprise Edition](https://api7.ai) API Gateway from your terminal. It wraps both the API7 EE control-plane API (`/api/*`) and the APISIX admin API (`/apisix/admin/*`) to provide convenient, scriptable access to gateway groups, routes, services, upstreams, consumers, SSL certificates, plugins, and more.

## Features

- **Resource CRUD** — Create, list, get, update, and delete 16 API7 EE resource types:
  - **Control Plane**: Gateway Group, Service Template
  - **Runtime**: Route, Upstream, Service, Consumer, SSL Certificate, Plugin, Global Rule, Stream Route, Plugin Config, Plugin Metadata, Consumer Group, Credential, Secret, Proto
- **Context management** — Switch between multiple API7 EE instances (`a7 context create`, `a7 context use`, `a7 context list`)
- **Gateway group scoping** — All runtime operations are scoped to a gateway group via `--gateway-group` flag or context config
- **Rich output** — Human-friendly tables in TTY, machine-readable JSON/YAML in pipes (`--output json|yaml|table`)
- **Shell completions** — Bash, Zsh, Fish, PowerShell (`a7 completion`)

## Installation

### Using Go

If you have Go 1.22+ installed:

```bash
go install github.com/api7/a7/cmd/a7@latest
```

This installs the `a7` binary to your `$GOPATH/bin` (or `$HOME/go/bin` by default). Make sure it's in your `PATH`.

### Build from Source

```bash
git clone https://github.com/api7/a7.git
cd a7
make build
# Binary is at ./bin/a7
```

## Quick Start

### Prerequisites

- `a7` installed (see [Installation](#installation) above)
- A running API7 Enterprise Edition instance with Admin API access
- An API7 EE access token

### 1. Configure a context

```bash
# Create a context pointing to your API7 EE instance
a7 context create dev \
  --server https://api7ee.example.com \
  --token a7ee.your-access-token \
  --gateway-group default

# Verify the active context
a7 context current

# List all contexts
a7 context list
```

### 2. Explore gateway groups

```bash
# List all gateway groups
a7 gateway-group list

# Get details of a specific gateway group
a7 gateway-group get <id> --output json
```

### 3. Create resources

```bash
# Create an upstream
a7 upstream create \
  --name my-upstream \
  --type roundrobin \
  --nodes "httpbin.org:80=1"

# Create a route
a7 route create \
  --name my-route \
  --uri "/api/v1/*" \
  --methods GET,POST \
  --upstream-id <upstream-id>
```

### 4. Read and explore

```bash
# List all routes (table output in terminal)
a7 route list

# Get a specific route in JSON
a7 route get <id> --output json

# Get it in YAML
a7 route get <id> --output yaml

# List available plugins
a7 plugin list
```

### 5. Update a resource

```bash
# Update the route description
a7 route update <id> \
  --name my-route \
  --desc "Updated route for API v1"
```

### 6. Clean up

```bash
# Delete the route and upstream
a7 route delete <id>
a7 upstream delete <id>

# Verify they're gone
a7 route list
a7 upstream list
```

## Commands

### Utility

| Command | Description |
|---------|-------------|
| `a7 version` | Print CLI version |
| `a7 completion <shell>` | Generate shell completions (bash, zsh, fish, powershell) |
| `a7 context create` | Create a connection context |
| `a7 context list` | List all contexts |
| `a7 context use <name>` | Switch active context |
| `a7 context current` | Show current context |
| `a7 context delete <name>` | Delete a context |

### Control Plane Resources

| Command | Alias | Actions | Description |
|---------|-------|---------|-------------|
| `a7 gateway-group` | `gg` | list, get, create, update, delete | Manage gateway groups |
| `a7 service-template` | `st` | list, get, create, update, delete, publish | Manage service templates |

### Runtime Resources

| Command | Alias | Actions | Description |
|---------|-------|---------|-------------|
| `a7 route` | `rt` | list, get, create, update, delete | Manage routes |
| `a7 upstream` | `us` | list, get, create, update, delete | Manage upstreams |
| `a7 service` | `svc` | list, get, create, update, delete | Manage runtime services |
| `a7 consumer` | `c` | list, get, create, update, delete | Manage consumers |
| `a7 ssl` | — | list, get, create, update, delete | Manage SSL certificates |
| `a7 plugin` | `pl` | list, get | Manage plugins |
| `a7 global-rule` | `gr` | list, get, create, update, delete | Manage global rules |
| `a7 stream-route` | `sr` | list, get, create, update, delete | Manage stream routes |
| `a7 plugin-config` | `pc` | list, get, create, update, delete | Manage plugin configs |
| `a7 plugin-metadata` | `pm` | get, create, update, delete | Manage plugin metadata |
| `a7 consumer-group` | `cg` | list, get, create, update, delete | Manage consumer groups |
| `a7 credential` | `cred` | list, get, create, update, delete | Manage consumer credentials |
| `a7 secret` | `sec` | list, get, create, update, delete | Manage secret providers |
| `a7 proto` | `pb` | list, get, create, update, delete | Manage protobuf definitions |

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

## Requirements

- Go 1.22+
- API7 Enterprise Edition 3.x with Admin API enabled

## Development

```bash
make build          # Build binary to ./bin/a7
make test           # Run unit tests
make test-verbose   # Tests with verbose output
make lint           # Run linter
make fmt            # Format code
make vet            # Run go vet
make check          # fmt + vet + lint + test
```

See [AGENTS.md](AGENTS.md) for the full development guide, coding conventions, and how to add new commands.

## Documentation

- [Product Requirements](PRD.md)
- [AI Agent Guide](AGENTS.md)

## Contributing

Contributions are welcome. Please see [AGENTS.md](AGENTS.md) for development workflow and coding conventions.

## License

[Apache License 2.0](LICENSE)
