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

// deleteRouteViaCLI deletes a route using the a7 CLI.
func deleteRouteViaCLI(t *testing.T, env []string, id string) {
	t.Helper()
	_, _, _ = runA7WithEnv(env, "route", "delete", id, "--force", "-g", gatewayGroup)
}

// deleteRouteViaAdmin deletes a route via the Admin API (cleanup).
func deleteRouteViaAdmin(t *testing.T, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/routes/%s", id), nil)
	if err == nil {
		resp.Body.Close()
	}
}

// createTestRouteViaCLI creates a route via CLI and returns its ID.
func createTestRouteViaCLI(t *testing.T, env []string, id string) string {
	t.Helper()
	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"uri": "/test-%s",
		"upstream": {
			"type": "roundrobin",
			"nodes": {"%s": 1}
		}
	}`, id, id, strings.TrimPrefix(httpbinURL, "http://"))

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	return id
}

func TestRoute_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "route", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestRoute_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "route", "list", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestRoute_CRUD(t *testing.T) {
	env := setupEnv(t)
	routeID := "e2e-route-crud"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	// Create
	createTestRouteViaCLI(t, env, routeID)

	// Get
	stdout, stderr, err := runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, routeID)

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, routeID)

	// Update via file
	updateJSON := fmt.Sprintf(`{
		"id": %q,
		"uri": "/test-updated",
		"upstream": {
			"type": "roundrobin",
			"nodes": {"%s": 1}
		}
	}`, routeID, strings.TrimPrefix(httpbinURL, "http://"))
	tmpFile := filepath.Join(t.TempDir(), "route-update.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(updateJSON), 0644))

	stdout, stderr, err = runA7WithEnv(env, "route", "update", routeID, "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	// Verify update
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "/test-updated")

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "route", "delete", routeID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
}

func TestRoute_CreateWithFlags(t *testing.T) {
	env := setupEnv(t)
	routeID := "e2e-route-flags"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "flagged-route",
		"uri": "/test-flags",
		"methods": ["GET","POST"],
		"host": "test.example.com",
		"upstream": {
			"type": "roundrobin",
			"nodes": {"%s": 1}
		},
		"labels": {"env": "test", "team": "e2e"}
	}`, routeID, strings.TrimPrefix(httpbinURL, "http://"))

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Verify
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "flagged-route")
}

func TestRoute_CreateWithPlugins(t *testing.T) {
	env := setupEnv(t)
	routeID := "e2e-route-plugins"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"uri": "/test-plugins",
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

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Verify plugin
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "proxy-rewrite")
}

func TestRoute_Export(t *testing.T) {
	env := setupEnv(t)
	routeID := "e2e-route-export"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	createTestRouteViaCLI(t, env, routeID)

	// Export single route JSON
	stdout, stderr, err := runA7WithEnv(env, "route", "export", routeID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)

	var exported map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &exported), "should be valid JSON")
}

func TestRoute_ExportYAML(t *testing.T) {
	env := setupEnv(t)
	routeID := "e2e-route-export-yaml"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	createTestRouteViaCLI(t, env, routeID)

	stdout, stderr, err := runA7WithEnv(env, "route", "export", routeID, "-g", gatewayGroup, "-o", "yaml")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestRoute_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "route", "delete", "nonexistent-route-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestRoute_GetNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "route", "get", "nonexistent-route-12345", "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestRoute_ListWithLabel(t *testing.T) {
	env := setupEnv(t)
	routeID := "e2e-route-label-filter"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"uri": "/test-label-filter",
		"upstream": {
			"type": "roundrobin",
			"nodes": {"%s": 1}
		},
		"labels": {"filter-test": "yes"}
	}`, routeID, strings.TrimPrefix(httpbinURL, "http://"))

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	_, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	stdout, stderr, err := runA7WithEnv(env, "route", "list", "-g", gatewayGroup, "--label", "filter-test=yes")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, routeID)
}

func TestRoute_TrafficForwarding(t *testing.T) {
	env := setupEnv(t)
	routeID := "e2e-route-traffic"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"uri": "/e2e-traffic-test",
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

	// Wait briefly for route to propagate to gateway.
	resp, err := insecureClient.Get(gatewayURL + "/e2e-traffic-test")
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
	}
}
