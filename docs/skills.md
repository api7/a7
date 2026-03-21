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
├── a7-recipe-gateway-group/SKILL.md # Gateway group management recipe
├── a7-persona-platform-eng/SKILL.md # Platform Engineer persona
└── ...
```

Each skill lives in its own directory: `skills/<skill-name>/SKILL.md`.

## Skill Taxonomy

Skills follow a naming convention with four types:

| Prefix | Type | Description | Example |
|--------|------|-------------|---------|
| `a7-shared` | Shared | Core project conventions and patterns | `a7-shared` |
| `a7-plugin-*` | Plugin | One API7 EE plugin — config, examples, gateway group scoping | `a7-plugin-ai-proxy` |
| `a7-recipe-*` | Recipe | Multi-step operational task (e.g., setting up a service template) | `a7-recipe-service-template` |
| `a7-persona-*` | Persona | Role-specific workflow guidance | `a7-persona-platform-eng` |

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

Every PR that modifies `skills/` is validated by `scripts/validate-skills.sh`. The script checks:

1. Every `skills/*/SKILL.md` has valid YAML frontmatter
2. Required fields `name` and `description` are present
3. `name` matches the directory name
4. `name` follows kebab-case pattern
5. `description` is non-empty

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

## Skill Roadmap

| PR | Skills | Description |
|----|--------|-------------|
| PR-28 | 1 | Infrastructure + `a7-shared` |
| PR-29 | 6 | AI Gateway plugins (ai-proxy, ai-prompt-template, ai-prompt-decorator, ai-content-moderation, ai-rag, ai-token-limiter) |
| PR-30 | 5 | Enterprise core (gateway-group, service-template, rbac, portal, audit-log) |
| PR-31 | 5 | Authentication plugins (key-auth, jwt-auth, basic-auth, openid-connect, wolf-rbac) |
| PR-32 | 4 | Security + rate limiting (ip-restriction, cors, limit-count, limit-req) |
| PR-33 | 5 | Traffic + transformation (proxy-rewrite, response-rewrite, traffic-split, redirect, grpc-transcode) |
| PR-34 | 5 | Operational recipes (blue-green, canary, circuit-breaker, health-check, service-registry) |
| PR-35 | 6 | Observability (prometheus, skywalking, zipkin, http-logger, kafka-logger, datadog) |
| PR-36 | 3 | Advanced recipes + personas |

**Total**: 40 skills across 9 PRs.