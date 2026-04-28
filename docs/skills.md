# AI Agent Skills

This document describes the skill system for the a7 CLI. Skills are structured knowledge files that enable AI coding agents to work effectively with API7 Enterprise Edition through the a7 CLI.

## Overview

Skills are `SKILL.md` files stored in the `skills/` directory. Each skill provides domain-specific instructions, command patterns, and decision guidance for AI agents. The format is compatible with 39+ AI coding agents including Claude Code, OpenCode, Cursor, GitHub Copilot, and Windsurf.

## Directory Structure

```
skills/
├── a7-shared/SKILL.md               # Core a7 conventions (shared skill)
├── a7-plugin-ai-proxy/SKILL.md      # AI Gateway plugin skill
├── a7-plugin-key-auth/SKILL.md      # key-auth plugin skill
├── a7-recipe-canary/SKILL.md        # Canary release recipe
├── a7-persona-operator/SKILL.md     # Operator persona
└── ...
```

Each skill lives in its own directory: `skills/<skill-name>/SKILL.md`.

## Skill Taxonomy

Skills follow a naming convention with four types:

| Prefix | Type | Description | Example |
|--------|------|-------------|---------|
| `a7-shared` | Shared | Core project conventions and patterns | `a7-shared` |
| `a7-plugin-*` | Plugin | One API7 EE plugin — config, examples, gateway group scoping | `a7-plugin-ai-proxy` |
| `a7-recipe-*` | Recipe | Multi-step operational task | `a7-recipe-canary` |
| `a7-persona-*` | Persona | Role-specific workflow guidance | `a7-persona-operator` |

### Naming Rules

- **Format**: kebab-case
- **Pattern**: `^[a-z0-9]+(-[a-z0-9]+)*$`
- **Directory name must match the `name` field in frontmatter**

## SKILL.md Format

Every skill file has two parts: YAML frontmatter and Markdown body.

### Frontmatter (Required)

```yaml
---
name: a7-plugin-ai-proxy
description: >-
  Skill for configuring AI Proxy plugin on API7 EE routes and service templates.
  Covers LLM provider configuration, model selection, and endpoint routing.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: ai-gateway
  apisix_version: ">=3.0.0"
  plugin_name: ai-proxy
  a7_commands:
    - a7 route create
    - a7 service-template create
    - a7 plugin list
---
```

**Required fields:**

| Field | Description |
|-------|-------------|
| `name` | Skill identifier. Must match directory name. Kebab-case. |
| `description` | Multi-line description of what this skill covers. |

**Recommended fields:**

| Field | Description |
|-------|-------------|
| `version` | Semantic version of the skill content. |
| `author` | Who authored the skill. |
| `license` | License identifier (e.g., `Apache-2.0`). |
| `metadata` | Structured metadata for categorization and filtering. |

### Body (Markdown)

The body content depends on the skill type:

**Plugin skills** (EE specific):
- Plugin description and AI Gateway context
- Configuration schema reference
- **Gateway Group Scoping**: How to enable per gateway group
- Example: Enabling on a Route
- Example: Enabling on a Service Template
- Enterprise-only features and limitations

**Recipe skills** (EE specific):
- Enterprise workflow goal (e.g., "Create a cross-group service template")
- Prerequisites (e.g., "Existing gateway groups")
- Step-by-step instructions with `a7` commands
- Verification using `a7` list/get commands
- Rollback procedure

**Persona skills**:
- Role description (Platform Engineer, API Architect, App Developer)
- Common enterprise workflows
- Decision trees for resource selection (e.g., "Route vs Service Template")
- Which other skills to load for specific tasks

## CI Validation

Every PR validates `skills/` with `scripts/validate-skills.sh`. The script checks:

1. Every `skills/*/SKILL.md` has valid YAML frontmatter
2. Required fields `name` and `description` are present
3. `name` matches the directory name
4. `name` follows kebab-case pattern
5. `description` is non-empty
6. skill names are unique

The E2E suite also contains static skill checks under `test/e2e/skills`.
Those checks keep this document aligned with the actual `skills/` inventory and
prevent references to known removed commands such as the old health and portal
commands.

Run locally:

```bash
make validate-skills
```

## Adding a New Skill

1. Choose the skill type and name following the [taxonomy](#skill-taxonomy)
2. Create the directory: `mkdir skills/<skill-name>`
3. Create `skills/<skill-name>/SKILL.md` with frontmatter and body
4. Run validation: `make validate-skills`
5. Update this document if adding a new skill type or category

## Current Inventory

The repository currently contains 40 skills:

**Shared**

- `a7-shared`

**Personas**

- `a7-persona-developer`
- `a7-persona-operator`

**Plugin Skills**

- `a7-plugin-ai-content-moderation`
- `a7-plugin-ai-prompt-decorator`
- `a7-plugin-ai-prompt-template`
- `a7-plugin-ai-proxy`
- `a7-plugin-basic-auth`
- `a7-plugin-consumer-restriction`
- `a7-plugin-cors`
- `a7-plugin-datadog`
- `a7-plugin-ext-plugin`
- `a7-plugin-fault-injection`
- `a7-plugin-grpc-transcode`
- `a7-plugin-hmac-auth`
- `a7-plugin-http-logger`
- `a7-plugin-ip-restriction`
- `a7-plugin-jwt-auth`
- `a7-plugin-kafka-logger`
- `a7-plugin-key-auth`
- `a7-plugin-limit-count`
- `a7-plugin-limit-req`
- `a7-plugin-openid-connect`
- `a7-plugin-prometheus`
- `a7-plugin-proxy-rewrite`
- `a7-plugin-redirect`
- `a7-plugin-response-rewrite`
- `a7-plugin-serverless`
- `a7-plugin-skywalking`
- `a7-plugin-traffic-split`
- `a7-plugin-wolf-rbac`
- `a7-plugin-zipkin`

**Recipe Skills**

- `a7-recipe-api-versioning`
- `a7-recipe-blue-green`
- `a7-recipe-canary`
- `a7-recipe-circuit-breaker`
- `a7-recipe-graphql-proxy`
- `a7-recipe-health-check`
- `a7-recipe-mtls`
- `a7-recipe-multi-tenant`

## Current Compatibility Notes

- Route examples should use the current API7 EE model: create a service, then create routes with `service_id`.
- Auth examples should use `consumer create` plus `credential create`; do not put auth plugin credentials directly in the consumer body.
- Standalone upstream workflows are not the preferred `a7` path for current API7 EE. Use service inline upstreams and service-backed routes unless you are intentionally documenting APISIX-compatible behavior.
- Gateway/httpbin traffic checks are optional for `a7`; the default CI focuses on CLI-driven control-plane resource CRUD and structured `get/list/dump` assertions.
