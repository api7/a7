//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugTrace_JSONOutput(t *testing.T) {
	env := setupEnv(t)
	routeID := "e2e-debug-trace-route"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	// Create a route for tracing.
	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"uri": "/debug-trace-test",
		"upstream": {
			"type": "roundrobin",
			"nodes": {"%s": 1}
		}
	}`, routeID, strings.TrimPrefix(httpbinURL, "http://"))

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	_, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

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
	env := setupEnv(t)
	routeID := "e2e-debug-trace-method"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"uri": "/debug-trace-method",
		"methods": ["GET", "POST"],
		"upstream": {
			"type": "roundrobin",
			"nodes": {"%s": 1}
		},
		"plugins": {
			"proxy-rewrite": {
				"uri": "/post"
			}
		}
	}`, routeID, strings.TrimPrefix(httpbinURL, "http://"))

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	_, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

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
	env := setupEnv(t)
	routeID := "e2e-debug-trace-headers"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"uri": "/debug-trace-headers",
		"upstream": {
			"type": "roundrobin",
			"nodes": {"%s": 1}
		}
	}`, routeID, strings.TrimPrefix(httpbinURL, "http://"))

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	_, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

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
	env := setupEnv(t)
	routeID := "e2e-debug-trace-host"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"uri": "/debug-trace-host",
		"host": "trace.example.com",
		"upstream": {
			"type": "roundrobin",
			"nodes": {"%s": 1}
		}
	}`, routeID, strings.TrimPrefix(httpbinURL, "http://"))

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	_, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

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
	env := setupEnv(t)
	routeID := "e2e-debug-trace-path"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"uri": "/debug-trace-path/*",
		"upstream": {
			"type": "roundrobin",
			"nodes": {"%s": 1}
		},
		"plugins": {
			"proxy-rewrite": {
				"uri": "/get"
			}
		}
	}`, routeID, strings.TrimPrefix(httpbinURL, "http://"))

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	_, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

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
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "debug", "trace", "nonexistent-route-12345",
		"-g", gatewayGroup,
		"--gateway-url", gatewayURL,
		"-o", "json",
	)
	assert.Error(t, err)
}

func TestDebugTrace_YAMLOutput(t *testing.T) {
	env := setupEnv(t)
	routeID := "e2e-debug-trace-yaml"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"uri": "/debug-trace-yaml",
		"upstream": {
			"type": "roundrobin",
			"nodes": {"%s": 1}
		}
	}`, routeID, strings.TrimPrefix(httpbinURL, "http://"))

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	_, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

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
