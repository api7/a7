# Product Requirements Document (PRD): a7 CLI

## Overview
- **Project Name**: a7 (repository: api7/a7)
- **Purpose**: A command-line tool that wraps the API7 Enterprise Edition Admin API, providing a convenient terminal interface for managing API7 EE resources including both control-plane and data-plane operations.
- **Target Users**: DevOps engineers, API developers, platform teams, and SREs responsible for managing API7 Enterprise gateways.
- **Design Philosophy**: AI-first development. Modeled after the a6 CLI (for Apache APISIX), extended for Enterprise-specific capabilities (gateway groups, RBAC, service templates, developer portal, etc.).

## Problem Statement
- API7 Enterprise Edition has a comprehensive REST API but lacks an imperative CLI for ad-hoc operations.
- The existing ADC tool (api7/adc) provides declarative sync/diff/dump but is TypeScript-based and does not support imperative CRUD operations.
- The a6 CLI only targets open-source Apache APISIX Admin API, not the API7 EE control-plane API.
- Users are forced to use `curl`, the Dashboard UI, or ADC for all interactions—there is no quick imperative CLI for terminal workflows.

## Goals and Non-Goals

### Goals
- Provide full CRUD operations for all API7 EE resources (both control-plane `/api/*` and APISIX admin `/apisix/admin/*` endpoints).
- Support **multiple authentication modes**: access tokens (X-API-KEY with `a7ee` prefix), gateway-group admin keys (`a7adm` prefix), and session-based login.
- Implement **gateway group** scoping for all operations (enterprise-specific concept).
- Implement context/profile management for switching between multiple API7 EE instances.
- Support rich terminal output (tables for TTY, JSON/YAML for pipes).
- Provide declarative config operations (sync, dump, diff, validate) absorbing ADC-like functionality.
- Provide shell completions (bash, zsh, fish, PowerShell).
- Maintain an extensible command architecture following a6 patterns (Factory + IOStreams + API Client).
- Reuse the API7 EE Go SDK (`sdk/` in api7ee-3-control-plane) where possible.

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

## Authentication

### Auth Modes (in precedence order)
1. **Flag**: `--token` / `--api-key`
2. **Environment Variable**: `A7_TOKEN`
3. **Profile/Context config**: stored in `~/.config/a7/config.yaml`

### Token Types Supported
- **Access Token** (prefix `a7ee`): Full control-plane + APISIX admin API access, user-scoped.
- **Gateway Admin Key** (prefix `a7adm`): Restricted to `/apisix/admin/*` endpoints, gateway-group-scoped.
- **Session Login**: `a7 auth login --username <user> --password <pass>` for cookie-based session auth.

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
- `--gateway-group` / `-g`: Specify gateway group (required for many operations).
- `--output` / `-o`: Output format: `table` (default in TTY), `json`, `yaml`.
- `--format`: Go template for custom output.
- `--verbose` / `-v`: Enable verbose HTTP request logging.
- `--force`: Skip confirmation prompts.
- `-f` / `--file`: Read resource from JSON/YAML file.
- `--tls-skip-verify`: Skip TLS certificate verification.
- `--ca-cert`: Path to CA certificate file.

### Resource Commands — Control Plane (`/api/*`)

#### Service Templates (design-time)
- `a7 service-template list|get|create|update|patch|delete`
- `a7 service-template publish --gateway-group <id>` (publish to gateway group)
- `a7 service-template conflict-check`

#### Routes (design-time, template routes)
- `a7 route-template list|get|create|update|patch|delete`

#### Stream Routes (design-time)
- `a7 stream-route-template list|get|create|update|patch|delete`

#### Gateway Groups
- `a7 gateway-group list|get|create|update|delete`
- `a7 gateway-group instances --gateway-group <id>` (list instances)
- `a7 gateway-group admin-key [create|get]` (manage admin keys)
- `a7 gateway-group install-script --type docker|helm|docker-compose`

#### Service Registries
- `a7 service-registry list|get|create|update|delete --gateway-group <id>`

#### Tokens (access tokens)
- `a7 token list|get|create|delete|regenerate`

#### Users & RBAC
- `a7 user list|get|create|update|delete`
- `a7 role list|get|create|update|delete`
- `a7 role attach-policy|detach-policy`
- `a7 permission-policy list|get|create|update|delete`

#### Developer Portal
- `a7 portal application list|get|create|update|delete`
- `a7 portal developer list|get|create|delete`
- `a7 portal subscription list|get|create|delete`
- `a7 portal token list|get|create|delete|regenerate`

#### Custom Plugins
- `a7 custom-plugin list|get|create|update|delete`

#### System
- `a7 system settings [get|update]`
- `a7 system info`
- `a7 audit-log list`
- `a7 alert-policy list|get|create|update|delete`

#### Labels
- `a7 label list --resource-type <type>`

### Resource Commands — Published / Runtime (`/apisix/admin/*`)

#### Published Services
- `a7 service list|get|create|update|patch|delete --gateway-group <id>`

#### Routes (published)
- `a7 route list|get|create|update|patch|delete --gateway-group <id>`

#### Stream Routes (published)
- `a7 stream-route list|get|create|update|patch|delete --gateway-group <id>`

#### Upstreams
- `a7 upstream list|get|create|update|patch|delete --service-id <id> --gateway-group <id>`

#### Consumers
- `a7 consumer list|get|create|update|delete --gateway-group <id>`

#### Consumer Credentials
- `a7 credential list|get|create|update|delete --consumer <username> --gateway-group <id>`

#### SSL Certificates
- `a7 ssl list|get|create|update|delete --gateway-group <id>`
- `a7 certificate list|get|create|update|delete --gateway-group <id>`
- `a7 ca-certificate list|get|create|update|delete --gateway-group <id>`
- `a7 sni list|get|create|update|delete --gateway-group <id>`

#### Plugins
- `a7 plugin list|get --gateway-group <id>` (read-only: list available plugins, get schema)

#### Global Rules
- `a7 global-rule list|get|create|update|delete --gateway-group <id>`

#### Plugin Metadata
- `a7 plugin-metadata list|get|create|update|delete --gateway-group <id>`

#### Secrets
- `a7 secret-provider list|get|create|update|delete --gateway-group <id>`

### Utility Commands
- `a7 context create|use|list|delete|current` — Manage contexts for multiple API7 EE instances.
- `a7 auth login|logout` — Session-based authentication.
- `a7 config sync|dump|diff|validate` — Declarative configuration operations.
- `a7 completion bash|zsh|fish|powershell` — Shell completion scripts.
- `a7 version` — Display version information.
- `a7 update` — Self-update binary.

### Resource-Specific Flags
- **list**: `--page`, `--page-size`, `--search`, `--label`, `--gateway-group`
- **create/update**: `-f/--file` (JSON/YAML), `--dry-run`
- **delete**: `--force` to skip confirmation
- **export**: `--label`, `--gateway-group`

## MVP Scope (Phase 1)
1. Context management (create, use, list, delete, current).
2. Authentication: token-based auth + `auth login/logout`.
3. Gateway Group CRUD + list instances.
4. Service Template CRUD + publish.
5. Route (published) CRUD.
6. Upstream CRUD.
7. Consumer CRUD.
8. SSL CRUD.
9. Plugin list + schema.
10. JSON/YAML/table output modes.
11. Shell completions.
12. `a7 version`.

## Phase 2
1. All remaining runtime resources: global-rule, stream-route, plugin-metadata, secret-provider, consumer credentials.
2. All remaining control-plane resources: custom-plugin, service-registry, labels.
3. Declarative config: sync, dump, diff, validate.
4. Token management (create/list/delete/regenerate).
5. Self-update mechanism.

## Phase 3
1. RBAC management: user, role, permission-policy.
2. Developer Portal: application, developer, subscription, portal token.
3. Audit log querying.
4. Alert policy management.
5. Debug session commands.
6. Interactive mode (fuzzy selection).
7. Extension/plugin system.
8. Bulk operations.

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
- **Config**: spf13/viper (for config file + env var merging)
- **HTTP Client**: net/http with custom transport (reuse patterns from a6, reference API7 EE SDK)
- **Output**: Custom tableprinter (tabwriter-based) + JSON/YAML exporters
- **Testing**: Unit tests with httpmock, E2E tests with real API7 EE instance
- **Release**: GoReleaser for cross-platform builds
- **License**: Apache 2.0
