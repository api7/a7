//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createDebugTraceRoute(t *testing.T, env []string, serviceID, routeID, path string, extraFields string) {
	t.Helper()
	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": %q,
		"service_id": %q,
		"paths": [%q]%s
	}`, routeID, routeID, serviceID, path, extraFields)

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
}

func waitForDebugTraceRoute(t *testing.T, path string) {
	t.Helper()
	status, err := waitForGatewayStatus(gatewayURL+path, func() (*http.Request, error) {
		return http.NewRequest("GET", gatewayURL+path, nil)
	}, func(code int) bool {
		return code != 404
	}, 15*time.Second)
	require.NoError(t, err)
	if status == 404 {
		t.Skipf("route %s did not propagate to the local gateway within timeout", path)
	}
}

func TestDebugTrace_JSONOutput(t *testing.T) {
	requireGatewayURL(t)
	requireHTTPBin(t)
	env := setupEnv(t)
	svcID := "e2e-debug-trace-svc"
	routeID := "e2e-debug-trace-route"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)
	createDebugTraceRoute(t, env, svcID, routeID, "/debug-trace-test", "")
	waitForDebugTraceRoute(t, "/debug-trace-test")

	// Trace the route with JSON output.
	stdout, stderr, err := runA7WithEnv(env, "debug", "trace", routeID,
		"-g", gatewayGroup,
		"--gateway-url", gatewayURL,
		"-o", "json",
	)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result), "should be valid JSON")
	assert.Contains(t, result, "route")
	assert.Contains(t, result, "request")
	assert.Contains(t, result, "response")
}

func TestDebugTrace_WithMethod(t *testing.T) {
	requireGatewayURL(t)
	requireHTTPBin(t)
	env := setupEnv(t)
	svcID := "e2e-debug-trace-method-svc"
	routeID := "e2e-debug-trace-method"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)
	createDebugTraceRoute(t, env, svcID, routeID, "/debug-trace-method",
		`, "methods": ["GET", "POST"], "plugins": {"proxy-rewrite": {"uri": "/post"}}`)
	waitForDebugTraceRoute(t, "/debug-trace-method")

	// Trace with --method POST.
	stdout, stderr, err := runA7WithEnv(env, "debug", "trace", routeID,
		"-g", gatewayGroup,
		"--gateway-url", gatewayURL,
		"--method", "POST",
		"-o", "json",
	)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	req, ok := result["request"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "POST", req["method"])
}

func TestDebugTrace_WithHeaders(t *testing.T) {
	requireGatewayURL(t)
	requireHTTPBin(t)
	env := setupEnv(t)
	svcID := "e2e-debug-trace-headers-svc"
	routeID := "e2e-debug-trace-headers"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)
	createDebugTraceRoute(t, env, svcID, routeID, "/debug-trace-headers", "")
	waitForDebugTraceRoute(t, "/debug-trace-headers")

	// Trace with custom header.
	stdout, stderr, err := runA7WithEnv(env, "debug", "trace", routeID,
		"-g", gatewayGroup,
		"--gateway-url", gatewayURL,
		"--header", "X-Custom: test-value",
		"-o", "json",
	)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	assert.NotEmpty(t, stdout)
}

func TestDebugTrace_WithHost(t *testing.T) {
	requireGatewayURL(t)
	requireHTTPBin(t)
	env := setupEnv(t)
	svcID := "e2e-debug-trace-host-svc"
	routeID := "e2e-debug-trace-host"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)
	createDebugTraceRoute(t, env, svcID, routeID, "/debug-trace-host",
		`, "host": "trace.example.com"`)
	waitForDebugTraceRoute(t, "/debug-trace-host")

	// Trace with --host flag.
	stdout, stderr, err := runA7WithEnv(env, "debug", "trace", routeID,
		"-g", gatewayGroup,
		"--gateway-url", gatewayURL,
		"--host", "trace.example.com",
		"-o", "json",
	)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
}

func TestDebugTrace_WithPath(t *testing.T) {
	requireGatewayURL(t)
	requireHTTPBin(t)
	env := setupEnv(t)
	svcID := "e2e-debug-trace-path-svc"
	routeID := "e2e-debug-trace-path"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)
	createDebugTraceRoute(t, env, svcID, routeID, "/debug-trace-path/*",
		`, "plugins": {"proxy-rewrite": {"uri": "/get"}}`)
	waitForDebugTraceRoute(t, "/debug-trace-path/sub")

	// Trace with --path flag override.
	stdout, stderr, err := runA7WithEnv(env, "debug", "trace", routeID,
		"-g", gatewayGroup,
		"--gateway-url", gatewayURL,
		"--path", "/debug-trace-path/sub",
		"-o", "json",
	)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	req, ok := result["request"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, req["url"], "/debug-trace-path/sub")
}

func TestDebugTrace_NonexistentRoute(t *testing.T) {
	requireGatewayURL(t)
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "debug", "trace", "nonexistent-route-12345",
		"-g", gatewayGroup,
		"--gateway-url", gatewayURL,
		"-o", "json",
	)
	assert.Error(t, err)
}

func TestDebugTrace_YAMLOutput(t *testing.T) {
	requireGatewayURL(t)
	requireHTTPBin(t)
	env := setupEnv(t)
	svcID := "e2e-debug-trace-yaml-svc"
	routeID := "e2e-debug-trace-yaml"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)
	createDebugTraceRoute(t, env, svcID, routeID, "/debug-trace-yaml", "")
	waitForDebugTraceRoute(t, "/debug-trace-yaml")

	stdout, stderr, err := runA7WithEnv(env, "debug", "trace", routeID,
		"-g", gatewayGroup,
		"--gateway-url", gatewayURL,
		"-o", "yaml",
	)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	assert.Contains(t, stdout, "route:")
}

func TestDebugLogs_FromFile(t *testing.T) {
	// Create a temporary log file with known content.
	logContent := ""
	for i := 1; i <= 20; i++ {
		logContent += fmt.Sprintf("2025/01/01 00:00:%02d [info] line %d\n", i, i)
	}

	logFile := filepath.Join(t.TempDir(), "test-access.log")
	require.NoError(t, os.WriteFile(logFile, []byte(logContent), 0644))

	// Read last 5 lines from file (no env needed — file tailing is local).
	stdout, stderr, err := runA7("debug", "logs", "--file", logFile, "-n", "5")
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	assert.Contains(t, stdout, "line 20")
	assert.Contains(t, stdout, "line 16")
}

func TestDebugLogs_FromFileAllLines(t *testing.T) {
	logContent := "line1\nline2\nline3\n"
	logFile := filepath.Join(t.TempDir(), "test-all.log")
	require.NoError(t, os.WriteFile(logFile, []byte(logContent), 0644))

	stdout, stderr, err := runA7("debug", "logs", "--file", logFile, "-n", "100")
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	assert.Contains(t, stdout, "line1")
	assert.Contains(t, stdout, "line2")
	assert.Contains(t, stdout, "line3")
}

func TestDebugLogs_FileNotFound(t *testing.T) {
	_, _, err := runA7("debug", "logs", "--file", "/nonexistent/path/to/log.file")
	assert.Error(t, err)
}
