# Testing Strategy

## Test Requirements
- Every exported function must have at least one corresponding test.
- Every command must be tested for:
  - Success cases
  - Error cases
  - TTY output (Table)
  - Non-TTY output (JSON)
- Aim for a code coverage target of 80% or higher for packages within the `pkg/` directory.

## Test File Location
Tests should be located in the same directory as the code they test. For example, `list.go` should have its tests in `list_test.go`.

Store test fixtures in `test/fixtures/<resource>_<action>.json`.

## Test Naming Convention
Follow the pattern `func Test<Function>_<Scenario>(t *testing.T) {}`.

Examples:
- `func TestRouteList_ReturnsTable(t *testing.T) {}`
- `func TestRouteList_EmptyResponse(t *testing.T) {}`
- `func TestRouteList_APIError(t *testing.T) {}`
- `func TestRouteList_JSONOutput(t *testing.T) {}`
- `func TestRouteList_NonTTY(t *testing.T) {}`

## HTTP Mocking Pattern
Use the project's internal `pkg/httpmock` package instead of external mock libraries.

```go
func TestRouteList_Success(t *testing.T) {
    // 1. Create mock registry
    reg := &httpmock.Registry{}
    
    // 2. Register expected request and response
    reg.Register(
        http.MethodGet,
        "/apisix/admin/routes", // Full path with dual-API prefix
        httpmock.JSONResponse("../../../../test/fixtures/route_list.json"),
    )
    
    // 3. Create test factory with mock dependencies
    ios, _, out, _ := iostreams.Test()
    f := &cmd.Factory{
        IOStreams: ios,
        HttpClient: func() (*http.Client, error) {
            return reg.GetClient(), nil
        },
        Config: func() (config.Config, error) {
            return &mockConfig{
                baseURL: "https://localhost:7443",
                token: "a7ee-test-token",
                gatewayGroup: "default",
            }, nil
        },
    }
    
    // 4. Create and execute command
    cmd := list.NewCmdList(f)
    err := cmd.Execute()
    
    // 5. Verify results
    require.NoError(t, err)
    assert.Contains(t, out.String(), "users-api")
    reg.Verify(t)
}
```

## Test Categories

### Unit Tests
Required for every command to verify:
- Command flag parsing
- HTTP request construction (URL, query parameters including `gateway_group_id`)
- Response parsing (handles `ListResponse[T]` and `SingleResponse[T]`)
- Output formatting for both table and JSON
- Error handling for API errors, network issues, and authentication failures

### TTY vs Non-TTY Tests
Every command must have tests for both TTY and non-TTY environments:

```go
func TestRouteList_TTY(t *testing.T) {
    ios, _, _, _ := iostreams.Test()
    ios.SetStdoutTTY(true)
    // Verify table output
}

func TestRouteList_NonTTY(t *testing.T) {
    ios, _, _, _ := iostreams.Test()
    ios.SetStdoutTTY(false)
    // Verify JSON output
}
```

## Test Fixtures
- **Location**: `test/fixtures/`
- **Naming**: `<resource>_<action>.json` (e.g., `route_list.json`)
- **Content**: Use realistic API7 EE responses. Redact any sensitive data.

## What NOT to Test
- Do not test cobra flag binding, as this is handled by the cobra framework.
- Do not test JSON marshaling, which is the responsibility of the standard library.
- Avoid writing integration tests against a real API7 EE instance in unit test files — use e2e tests for that.

## E2E Tests

E2E tests validate the CLI binary against a real API7 EE environment. They live in `test/e2e/` and use the `//go:build e2e` build tag.

### Infrastructure

E2E tests require a running API7 EE instance:

| Variable | Default | Purpose |
|---------|---------|---------|
| `A7_SERVER` | `https://127.0.0.1:7443` | API7 EE Control-plane URL |
| `A7_TOKEN` | (required) | API Access Token |
| `A7_GATEWAY_GROUP` | `default` | Gateway Group for tests |

### Running E2E Tests

**Locally** (requires API7 EE accessible):
```bash
export A7_SERVER=https://your-instance:7443
export A7_TOKEN=a7ee-your-token
make test-e2e
```

### E2E Test File Structure

- `test/e2e/setup_test.go` — `TestMain`, helper functions (`runA7`, `adminAPI`)
- `test/e2e/smoke_test.go` — Basic connectivity checks
- `test/e2e/<resource>_test.go` — Per-resource CRUD lifecycle tests

### Writing E2E Tests

```go
//go:build e2e

package e2e

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRoute_CRUD(t *testing.T) {
    // 1. Create a route via CLI
    stdout, _, err := runA7("route", "create", "--name", "test-route", "--uris", "/test", "--gateway-group", "default")
    require.NoError(t, err)
    assert.Contains(t, stdout, "created")

    // 2. List via CLI
    stdout, _, err = runA7("route", "list", "--gateway-group", "default")
    require.NoError(t, err)
    assert.Contains(t, stdout, "test-route")

    // 3. Cleanup: delete via CLI
    _, _, err = runA7("route", "delete", "test-route", "--gateway-group", "default")
    require.NoError(t, err)
}
```

## Running Tests
Use the following commands to run tests:
- `make test`: Runs all unit tests with race detection.
- `make test-verbose`: Runs unit tests with verbose output.
- `make test-e2e`: Runs E2E tests (requires configured environment).
- `make coverage`: Generates and opens a coverage report.

## Assertions
Use the `testify` library for assertions:
```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

require.NoError(t, err)           // Fatal if an error occurs
assert.Equal(t, expected, actual) // Continue if the assertion fails
assert.Contains(t, output, "ID")  // Check for a substring
```