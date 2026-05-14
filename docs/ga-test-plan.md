# a7 GA Test Plan — Local API7 EE 3.9.12

Execution plan for **Task #3 (real CLI smoke tests)** and **Task #4 (complete E2E coverage)**
from the GA Readiness Handoff. Run against a locally deployed API7 EE.

## Scope

- **In scope**: management-plane validation of the `a7` CLI — CRUD round-trips, declarative
  config, output formats, error handling, and confirmation that unsupported resources are
  removed/blocked.
- **Out of scope**: real gateway data-plane traffic forwarding. Per the handoff, traffic
  tests stay in the gateway repository. The separately-running gateway container is only
  needed to keep the EE deployment healthy; we do **not** assert on proxied traffic here.

## Local environment (assumed already running)

| Component | How it runs | Notes |
|---|---|---|
| API7 EE dashboard / control-plane + dp-manager + PostgreSQL | docker compose | the management API `a7` talks to |
| API7 gateway | standalone docker container | connected to the dp-manager; health only |

### 0. Pre-flight — capture connection facts

Fill these in before doing anything else:

```
A7_ADMIN_URL   = https://localhost:7443      # control-plane HTTPS port — confirm
A7_TOKEN       = a7ee-<your-access-token>    # access token from the dashboard UI / API
A7_GATEWAY_GROUP = default                   # resolved to a UUID at runtime
```

Confirm before proceeding:

1. **Reachability** — `curl -k -H "X-API-KEY: $A7_TOKEN" $A7_ADMIN_URL/api/gateway_groups`
   returns `200` with a non-empty `list`.
2. **Version is 3.9.12** — check the dashboard image tag in `docker compose ps` /
   `docker inspect`, or the dashboard `/api/version` endpoint. Record the exact version.
   If it is **not** 3.9.12, note the deviation in the report — the handoff specifically
   wants 3.9.12 validated.
3. **Gateway is registered & healthy** — the standalone gateway container shows up under
   the gateway group (dashboard UI or `GET /api/gateway_groups/{id}/instances`). This only
   needs to be healthy; we are not testing traffic through it.

## Build

```bash
cd ~/Desktop/repos/xisipa/a7
make build          # -> ./bin/a7
./bin/a7 version    # record the version string in the report
```

All manual steps below use `./bin/a7`. The automated suite (Phase A) builds its own binary.

---

## Phase A — Run the existing automated E2E suite

Establishes a known-good baseline against the local 3.9.12 instance before manual work.

```bash
cd ~/Desktop/repos/xisipa/a7
export A7_ADMIN_URL="https://localhost:7443"
export A7_TOKEN="a7ee.xxxxxxxx"
export A7_GATEWAY_GROUP="default"
# A7_GATEWAY_URL and HTTPBIN_URL intentionally left UNSET — data-plane tests are skipped.
make test-e2e
```

**Expected**: suite passes, or fails only in ways explained by known gaps. For every
failure, decide: real bug, missing/incorrect test, or environment issue. Record each one.

> The standard `test/e2e/docker-compose.yml` is **not** used here — you already have a
> local deployment. Only the env vars above are needed.

---

## Phase B — Manual CRUD smoke tests (per resource)

For **each** resource below, run the handoff's round-trip pattern with `./bin/a7`:

1. `create` (both flag-based and `-f file` where supported)
2. `get` — assert the returned config matches what was sent
3. `list` — assert the new resource appears
4. `update` — change a field
5. `get` again — assert the update took effect
6. `export` (where supported) — assert valid YAML/JSON
7. `delete` (try with and without `--force`)
8. `get` / `list` — assert it is gone

Also exercise, at least once per resource: `-o json`, `-o yaml`, and default table output;
plus one deliberate error case (bad ID, missing `--gateway-group`, malformed `-f` file) and
confirm the error message is clear and actionable.

**Use unique resource IDs** (`a7-ga-<resource>-<timestamp>`) and clean up everything, even
on a local instance — it keeps reruns deterministic.

### Resource checklist

| Resource | Command | create/get/list/update/delete | export | Notes |
|---|---|:--:|:--:|---|
| context | `a7 context` | ☐ | — | create/use/list/current/delete; no gateway group needed |
| gateway-group | `a7 gateway-group` | ☐ | — | control-plane `/api/*` |
| service | `a7 service` | ☐ | ☐ | inline upstream — **node array format only** |
| route | `a7 route` | ☐ | ☐ | requires `--service-id`; route is service-centered |
| consumer | `a7 consumer` | ☐ | ☐ | |
| credential | `a7 credential` | ☐ | — | nested under a consumer |
| ssl | `a7 ssl` | ☐ | ☐ | **shared/global** — unique IDs, careful cleanup |
| secret | `a7 secret` | ☐ | — | **shared/global** — unique IDs, careful cleanup |
| global-rule | `a7 global-rule` | ☐ | ☐ | **shared/global** — affects whole gateway group |
| plugin | `a7 plugin` | ☐ (list/get only) | — | read-only: list plugins, get schema |
| plugin-metadata | `a7 plugin-metadata` | ☐ (no list) | — | **shared/global**; keyed by plugin name |
| proto | `a7 proto` | ☐ | ☐ | protobuf definitions |
| stream-route | `a7 stream-route` | ☐ | ☐ | **confirmed exposed** by the CP — must work |

> **Shared/global resources** (ssl, secret, global-rule, plugin-metadata) are visible
> gateway-group-wide. On a shared environment they collide between CI runs; even locally,
> assert you only touch IDs you created and never delete pre-existing resources.

### Inline upstream format (API7 EE)

`nodes` must be an **array of objects**. The APISIX map form is rejected.

```yaml
upstream:
  type: roundrobin
  nodes:
    - host: 127.0.0.1
      port: 8080
      weight: 1
```

---

## Phase C — Verify unsupported resources are blocked

Confirm the resources removed in PR #21 (and the one pending in Task #2) are gone or clearly
rejected — not present as broken commands.

| Resource | Expected behavior |
|---|---|
| `a7 upstream ...` | command does not exist (unknown command error) |
| `a7 consumer-group ...` | command does not exist |
| `a7 service-template ...` | command does not exist |
| `a7 plugin-config ...` | **after Task #2**: command does not exist |
| `a7 stream-route ...` | **works** — full CRUD (verified exposed by the control plane) |

Declarative config — these top-level sections must be **rejected with a clear error**:

```yaml
upstreams: [...]          # -> validation error
consumer_groups: [...]    # -> validation error
service_templates: [...]  # -> validation error
plugin_configs: [...]     # -> validation error (after Task #2)
```

> Caveat: the control plane still accepts `plugin_configs` *inside batch config-validation
> payloads*. The rejection applies to `a7`'s standalone command and top-level declarative
> section, not to that internal validation path.

---

## Phase D — Declarative config

Run the full declarative workflow end to end:

1. `a7 config dump` — export the live gateway-group config to a YAML file. Assert it is
   valid YAML and contains only supported resource sections.
2. `a7 config validate -f dump.yaml` — assert it passes for a known-good file, and fails
   with a clear message for a file containing an unsupported section (see Phase C).
3. `a7 config diff -f dump.yaml` — assert "no diff" against the just-dumped state; then
   edit a field and assert the diff is reported correctly.
4. `a7 config sync -f dump.yaml` — apply changes. Verify `--dry-run` first if available,
   then a real sync, then `diff` again to confirm convergence.

Confirm all four work with **service-centered** API7 EE resources (no standalone upstreams).

---

## Bug-handling protocol

Per the handoff: **when a bug is found, add or update E2E coverage before fixing it.**

1. Reproduce with the real binary; capture exact command, output, and HTTP status.
2. Write or update a `test/e2e/<resource>_test.go` case that fails for the same reason.
3. Fix the code.
4. Re-run that test plus `make test-e2e` to confirm green.
5. Do **not** change code purely to satisfy a test if it contradicts real API7 EE behavior.

---

## Reporting

Record results in a table and attach it to the GA tracking issue / PR:

| Phase | Resource / area | Result | Bug? | E2E test added | Notes |
|---|---|---|---|---|---|
| A | automated suite | | | | |
| B | service | | | | |
| B | route | | | | |
| ... | ... | | | | |
| C | unsupported resources | | | | |
| D | config dump/validate/diff/sync | | | | |

Also capture: exact API7 EE version tested, `a7 version` string, date, and any environment
deviations from "API7 EE 3.9.12".

## Exit criteria

- Phase A suite green against the local instance.
- Phase B round-trip passes for every resource in the checklist.
- Phase C confirms all unsupported resources are absent/blocked and `stream-route` works.
- Phase D confirms declarative config works with service-centered resources.
- Every bug found has a corresponding E2E test and a fix (or a tracked follow-up).
