# Testing Strategy

## Principles

- CLI behavior that depends on API7 EE or APISIX Admin API must be verified with E2E tests against a real environment.
- Unit tests are only allowed for self-contained logic that does not need mocked external systems.
- Do not add new command-level tests that mock the Admin API, gateway, or control-plane behavior.
- New or modified E2E coverage must be written with Ginkgo.

## Test Pyramid For `a7`

### Pure unit tests

Allowed targets:

- parsing and normalization helpers
- config merge / override rules
- output formatting helpers
- deterministic business logic with no network, process, or filesystem side effects beyond temp files

Examples:

- `internal/config`
- `internal/update`
- helper logic in `pkg/cmd/debug/logs`
- helper logic in `pkg/cmd/debug/trace`
- API client envelope / transport helpers that can be tested with stub round trippers instead of HTTP servers

### E2E tests

Required targets:

- all command flows that talk to API7 EE or APISIX Admin API
- CRUD coverage for runtime resources
- traffic assertions that require a real gateway
- auth, route, service, secret, stream-route, and debug flows

E2E tests live in `test/e2e/` behind the `e2e` build tag and are run with Ginkgo.

## What To Remove

- Existing unit tests that rely on `pkg/httpmock`
- Existing unit tests that spin up fake HTTP servers to emulate Admin API behavior for command coverage
- Any new tests that validate CLI request construction by mocking remote APIs instead of exercising the real system

## Local E2E Environment

Required environment variables:

- `A7_ADMIN_URL`
- `A7_TOKEN`

Optional but strongly recommended for traffic coverage:

- `A7_GATEWAY_URL`
- `HTTPBIN_URL`
- `A7_GATEWAY_GROUP`

## Running Tests

Pure unit tests:

```bash
go test ./... -count=1
```

E2E tests:

```bash
make test-e2e
```

Equivalent direct command:

```bash
go run github.com/onsi/ginkgo/v2/ginkgo -r --procs=1 --tags=e2e --timeout=45m ./test/e2e/...
```

## Authoring Guidance

- Prefer `Ordered` suites only when resource lifecycle or environment reuse requires it.
- Use shared helpers for context creation, cleanup, and propagation polling.
- When live gateway behavior may depend on optional local infrastructure, fail only on product regressions and `Skip` on missing external capabilities.
- Keep E2E assertions aligned with the current EE data model:
  - routes should use `service_id`
  - auth flows should use `consumer create + credential create`
  - file-based updates should not require redundant flags that are already present in the payload
