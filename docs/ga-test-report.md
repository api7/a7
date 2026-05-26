# a7 GA Test Report

Execution of [`docs/ga-test-plan.md`](./ga-test-plan.md) against a locally deployed API7 EE.

## Environment

| Item | Value |
|---|---|
| Date | 2026-05-14 |
| API7 EE version | **3.9.12** (image `api7/api7-ee-3-integrated:v3.9.12`; `/api/version` → `v3.9.12`) |
| `a7 version` | `7b67da5` (then `7b67da5-dirty` after the fixes in this report) |
| Admin URL | `https://localhost:7443` |
| Gateway group | `default` |
| Gateway instance | `api7-ee-gateway-1` — `status: Healthy`, `compatibility: Compatible` |
| Deviations | (1) `a7 plugin-config` command still present — Task #2 not yet done. (2) Data-plane traffic tests not run (out of scope per plan; `A7_GATEWAY_URL`/`HTTPBIN_URL` unset). |

## Summary

All four phases executed. **5 real bugs found, all fixed with E2E + unit coverage.** Final automated suite: **128 passed / 10 skipped / 0 failed** (skips are the intentionally-unset data-plane and gateway-group-CRUD tests).

## Results

| Phase | Resource / area | Result | Bug? | E2E test added | Notes |
|---|---|---|---|---|---|
| A | automated suite | PASS | — | — | 122→128 pass, 10 skip, 0 fail. Skips = data-plane traffic + gateway-group CRUD (intentionally disabled). |
| B | context | PASS | — | — | create/list/use/current/delete, json output, error case. |
| B | gateway-group | PASS | — | — | full CRUD; non-interactive `delete` needs `--force`. |
| B | service | PASS | — | — | create (file+flags), get, list, update, export. Map-form upstream nodes correctly rejected (400). |
| B | route | PASS | — | — | create (file+flags), get, list, update, export, delete. `get`/`delete` take id only; `update` uses `--uri`. |
| B | consumer | PASS | — | — | CRUD + export. MINOR: file `description:` key silently dropped (accepted key is `desc`). |
| B | credential | PASS | — | — | CRUD via server-returned id. MINOR: `create [id]` positional is treated as `name` (id is server-generated). |
| B | ssl | **FIXED** | **BUG-1** | `TestSSL_UpdateFlagsRequireCertAndKey` | flag-based `ssl update` silently lost partial updates. |
| B | secret | PASS | — | — | CRUD (file+flags); id format is `provider/id`. |
| B | global-rule | PASS | — | — | CRUD (file+flags), export; id must equal the plugin name. MINOR: flag `--id` is required but its value is ignored by EE. |
| B | plugin | **FIXED** | **BUG-3** | `TestPlugin_GetYAML` | `plugin get -o yaml` emitted JSON. list/get otherwise fine. |
| B | plugin-metadata | **FIXED** | **BUG-2** | `TestPluginMetadata_GetYAML` | `plugin-metadata get -o yaml` emitted a byte-array. CRUD otherwise fine. |
| B | proto | PASS | — | — | CRUD (file+flags), export. MINOR: `--desc` / `desc:` silently dropped. |
| B | stream-route | **FIXED** | **BUG-4** | `TestStreamRoute_CreateWithFlags`, `TestStreamRoute_CreateRequiresName` | `create` had no `--name` flag; EE requires `name`. CRUD via `-f` worked. |
| C | unsupported commands | PASS | — | — | `upstream` / `consumer-group` / `service-template` removed. `plugin-config` still present (Task #2 pending) — gives a clear runtime error. `stream-route` works. |
| C | declarative unsupported sections | **FIXED** | **BUG-5** | `TestConfigValidate_RejectsUnsupportedServiceTemplates` | `upstreams`/`consumer_groups` rejected; `service_templates` was silently accepted. `plugin_configs` rejection is expected-pending per plan. |
| D | config dump/validate/diff/sync | PASS | — | — | dump → valid YAML, supported sections only. validate/diff/sync round-trip converges; `--dry-run` applies nothing; post-sync diff clean; route update verified persisted. |

## Bugs found & fixed

Per the bug-handling protocol, each bug was reproduced with the real binary, given failing E2E coverage, fixed, and re-verified.

### BUG-1 — `ssl update` silently lost partial flag-based updates
`a7 ssl update <id> --sni new` (without `--cert`/`--key`) exited 0 and echoed the new SNI, but the server was never updated. Root cause: the command does a GET-merge-PUT, but API7 EE never returns `cert`/`key` on GET, so the merged PUT dropped the certificate material and the EE ignored it.
**Fix:** `pkg/cmd/ssl/update/update.go` now requires `--cert` and `--key` for flag-based updates (consistent with `ssl create`), with a clear error pointing to the `-f` path. `-f` updates with a full definition are unaffected.

### BUG-2 — `plugin-metadata get -o yaml` emitted a byte-array
The YAML output was a list of integers. Root cause: `pkg/cmd/plugin-metadata/get/get.go` passed `json.RawMessage` (a `[]byte`) to the YAML encoder.
**Fix:** pass the decoded `map` to the exporter.

### BUG-3 — `plugin get -o yaml` emitted JSON
Root cause: `pkg/cmd/plugin/get/get.go` only special-cased `-o json`; every other value fell through to a hardcoded JSON encoder.
**Fix:** honor any explicit `--output` value via the exporter.

### BUG-4 — `stream-route create` had no `--name` flag
API7 EE 3.9.12 requires `name` on stream routes, but the create command exposed no `--name` flag, making flag-based creation impossible.
**Fix:** added a `Name` field to `api.StreamRoute`, a `--name` flag, and required-field validation on both the flag and `-f` paths.

### BUG-5 — `config validate` silently accepted `service_templates`
A declarative file with a top-level `service_templates:` section validated as "Config is valid". Root cause: `api.ConfigFile` had no field for it, so the section was dropped on unmarshal.
**Fix:** added a `ServiceTemplates` field and an explicit rejection in both `config validate` and `configutil.ValidateSupportedSections` (used by `diff`/`sync`), mirroring `upstreams`/`consumer_groups`.

## Minor observations (not fixed — low severity / by-design)

- **consumer**: `-f` file with a `description:` key is silently ignored; the accepted key is `desc`.
- **credential**: `create [id]` help text is misleading — the positional arg becomes the `name`; the `id` is server-generated. This is codified by `TestCredential_CreateWithPositionalID`, so it is current intended behavior.
- **global-rule**: flag-based `create` requires `--id`, but API7 EE forces the id to equal the single plugin key, so the `--id` value is effectively ignored. File-based create errors clearly on a mismatch.
- **proto**: `--desc` and a `desc:` file field are silently dropped.
- **stream-route / EE behavior**: API7 EE rejects stream routes bound to a service it has classified as HTTP. Binding to a `type: stream` service works reliably. `a7` surfaces the EE error cleanly; this is EE-side behavior, not an `a7` bug.
- **service**: the EE's schema-mismatch error (e.g. map-form upstream nodes) is correct and actionable but very verbose (dumps the full JSON schema).
- **tooling**: `golangci-lint run ./...` reports spurious `undefined: yaml (typecheck)` on files that correctly import `gopkg.in/yaml.v3` (including untouched files). `go build`, `go vet`, and `go test` are all clean — this is a pre-existing lint-environment issue, unrelated to these changes.

## Exit criteria

| Criterion | Status |
|---|---|
| Phase A suite green against the local instance | ✅ 128 pass / 10 skip / 0 fail |
| Phase B round-trip passes for every resource | ✅ all 13 resources pass (ssl/plugin/plugin-metadata/stream-route after fixes) |
| Phase C confirms unsupported resources absent/blocked and `stream-route` works | ✅ — except `plugin-config` command still present (Task #2 pending, tracked) |
| Phase D declarative config works with service-centered resources | ✅ dump/validate/diff/sync all verified |
| Every bug has a corresponding E2E test and a fix | ✅ 5/5 fixed with E2E + unit coverage |

## Follow-ups

- **Task #2** (remove the `plugin-config` standalone command) is still outstanding. Once done, Phase C also expects the declarative `plugin_configs` top-level section to be rejected — the `service_templates` rejection added here is the template for that change.

---

# Run 2 (post-#31, post-#34)

## Environment

| Item | Value |
|---|---|
| Date | 2026-05-23 |
| API7 EE version | **3.9.12** (`/api/version` → `v3.9.12`) |
| `a7 version` | `ac31b9d` (master after PR #34 merged) |
| Admin URL | `https://localhost:7443` |
| Gateway group | `default` |
| Deviations | Data-plane traffic tests not run (out of scope per plan). |

## Summary

Re-run after the Run 1 fixes (PR #31) and the `plugin-config` removal (PR #34) merged to master. **All 6 targeted regression checks hold.** Each of the 13 core resources round-trips. **5 new findings surfaced (3 bugs + 1 cosmetic + 1 UX)** — none block GA, but the 3 bugs each warrant a sub-issue and a test-before-fix.

## Regression checks (all pass)

| Check | Source | Result |
|---|---|---|
| `ssl update` without `--cert/--key` → clean error | PR #31 BUG-1 | ✅ |
| `plugin-metadata get -o yaml` → YAML map | PR #31 BUG-2 | ✅ |
| `plugin get -o yaml` → YAML (not JSON) | PR #31 BUG-3 | ✅ |
| `stream-route create` without `--name` → error; `-f -o yaml` → YAML map | PR #31 BUG-4 + review fixup | ✅ |
| `config validate` rejects all 4 unsupported sections (incl. empty `[]`) | PR #31 BUG-5 + PR #34 | ✅ |
| `a7 plugin-config` / `upstream` / `consumer-group` / `service-template` → `unknown command` | PR #34 | ✅ |

## New findings

| # | Severity | Resource | Finding | Disposition |
|---|---|---|---|---|
| R2-1 | 🟡 Bug | route | README documents `a7 route update <id> --desc "..."` but neither `route create` nor `route update` exposes a `--desc` flag. Description is only settable through `-f`. | ✅ Fixed in PR #39 / closes #35 |
| R2-2 | 🟡 UX | route | `a7 route list -g default` errors with `--service-id is required by API7 EE`. The e2e helper iterates services to aggregate routes; the CLI doesn't. | #42 (post-GA candidate) |
| R2-3 | 🟡 Bug | credential | `a7 credential create smoke-cred-X --consumer Y ...` returned a server-assigned UUID, ignoring the positional id. (Run 1 noted this as "intended" via `TestCredential_CreateWithPositionalID` — needs re-confirmation; if intended, drop the misleading `[id]` from the help.) | #36 |
| R2-4 | 🟡 Bug | global-rule | `a7 global-rule create --id X --plugins-json '{"cors":...}'` ignores `--id`; resource is created at `id=cors`. Run 1 noted "value is ignored by EE" as a minor; treat as a real bug: the CLI shouldn't accept a flag it will silently override. | #37 |
| R2-5 | 🟢 Cosmetic | stream-route | `-o yaml` renders `created_at: 1.779521636e+09` in scientific notation. Should be plain integer. | Note only, untracked |

## Exit criteria (Run 2)

| Criterion | Status |
|---|---|
| API7 EE version pinned to 3.9.12 in writing | ✅ |
| All Run 1 fixes hold against current master | ✅ 6/6 regression checks pass |
| Phase C unsupported commands gone (including `plugin-config`) | ✅ all 4 return `unknown command` |
| Phase D `config dump/validate` work on a clean instance | ✅ (no Run 2 dirty-state diff test, but Run 1 covered diff/sync convergence) |
| Each new bug gets a tracked sub-issue with test-before-fix protocol | ✅ #35 (fixed via PR #39), #36, #37, #42 all filed |
