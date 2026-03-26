# Product Requirements Document (PRD): a7 CLI

## Overview
- **Project Name**: a7 (repository: api7/a7)
- **Purpose**: A command-line tool that wraps the API7 Enterprise Edition Admin API, providing a convenient terminal interface for managing API7 EE resources including both control-plane and data-plane operations.
- **Target Users**: DevOps engineers, API developers, platform teams, and SREs responsible for managing API7 Enterprise gateways.
- **Design Philosophy**: AI-first development. Modeled after the [a6 CLI](https://github.com/moonming/a6) (for Apache APISIX), extended for Enterprise-specific capabilities (gateway groups, RBAC, service templates, developer portal, etc.).
- **Reference Implementation**: [a6 CLI](https://github.com/moonming/a6) — all features in a6 should have equivalents in a7, adapted for API7 EE's dual-API architecture.

## Problem Statement
- API7 Enterprise Edition has a comprehensive REST API but lacks an imperative CLI for ad-hoc operations.
- The existing ADC tool (api7/adc) provides declarative sync/diff/dump but is TypeScript-based and does not support imperative CRUD operations.
- The a6 CLI only targets open-source Apache APISIX Admin API, not the API7 EE control-plane API.
- Users are forced to use `curl`, the Dashboard UI, or ADC for all interactions—there is no quick imperative CLI for terminal workflows.

## Goals and Non-Goals

### Goals
- Provide full CRUD operations for all API7 EE resources (both control-plane `/api/*` and APISIX admin `/apisix/admin/*` endpoints). Note: 3 APISIX resources (standalone upstream, consumer group, plugin config) are not exposed in API7 EE.
- Support **multiple authentication modes**: access tokens (X-API-KEY with `a7ee` prefix) and session-based login.
- Implement **gateway group** scoping for all runtime operations (enterprise-specific concept).
- Implement context/profile management for switching between multiple API7 EE instances.
- Support rich terminal output (tables for TTY, JSON/YAML for pipes).
- Support **file-based resource creation and update** (`-f/--file`) for all resource types.
- Support **resource export** to YAML/JSON with label filtering for all applicable resources.
- Provide **declarative configuration** operations (sync, dump, diff, validate) absorbing ADC-like functionality.
- Provide **debug commands** for real-time log streaming and request tracing.
- Provide shell completions (bash, zsh, fish, PowerShell).
- Provide **AI agent skills** (SKILL.md files) for AI coding agents to work effectively with API7 EE.
- Provide comprehensive **documentation**: ADRs, coding standards, golden example, testing strategy, user guides.
- Provide **end-to-end tests** with Docker-based API7 EE test environment.
- Maintain an extensible command architecture following a6 patterns (Factory + IOStreams + API Client).

### Non-Goals
- Not replacing the API7 Dashboard (Web UI).
- Not managing gateway instance lifecycle (start/stop/restart).
- Not building a GUI.
- Not supporting APISIX open-source directly (use a6 for that).
- Not supporting API7 EE versions prior to v3.x.

## Target API7 EE Version
- **Supported Version**: API7 Enterprise 3.x (v3.5+)
- **API Prefixes**:
  - Control-plane: `/api/*` (service templates, gateway groups, users, tokens, RBAC, portal, etc.)
  - APISIX admin: `/apisix/admin/*` (published services, routes, upstreams, consumers, SSL, etc.)
- **Response Format**:
  - Single resource: `{"value":{...}}` or raw object
  - List: `{"total":N,"list":[...]}`
  - PATCH uses JSON Patch (RFC 6902)

## Authentication

### Auth Modes (in precedence order)
1. **Flag**: `--token` / `--api-key`
2. **Environment Variable**: `A7_TOKEN`
3. **Profile/Context config**: stored in `~/.config/a7/config.yaml`

### Token Types Supported
- **Access Token** (prefix `a7ee`): Full control-plane + APISIX admin API access, user-scoped.
- **Session Login**: `a7 auth login --username <user> --password <pass>` for cookie-based session auth.

### Security
- Prevent sensitive data from appearing in shell history by supporting key input from files or stdin.

## Command Design

### Command Structure
Noun-verb pattern (consistent with a6):
```
a7 <resource> <action> [args] [flags]
```

### Global Flags
- `--context` / `-c`: Override active context/profile.
- `--server` / `-s`: Override target server URL.
- `--token`: Override API token (X-API-KEY).
- `--gateway-group` / `-g`: Specify gateway group (required for runtime operations).
- `--output` / `-o`: Output format: `table` (default in TTY), `json`, `yaml`.
- `--format`: Go template for custom output.
- `--verbose` / `-v`: Enable verbose HTTP request logging.
- `--force`: Skip confirmation prompts.
- `-f` / `--file`: Read resource from JSON/YAML file.
- `--tls-skip-verify`: Skip TLS certificate verification.
- `--ca-cert`: Path to CA certificate file.

### Resource-Specific Flags
- **list**: `--page`, `--page-size`, `--search`, `--label`, `--gateway-group`
- **create/update**: `-f/--file` (JSON/YAML), `--dry-run`
- **delete**: `--force` to skip confirmation
- **export**: `--label`, `--gateway-group`, `--output` (yaml/json)

### Resource Commands — Control Plane (`/api/*`)

#### Service Templates (design-time)
- `a7 service-template list|get|create|update|patch|delete`
- `a7 service-template publish --gateway-group <id>` (publish to gateway group)

#### Gateway Groups
- `a7 gateway-group list|get|create|update|delete`

#### Tokens (access tokens)
- `a7 token list|get|create|delete|regenerate`

#### Users & RBAC
- `a7 user list|get|create|update|delete`
- `a7 role list|get|create|update|delete`
- `a7 permission-policy list|get|create|update|delete`

#### Developer Portal
- `a7 portal application list|get|create|update|delete`
- `a7 portal developer list|get|create|delete`

#### Custom Plugins
- `a7 custom-plugin list|get|create|update|delete`

#### System
- `a7 system settings [get|update]`
- `a7 audit-log list`

### Resource Commands — Published / Runtime (`/apisix/admin/*`)

All runtime commands require `--gateway-group <id>` (or default from context).

#### Routes
- `a7 route list|get|create|update|delete|export --gateway-group <id>`

#### Services (published)
- `a7 service list|get|create|update|delete|export --gateway-group <id>`

#### Upstreams
> **⚠️ NOT SUPPORTED**: Standalone upstreams are not exposed via the API7 EE Admin API. Upstreams exist only as inline objects within services and routes. The `a7 upstream` commands exist for APISIX compatibility but will not work against API7 EE.

#### Consumers
- `a7 consumer list|get|create|update|delete|export --gateway-group <id>`

#### Consumer Credentials
- `a7 credential list|get|create|update|delete --consumer <username> --gateway-group <id>`

#### Consumer Groups
> **⚠️ NOT SUPPORTED**: Consumer groups are not exposed via the API7 EE Admin API. The `a7 consumer-group` commands exist for APISIX compatibility but will not work against API7 EE.

#### SSL Certificates
- `a7 ssl list|get|create|update|delete|export --gateway-group <id>`

#### Plugins
- `a7 plugin list|get --gateway-group <id>` (read-only: list available plugins, get schema)

#### Global Rules
- `a7 global-rule list|get|create|update|delete|export --gateway-group <id>`

#### Stream Routes
- `a7 stream-route list|get|create|update|delete|export --gateway-group <id>`

#### Plugin Config
> **⚠️ NOT SUPPORTED**: Plugin configs are not exposed via the API7 EE Admin API. The `a7 plugin-config` commands exist for APISIX compatibility but will not work against API7 EE.

#### Plugin Metadata
- `a7 plugin-metadata get|create|update|delete --gateway-group <id>` (no list — keyed by plugin name)

#### Secrets
- `a7 secret list|get|create|update|delete --gateway-group <id>`

#### Proto
- `a7 proto list|get|create|update|delete|export --gateway-group <id>`

### Utility Commands
- `a7 context create|use|list|delete|current` — Manage contexts for multiple API7 EE instances.
- `a7 auth login|logout` — Session-based authentication.
- `a7 config sync|dump|diff|validate` — Declarative configuration operations.
- `a7 debug logs` — Stream real-time API7 EE error logs.
- `a7 debug trace` — Trace request flow through the gateway.
- `a7 completion bash|zsh|fish|powershell` — Shell completion scripts.
- `a7 version` — Display version information.
- `a7 update` — Self-update binary.

## Implementation Phases

### Phase 1 — Core CLI Scaffold ✅ COMPLETE
1. ✅ Project structure, Makefile, CI workflow, GoReleaser config.
2. ✅ Factory DI pattern (IOStreams, HttpClient, Config).
3. ✅ Config/context management (create, use, list, delete, current).
4. ✅ Authentication: token-based auth (X-API-KEY header).
5. ✅ API client with auth transport.
6. ✅ HTTP mock framework for unit tests.
7. ✅ JSON/YAML/table output modes.
8. ✅ Shell completions (bash, zsh, fish, powershell).
9. ✅ `a7 version`.

### Phase 2 — Resource CRUD ✅ COMPLETE
1. ✅ Gateway Group CRUD (list, get, create, update, delete).
2. ✅ Service Template CRUD + publish.
3. ✅ Route CRUD.
4. ✅ Upstream CRUD. ⚠️ Not functional in API7 EE (standalone upstreams not exposed).
5. ✅ Service (runtime) CRUD.
6. ✅ Consumer CRUD.
7. ✅ SSL CRUD.
8. ✅ Plugin list + get.
9. ✅ Global Rule CRUD.
10. ✅ Stream Route CRUD.
11. ✅ Plugin Config CRUD. ⚠️ Not functional in API7 EE (plugin_config not exposed).
12. ✅ Plugin Metadata CRUD (no list).
13. ✅ Consumer Group CRUD. ⚠️ Not functional in API7 EE (consumer_group not exposed).
14. ✅ Credential CRUD (nested under consumer).
15. ✅ Secret CRUD.
16. ✅ Proto CRUD.
17. ✅ Unit tests for all new commands (56 test files).

### Phase 3 — CLI Usability ✅ COMPLETE
1. ✅ `-f/--file` flag: file-based create/update for all resource commands.
2. ✅ `export` subcommand for all applicable resources (route, service, upstream, consumer, consumer-group, ssl, global-rule, stream-route, plugin-config, proto).
3. ✅ `--force` flag for delete commands (skip confirmation).
4. ✅ `--label` flag for list/export commands (label-based filtering).
5. 🔲 `--dry-run` flag for create/update commands.
6. 🔲 `--verbose` flag for HTTP request/response logging.

### Phase 4 — Declarative Configuration ✅ COMPLETE
1. ✅ `a7 config dump` — Export full gateway configuration to YAML file.
2. ✅ `a7 config validate` — Validate a configuration file against API7 EE schema.
3. ✅ `a7 config diff` — Compare local config file against live gateway state.
4. ✅ `a7 config sync` — Apply a configuration file to the gateway (with `--dry-run` support).
5. ✅ `configutil` package — Shared helpers for config file parsing, resource ordering, and diff logic.

### Phase 5 — Documentation ✅ COMPLETE
1. ✅ `docs/adr/001-tech-stack.md` — Architecture Decision Record.
2. ✅ `docs/coding-standards.md` — Go coding standards and conventions.
3. ✅ `docs/golden-example.md` — Canonical implementation example (Factory, IOStreams, tests).
4. ✅ `docs/testing-strategy.md` — Unit test and E2E test patterns.
5. ✅ `docs/skills.md` — AI agent skill format specification.
6. ✅ `docs/documentation-maintenance.md` — Doc update rules and templates.
7. ✅ `docs/roadmap.md` — Per-PR development plan for Phases 5-9.
8. ✅ `docs/api7ee-api-spec.md` — API7 EE Admin API reference (16 resources, dual-API).
9. ✅ `docs/user-guide/` — 21 per-resource user guides (getting-started, configuration, route, service, upstream, consumer, ssl, plugin, global-rule, stream-route, plugin-config, plugin-metadata, consumer-group, credential, secret, proto, declarative-config, gateway-group, service-template, debug, bulk-operations).

### Phase 6 — AI Agent Skills ✅ COMPLETE
Port and adapt 40 SKILL.md files from a6, organized by category:

| Category | Count | Skills |
|----------|-------|--------|
| **Shared** | 1 | Core a7 conventions and patterns |
| **Authentication** | 5 | key-auth, jwt-auth, basic-auth, hmac-auth, openid-connect |
| **Security & Rate Limiting** | 4 | ip-restriction, cors, limit-count, limit-req |
| **Traffic & Transformation** | 5 | proxy-rewrite, response-rewrite, traffic-split, redirect, grpc-transcode |
| **AI Gateway** | 4 | ai-proxy, ai-prompt-template, ai-prompt-decorator, ai-content-moderation |
| **Observability** | 6 | prometheus, skywalking, zipkin, http-logger, kafka-logger, datadog |
| **Advanced Plugins** | 5 | serverless, ext-plugin, fault-injection, consumer-restriction, wolf-rbac |
| **Operational Recipes** | 5 | blue-green, canary, circuit-breaker, health-check, mTLS |
| **Advanced Recipes** | 3 | multi-tenant, api-versioning, graphql-proxy |
| **Personas** | 2 | operator, developer |

### Phase 7 — Debug & Operations ✅ COMPLETE
1. ✅ `a7 debug logs` — Stream real-time error logs from API7 EE.
2. ✅ `a7 debug trace` — Trace a request through the gateway with timing breakdown.
3. ✅ `a7 update` — Self-update CLI binary from GitHub releases.

### Phase 8 — End-to-End Tests ✅ COMPLETE
1. ✅ `test/e2e/docker-compose.yml` — Docker Compose for API7 EE (Dashboard + DP Manager + Gateway + PostgreSQL).
2. ✅ `test/e2e/setup_test.go` — TestMain, binary build, admin/control API helpers, shared test utilities.
3. ✅ Per-resource E2E tests (22 test files): route, service, upstream, consumer, ssl, plugin, global-rule, stream-route, plugin-config, plugin-metadata, consumer-group, credential, secret, proto, context, gateway-group, service-template.
4. ✅ Declarative config E2E tests: dump, diff, sync, validate (config_test.go + config_sync_test.go).
5. ✅ Export and label E2E tests (integrated into resource test files).
6. ✅ Debug E2E tests: trace (JSON/method/headers/host/path) + logs (file mode).
7. ✅ Completion + version + help E2E tests.
8. ✅ `.github/workflows/e2e.yml` — CI workflow for automated E2E testing.
9. ✅ Makefile targets: `docker-up`, `docker-down`, `test-e2e`.
10. ✅ All 40 skill examples covered in resource-specific tests.

### Phase 9 — Enterprise-Specific Features 🔲 FUTURE
1. 🔲 Token management (create/list/delete/regenerate).
2. 🔲 RBAC management: user, role, permission-policy.
3. 🔲 Developer Portal: application, developer, subscription.
4. 🔲 Custom plugin management.
5. 🔲 Audit log querying.
6. 🔲 `a7 auth login/logout` — Session-based authentication.
7. 🔲 Interactive mode (fuzzy selection).
8. 🔲 Extension/plugin system.
9. 🔲 Bulk operations.

## UX Requirements
- **TTY Detection**: Default to tables in interactive terminals and JSON when piped.
- **Colors**: ANSI colors when supported. Respect `NO_COLOR` environment variable.
- **Confirmation Prompts**: Destructive operations require confirmation unless `--force`.
- **Error Messages**: HTTP status + API7 error message + suggestion.
- **Progress Feedback**: Spinners for long operations.
- **Consistent Sorting**: Resources listed by ID ascending by default.
- **Gateway Group Awareness**: Commands that require a gateway group should error clearly if none specified.

## Configuration Storage
- **Location**: `~/.config/a7/config.yaml`. Respect `XDG_CONFIG_HOME`.
- **Format**:
  ```yaml
  current-context: local
  contexts:
    - name: local
      server: https://localhost:7443
      token: a7ee.xxxxxxxxxx
      gateway-group: default
      tls-skip-verify: true
    - name: production
      server: https://api7.example.com
      token: a7ee.yyyyyyyyyy
      gateway-group: production
      ca-cert: /path/to/ca.pem
  ```

## Technical Decisions
- **Language**: Go 1.22+
- **CLI Framework**: spf13/cobra
- **HTTP Client**: net/http with custom auth transport
- **Output**: Custom tableprinter (tabwriter-based) + JSON/YAML exporters
- **Testing**: Unit tests with httpmock, E2E tests with real API7 EE instance
- **Release**: GoReleaser for cross-platform builds
- **License**: Apache 2.0

## Feature Parity with a6

The following table tracks feature parity between a7 and [a6](https://github.com/moonming/a6):

| Feature | a6 | a7 | Notes |
|---------|----|----|-------|
| Resource CRUD (14 types) | ✅ | ✅ (13 functional in API7 EE) | a7 adds gateway-group, service-template; 3 APISIX resources (upstream, consumer-group, plugin-config) not exposed in API7 EE |
| Context management | ✅ | ✅ | |
| Shell completions | ✅ | ✅ | |
| JSON/YAML/table output | ✅ | ✅ | |
| `-f/--file` create/update | ✅ | ✅ | Phase 3 |
| `export` subcommand | ✅ | ✅ | Phase 3 |
| `--force` delete flag | ✅ | ✅ | Phase 3 |
| `--label` filtering | ✅ | ✅ | Phase 3 |
| `--verbose` HTTP logging | ✅ | 🔲 | Phase 3 |
| Declarative config (dump/diff/sync/validate) | ✅ | ✅ | Phase 4 |
| docs/ (29 files) | ✅ | ✅ | Phase 5 |
| skills/ (40 SKILL.md) | ✅ | ✅ | Phase 6 |
| Debug (logs + trace) | ✅ | ✅ | Phase 7 |
| Self-update | ✅ | ✅ | Phase 7 |
| E2E tests | ✅ | ✅ | Phase 8 |
| Bulk operations | ✅ | 🔲 | Phase 9 |
| Interactive mode | ✅ | 🔲 | Phase 9 |
| Extension system | ✅ | 🔲 | Phase 9 |
