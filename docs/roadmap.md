# Development Roadmap

This document defines the per-PR development plan for the a7 CLI (API7 Enterprise Edition). Each PR is self-contained and ships implementation code, e2e tests against a real API7 EE instance, and user-facing documentation updates.

> **Status: 🚧 Phase 5 (Documentation) IN PROGRESS**

---

## Phase 5 — Documentation (Current)

**Goal**: Establish comprehensive documentation for the a7 project, adapting a6 references for API7 EE specific concepts (dual-API, gateway groups, JSON Patch).

### PR-28: Core Documentation Infrastructure

| File | Purpose |
|------|---------|
| `docs/adr/001-tech-stack.md` | Tech stack decisions, dual-API architecture |
| `docs/coding-standards.md` | Go style, naming (Tokens, Gateway Groups) |
| `docs/golden-example.md` | Reference implementation for action commands |
| `docs/testing-strategy.md` | Test patterns, E2E setup for EE |
| `docs/skills.md` | AI agent skill taxonomy and roadmap |
| `docs/documentation-maintenance.md` | Rules for keeping docs in sync |
| `docs/roadmap.md` | THIS file |
| `docs/api7ee-api-spec.md` | API7 EE Admin API reference |

---

## Phase 6 — AI Agent Skills

**Goal**: Create 40 `SKILL.md` files enabling AI agents to operate API7 EE.

### PR-29: AI Gateway & Enterprise Skills
- **Skills**: `a7-plugin-ai-proxy`, `a7-plugin-ai-prompt-template`, `a7-plugin-ai-rag`, `a7-recipe-gateway-group`, `a7-recipe-service-template`
- **Focus**: Configuring LLM providers and enterprise-only design-time resources.

### PR-30: Security & Auth Skills
- **Skills**: `a7-plugin-key-auth`, `a7-plugin-openid-connect`, `a7-plugin-wolf-rbac`, `a7-recipe-rbac-setup`
- **Focus**: Enterprise authentication and role-based access control.

---

## Phase 7 — Debug & Operations

**Goal**: Implement operational tools for troubleshooting API7 EE environments.

### PR-31: Debug Logs & Trace
- **Commands**: `a7 debug logs`, `a7 debug trace <route-id>`
- **Function**: Stream logs from EE control-plane/gateway; trace request execution path with EE-specific headers.

### PR-32: Resource Update (JSON Patch)
- **Commands**: `a7 <resource> update <id> --patch`
- **Function**: Implement precise updates using JSON Patch (RFC 6902) as required by API7 EE.

---

## Phase 8 — E2E Tests

**Goal**: Establish a robust integration test suite against a real API7 EE instance.

### PR-33: E2E Framework & Smoke Tests
- **Infrastructure**: `docker-compose.yml` with API7 EE + etcd + httpbin.
- **Scenarios**: `TestSmoke_Connectivity`, `TestAuth_TokenValidation`.

### PR-34: Resource Lifecycle Tests
- **Scenarios**: CRUD tests for all 16 core resources, verifying gateway group scoping and dual-API prefix handling.

---

## Phase 9 — Enterprise Features

**Goal**: Finalize CLI support for advanced API7 EE modules.

### PR-35: Token & RBAC Management
- **Commands**: `a7 token create|list`, `a7 user/role/policy`
- **Focus**: Managing CLI access tokens and platform permissions.

### PR-36: Portal & Custom Plugins
- **Commands**: `a7 portal`, `a7 custom-plugin`
- **Focus**: Developer portal configuration and custom Lua plugin management.

---

## Summary Table (Remaining Phases)

| Phase | PR | Scope | Validation |
|-------|----|-------|------------|
| 5 | PR-28 | Documentation Infrastructure | Doc linting |
| 6 | PR-29 | AI Gateway Skills | Skill validation |
| 6 | PR-30 | Security & Auth Skills | Skill validation |
| 7 | PR-31 | Debug Tooling | E2E output validation |
| 7 | PR-32 | JSON Patch Support | API integration tests |
| 8 | PR-33 | E2E Test Infra | Smoke tests |
| 8 | PR-34 | Resource CRUD E2E | Lifecycle validation |
| 9 | PR-35 | Token & RBAC | Multi-user E2E |
| 9 | PR-36 | Portal & Extensions | Feature-specific E2E |

**Total**: 9 PRs to reach production readiness.