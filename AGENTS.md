# AGENTS.md — a7 Development Guide
> Entry point for developers and AI coding agents working on the a7 CLI.
> Read this FIRST before making any changes.

### Project Overview
a7 is a Go CLI wrapping the API7 Enterprise Edition Admin API (control-plane + APISIX admin).
- Binary name: `a7`
- Module path: `github.com/api7/a7`
- Go version: 1.22+
- License: Apache 2.0
- Reference project: [a6 CLI](https://github.com/moonming/a6) (APISIX CLI)

### Document Map

| Document | Purpose | When to Read |
|---|---|---|
| `AGENTS.md` (this file) | Entry point, development guide | Always — read first |
| `PRD.md` | Product requirements, command design, scope | Before adding features |
| `docs/roadmap.md` | Per-PR development plan with dependencies | Before starting any PR |
| `docs/api7ee-api-spec.md` | API7 EE Admin API reference | When implementing API client |
| `docs/adr/001-tech-stack.md` | Architecture decisions, patterns | Before writing code |
| `docs/golden-example.md` | Complete reference implementation | When adding new commands |
| `docs/coding-standards.md` | Go style, naming conventions | Before writing code |
| `docs/testing-strategy.md` | Test patterns and practices | Before writing tests |

### Project Structure

```
a7/
├── .github/workflows/             # CI/CD
│   ├── ci.yml                     # Unit test + lint
│   └── e2e.yml                    # E2E tests with real API7 EE
├── cmd/a7/main.go                 # Entry point
├── pkg/cmd/                       # Command implementations
│   ├── root/root.go              # Root command registration
│   ├── factory.go                # Factory DI container
│   ├── route/                    # Published route commands
│   ├── upstream/                 # Published upstream commands
│   ├── service/                  # Published service commands
│   ├── consumer/                 # Consumer commands
│   ├── ssl/                      # SSL certificate commands
│   ├── plugin/                   # Plugin listing commands
│   ├── global-rule/              # Global rule commands
│   ├── stream-route/             # Stream route commands
│   ├── secret-provider/          # Secret provider commands
│   ├── plugin-metadata/          # Plugin metadata commands
│   ├── credential/               # Consumer credential commands
│   ├── service-template/         # Service template commands (EE)
│   ├── route-template/           # Route template commands (EE)
│   ├── gateway-group/            # Gateway group commands (EE)
│   ├── service-registry/         # Service registry commands (EE)
│   ├── token/                    # Access token commands (EE)
│   ├── user/                     # User management commands (EE)
│   ├── role/                     # Role & RBAC commands (EE)
│   ├── permission-policy/        # Permission policy commands (EE)
│   ├── custom-plugin/            # Custom plugin commands (EE)
│   ├── portal/                   # Developer portal commands (EE)
│   ├── audit-log/                # Audit log commands (EE)
│   ├── context/                  # Context/profile management
│   ├── auth/                     # Auth login/logout commands
│   ├── config/                   # Declarative config commands
│   ├── completion/               # Shell completion
│   └── version/                  # Version command
├── pkg/api/                       # API7 EE API client
│   ├── client.go                 # HTTP client (auth transport)
│   ├── types.go                  # Generic response types
│   ├── types_route.go            # Route types
│   ├── types_service.go          # Service types
│   ├── types_upstream.go         # Upstream types
│   ├── types_consumer.go         # Consumer types
│   ├── types_ssl.go              # SSL types
│   ├── types_gateway_group.go    # Gateway group types (EE)
│   ├── types_service_template.go # Service template types (EE)
│   ├── types_token.go            # Token types (EE)
│   └── ...                       # One types file per resource
├── pkg/iostreams/                 # I/O abstraction (TTY detection)
├── pkg/cmdutil/                   # Shared command utilities
│   ├── errors.go                 # Error formatting
│   └── exporter.go               # JSON/YAML export
├── pkg/tableprinter/              # Table output rendering
├── pkg/httpmock/                  # HTTP mocking for tests
├── internal/config/               # Configuration/context management
├── internal/version/              # Build version info
├── internal/update/               # Self-update mechanism
├── docs/                          # Documentation
├── test/fixtures/                 # JSON fixtures for tests
├── test/e2e/                      # E2E tests
├── skills/                        # AI agent skill files
├── scripts/                       # CI/utility scripts
├── Makefile                       # Build, test, lint commands
└── .goreleaser.yml                # Cross-platform release config
```

### Key Architecture Patterns
1. **Factory Pattern**: Every command receives a Factory with IOStreams, HttpClient, Config. No global state.
2. **Command Pattern**: Options struct + `NewCmd()` + `Run()` per command.
3. **Output Pattern**: Table for TTY, JSON for pipes. `--output` flag overrides.
4. **Testing Pattern**: `httpmock` stubs + test IOStreams. No real network in unit tests.
5. **Gateway Group Scoping**: Most resource operations require `--gateway-group`. The client passes this as a query param or path segment.
6. **Dual API Prefix**: Control-plane resources use `/api/*`, runtime resources use `/apisix/admin/*`. The client handles prefix selection transparently.

### API7 EE vs APISIX Differences (Key)
- API7 EE uses `/api/services/template` for design-time services, `/apisix/admin/services` for published (runtime) services.
- Gateway groups scope all operations. Many endpoints require `gateway_group_id` as a query parameter.
- Auth tokens have prefixes: `a7ee` (access token), `a7adm` (gateway admin key), `a7prt` (portal token).
- PATCH endpoints use JSON Patch (RFC 6902) arrays, not merge-patch.
- Enterprise-specific resources: gateway groups, service templates, RBAC (users/roles/policies), developer portal, audit logs, custom plugins, service registries, tokens.

### How to Add a New Command
1. Read `PRD.md` for the resource spec and `docs/api7ee-api-spec.md` for the API.
2. Create types in `pkg/api/types_<resource>.go`.
3. Create parent command: `pkg/cmd/<resource>/<resource>.go`.
4. Create action files: `pkg/cmd/<resource>/list/list.go`, `get/get.go`, etc.
5. Add tests: `*_test.go` alongside each action.
6. Add fixture: `test/fixtures/<resource>_list.json`.
7. Register in `pkg/cmd/root/root.go`.
8. Run `make check`.

### Common Commands
```bash
make build            # Build binary to ./bin/a7
make test             # Run unit tests
make test-verbose     # Tests with verbose output
make test-e2e         # E2E tests (requires API7 EE)
make lint             # golangci-lint
make fmt              # Format code
make check            # fmt + vet + lint + test
make clean            # Remove build artifacts
```

### Environment Variables
| Variable | Purpose | Default |
|---|---|---|
| `A7_TOKEN` | API access token | (from context config) |
| `A7_SERVER` | Control-plane server URL | (from context config) |
| `A7_GATEWAY_GROUP` | Default gateway group | (from context config) |
| `A7_CONFIG_DIR` | Config directory | `~/.config/a7` |
| `NO_COLOR` | Disable color output | (unset) |

### Commit Message Format
```
<type>(<scope>): <description>

Types: feat, fix, refactor, test, docs, chore
Scope: route, upstream, service, consumer, ssl, plugin, context, api, config, gateway-group, token, etc.
Example: feat(gateway-group): add gateway-group list command with table output
```

### Mandatory Rules
1. Every code change must have tests.
2. Every new command follows the golden example pattern.
3. Never suppress errors. Always handle and propagate.
4. Never use global state. Use Factory DI.
5. Run `make check` before committing.
