---
name: a7-shared
description: >-
  Core skill for working with the a7 CLI — the command-line tool for API7 Enterprise Edition.
  Provides project conventions, command patterns, dual-API architecture, and
  development workflow. Load this skill when working on a7 source code, adding
  new commands, writing tests, or modifying any a7 component.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: shared
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 route
    - a7 service
    - a7 consumer
    - a7 ssl
    - a7 plugin
    - a7 gateway-group
    - a7 service-template
    - a7 config
    - a7 context
---

# a7 Shared Skill

## What is a7

a7 is a Go CLI for API7 Enterprise Edition (API7 EE). It provides imperative CRUD
for 13 resource types, declarative config sync, context management, and debug tooling.

- **Binary**: `a7`
- **Module**: `github.com/api7/a7`
- **Go**: 1.22+
- **Pattern**: noun-verb (`a7 <resource> <action> [flags]`)
- **Dual-API Architecture**:
  - **Control-plane API**: `/api/*` (e.g., gateway groups, users, service templates)
  - **Runtime Admin API**: `/apisix/admin/*` (e.g., routes, upstreams, services)
- **Gateway Group Scoping**: All runtime resources must be scoped to a gateway group using `--gateway-group` or `-g`.

## Project Layout

```
a7/
├── cmd/a7/main.go                  # Entry point
├── pkg/cmd/                        # Command implementations
│   ├── root/root.go                # Root command, registers all subcommands
│   ├── factory.go                  # DI: IOStreams, HttpClient, Config
│   ├── route/                      # a7 route list|get|create|update|delete -g <group>
│   ├── upstream/                   # ⚠️ NOT EXPOSED in API7 EE — upstreams are inline-only (defined within services/routes)
│   ├── service/                    # a7 service ... -g <group>
│   ├── gateway-group/              # a7 gateway-group list|get|create|update|delete
│   ├── service-template/           # a7 service-template list|get|create|update|delete
│   ├── consumer/                   # a7 consumer ... -g <group>
│   ├── ssl/                        # a7 ssl ... -g <group>
│   ├── plugin/                     # a7 plugin list|get
│   ├── config/                     # a7 config sync|diff|dump|validate -g <group>
│   └── context/                    # a7 context create|use|list|delete|current
├── pkg/api/                        # API client + types
│   ├── client.go                   # HTTP wrapper with token auth
│   └── types_*.go                  # Go structs per resource (Route, Upstream, etc.)
├── pkg/iostreams/                  # I/O abstraction (TTY detection)
├── pkg/cmdutil/                    # Shared utilities (errors, exporter, flags)
├── pkg/tableprinter/               # Table rendering
├── pkg/httpmock/                   # HTTP mock for unit tests
├── internal/config/                # Context/config file management (~/.config/a7/)
├── test/fixtures/                  # JSON fixtures for unit tests
├── test/e2e/                       # E2E tests (build tag: e2e)
├── skills/                         # AI agent skill files
└── docs/                           # Project documentation
```

## Architecture Patterns

### Factory Pattern (Dependency Injection)

Every command receives a `*cmd.Factory` containing `IOStreams`, `HttpClient()`,
and `Config()`. No global state.

```go
type Factory struct {
    IOStreams   *iostreams.IOStreams
    HttpClient func() (*http.Client, error)
    Config     func() (config.Config, error)
}
```

The `Config()` interface provides:
- `Token()`: Returns the auth token (prefixed with `a7ee`)
- `GatewayGroup()`: Returns the default gateway group
- `TLSSkipVerify()`: TLS verification setting
- `CACert()`: Custom CA certificate path

### Command Pattern (Options + NewCmd + Run)

Every command follows the same structure:

```go
type Options struct {
    IO     *iostreams.IOStreams
    Client func() (*http.Client, error)
    Config func() (config.Config, error)
    GatewayGroup string // Required for runtime resources
    // command-specific fields
}

func NewCmdXxx(f *cmd.Factory) *cobra.Command { ... }
func xxxRun(opts *Options) error { ... }
```

### Authentication

API7 EE uses the `X-API-KEY` header. Tokens are prefixed with `a7ee`.

### PATCH Method

For `update` actions that use `PATCH`, a7 implements JSON Patch (RFC 6902) support.

## Resource Types Covered

| Resource | Key Field | API Path (Prefix) |
|----------|-----------|-------------------|
| Gateway Group | `id` | `/api/gateway_groups` |
| Service Template | `id` | `/api/services/template` |
| Route | `id` | `/apisix/admin/routes` |
| Service | `id` | `/apisix/admin/services` |
| Upstream | `id` | ⚠️ NOT EXPOSED — upstreams are inline-only in API7 EE (defined within services/routes) |
| Consumer | `username` | `/apisix/admin/consumers` |
| SSL | `id` | `/apisix/admin/ssl` |
| Global Rule | `id` | `/apisix/admin/global_rules` |
| Stream Route | `id` | `/apisix/admin/stream_routes` |
| Proto | `id` | `/apisix/admin/protos` |
| Secret | `id` | `/apisix/admin/secrets/{manager}/{id}` |
| Plugin Metadata | `plugin_name` | `/apisix/admin/plugin_metadata/{name}` |
| Plugin (read-only) | `name` | `/apisix/admin/plugins` |
| Credential | `id` | `/apisix/admin/consumers/{username}/credentials` |

Note: Runtime resources (routes, services, upstreams, etc.) are always scoped by the gateway group in the request URL or via headers.

## Common Commands

```bash
make build          # Build to ./bin/a7
make test           # Unit tests
make test-e2e       # E2E tests (requires API7 EE instance)
make lint           # golangci-lint
make fmt            # gofmt
make docker-up      # Start local stack
```

## Config Sync Workflow

The declarative config system (`a7 config sync/diff/dump/validate`) is scoped by gateway group:

```bash
a7 config sync -f config.yaml --gateway-group default
```

```yaml
version: "1"
routes:
  - id: my-route
    name: my-route
    paths:
      - /api/*
    service_id: my-service
services:
  - id: my-service
    name: my-service
    upstream:
      type: roundrobin
      nodes:
        - host: httpbin
          port: 8080
          weight: 1
```
