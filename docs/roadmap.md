# Development Roadmap

This document tracks phase status for the a7 CLI (API7 Enterprise Edition) and records what is **deferred (post-GA)**. For the authoritative scope statement see `PRD.md`. For per-resource validation against a live API7 EE see `docs/ga-test-plan.md` and `docs/ga-test-report.md`.

> **Status: GA Readiness Handoff.** Phases 1 through 8 are complete. Current effort is GA validation of the management plane against a real API7 EE instance (see issue #22). Phase 9 work is deferred post-GA.

---

## Completed Phases

| Phase | Scope | Status |
|-------|-------|--------|
| 1 | Core CLI scaffold (Factory DI, IOStreams, HTTP client, mock framework, completions, `a7 version`) | ✅ |
| 2 | Resource CRUD for all API7 EE supported resources (route, service, consumer, credential, ssl, plugin, plugin-metadata, global-rule, stream-route, secret, proto, gateway-group) | ✅ |
| 3 | CLI usability (`-f/--file`, `export`, `--force`, `--label`). `--dry-run` and `--verbose` are partial; see PRD Phase 3. | ✅ (partial flags noted in PRD) |
| 4 | Declarative configuration (`a7 config dump|diff|sync|validate`) | ✅ |
| 5 | Documentation (ADR, coding standards, golden example, testing strategy, skills, api spec, user guides) | ✅ |
| 6 | AI agent skills (40 SKILL.md files, taxonomy in `docs/skills.md`) | ✅ |
| 7 | Debug & operations (`a7 debug logs`, `a7 debug trace`, `a7 update`) | ✅ |
| 8 | E2E tests against a real API7 EE Docker stack (per-resource CRUD, config sync/diff/dump/validate, debug, completion, version) | ✅ |

For the detailed checklist of items under each phase see `PRD.md` "Implementation Phases".

---

## GA Readiness Handoff (Current Focus)

Tracking issue: **api7/a7#22**. Scope: management-plane validation against API7 EE 3.9.12, not new enterprise features and not data-plane traffic forwarding.

Working items live as sub-issues of #22. Exit criteria (copied from #22 for visibility):

- Core resource commands work against real API7 EE.
- E2E covers full CRUD round-trips through the CLI.
- `config sync/diff/dump/validate` work with service-centered resources.
- Unsupported resources are removed or clearly blocked.
- Docs and skills only describe supported workflows.
- CI green on `master`.
- Release notes state supported API7 EE versions and unsupported resources.

`docs/ga-test-plan.md` is the per-resource test plan; `docs/ga-test-report.md` records run dispositions.

---

## Deferred (Post-GA)

Previously tracked as "Phase 9". These are **not** in GA scope and not committed for the GA release. They are recorded so the design intent is preserved; revisit only after GA stability is locked in.

| Area | Notes |
|------|-------|
| Token management (`a7 token ...`) | Access-token CRUD on the control plane. |
| RBAC (`a7 user`, `a7 role`, `a7 permission-policy`) | User and role administration. |
| Developer Portal (`a7 portal ...`) | Applications, developers, subscriptions. |
| Custom plugin management (`a7 custom-plugin ...`) | When and if the EE API surface stabilizes. |
| Audit log querying (`a7 audit-log list`) | |
| System settings (`a7 system settings ...`) | |
| Session login (`a7 auth login|logout`) | Cookie-based session auth; access tokens cover GA. |
| Interactive mode | Fuzzy selection across resources. |
| Extension / plugin system | Out-of-tree CLI extensions. |
| Bulk operations | Multi-resource batch CRUD. |

These items are listed under "Deferred (Post-GA)" in `PRD.md`.
