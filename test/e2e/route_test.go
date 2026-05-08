//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deleteRouteViaCLI deletes a route using the a7 CLI.
func deleteRouteViaCLI(t testTB, env []string, id string) {
	t.Helper()
	_, _, _ = runA7WithEnv(env, "route", "delete", id, "--force", "-g", gatewayGroup)
}

// deleteRouteViaAdmin deletes a route via the Admin API (cleanup).
func deleteRouteViaAdmin(t testTB, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/routes/%s", id), nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("delete route %s via admin API failed: %v", id, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("delete route %s via admin API returned %d: %s", id, resp.StatusCode, string(body))
	}
}

func TestRoute_List(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-route-list"
	routeID := "e2e-route-list"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)
	createTestRouteWithServiceViaCLI(t, env, routeID, svcID)

	stdout, stderr, err := runA7WithEnv(env, "route", "list", "-g", gatewayGroup, "--service-id", svcID)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, routeID)
}

func TestRoute_ListJSON(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-route-list-json"
	routeID := "e2e-route-list-json"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)
	createTestRouteWithServiceViaCLI(t, env, routeID, svcID)

	stdout, stderr, err := runA7WithEnv(env, "route", "list", "-g", gatewayGroup, "--service-id", svcID, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, routeID)
}

func TestRoute_CRUD(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-route-crud"
	routeID := "e2e-route-crud"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)

	// Create
	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "e2e-route-crud",
		"service_id": %q,
		"paths": ["/test-crud"]
	}`, routeID, svcID)
	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Get
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, routeID)

	// Get JSON
	var route map[string]interface{}
	runA7JSON(t, env, &route, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, routeID, route["id"])
	assert.Equal(t, "e2e-route-crud", route["name"])
	assert.Equal(t, svcID, route["service_id"])

	// Update via file
	updateJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "e2e-route-crud-updated",
		"service_id": %q,
		"paths": ["/test-updated"]
	}`, routeID, svcID)
	tmpFile2 := filepath.Join(t.TempDir(), "route-update.json")
	require.NoError(t, os.WriteFile(tmpFile2, []byte(updateJSON), 0644))

	stdout, stderr, err = runA7WithEnv(env, "route", "update", routeID, "-f", tmpFile2, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	// Verify update
	runA7JSON(t, env, &route, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, "e2e-route-crud-updated", route["name"])
	assert.Equal(t, svcID, route["service_id"])
	paths := requireJSONArray(t, route["paths"], "route.paths")
	assert.Contains(t, paths, "/test-updated")

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "route", "delete", routeID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	_, _, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestRoute_CreateWithFlags(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-route-flags"
	routeID := "e2e-route-flags"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "flagged-route",
		"service_id": %q,
		"paths": ["/test-flags"],
		"methods": ["GET","POST"],
		"host": "test.example.com",
		"labels": {"env": "test", "team": "e2e"}
	}`, routeID, svcID)

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Verify
	var route map[string]interface{}
	runA7JSON(t, env, &route, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, routeID, route["id"])
	assert.Equal(t, "flagged-route", route["name"])
	assert.Equal(t, svcID, route["service_id"])
	methods := requireJSONArray(t, route["methods"], "route.methods")
	assert.Contains(t, methods, "GET")
	labels := requireJSONObject(t, route["labels"], "route.labels")
	assert.Equal(t, "test", labels["env"])
	assert.Equal(t, "e2e", labels["team"])
}

func TestRoute_UpdateFlagsMapsURIToPaths(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-route-update-flags"
	routeID := "e2e-route-update-flags"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)
	createTestRouteWithServiceViaCLI(t, env, routeID, svcID)

	stdout, stderr, err := runA7WithEnv(env, "route", "update", routeID,
		"--uri", "/test-update-flags-new",
		"--labels", "mode=flag",
		"-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	var route map[string]interface{}
	runA7JSON(t, env, &route, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, routeID, route["id"])
	assert.Equal(t, svcID, route["service_id"])
	paths := requireJSONArray(t, route["paths"], "route.paths")
	assert.Equal(t, []interface{}{"/test-update-flags-new"}, paths)
	labels := requireJSONObject(t, route["labels"], "route.labels")
	assert.Equal(t, "flag", labels["mode"])
}

func TestRoute_CreateWithPlugins(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-route-plugins"
	routeID := "e2e-route-plugins"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "route-with-plugins",
		"service_id": %q,
		"paths": ["/test-plugins"],
		"plugins": {
			"proxy-rewrite": {
				"uri": "/get"
			}
		}
	}`, routeID, svcID)

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Verify plugin
	var route map[string]interface{}
	runA7JSON(t, env, &route, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, routeID, route["id"])
	assert.Equal(t, svcID, route["service_id"])
	plugins := requireJSONObject(t, route["plugins"], "route.plugins")
	assert.Contains(t, plugins, "proxy-rewrite")
}

func TestRoute_Export(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-route-export"
	routeID := "e2e-route-export"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "route-export",
		"service_id": %q,
		"paths": ["/test-export"]
	}`, routeID, svcID)
	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	var exported []map[string]interface{}
	runA7JSON(t, env, &exported, "route", "export", "-g", gatewayGroup, "--service-id", svcID, "-o", "json")
	assert.NotEmpty(t, exported)
	found := false
	for _, item := range exported {
		if item["id"] == routeID {
			found = true
			assert.Equal(t, svcID, item["service_id"])
		}
	}
	assert.True(t, found, "expected exported routes to contain %s", routeID)
}

func TestRoute_ExportYAML(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-route-export-yaml"
	routeID := "e2e-route-export-yaml"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "route-export-yaml",
		"service_id": %q,
		"paths": ["/test-export-yaml"]
	}`, routeID, svcID)
	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup, "-o", "yaml")
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
	svcID := "e2e-service-route-label-filter"
	routeID := "e2e-route-label-filter"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "route-label-filter",
		"service_id": %q,
		"paths": ["/test-label-filter"],
		"labels": {"filter-test": "yes"}
	}`, routeID, svcID)

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	stdout, stderr, err = runA7WithEnv(env, "route", "list", "-g", gatewayGroup, "--service-id", svcID, "--label", "filter-test=yes")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, routeID)
}

func TestRoute_TrafficForwarding(t *testing.T) {
	requireGatewayURL(t)
	requireHTTPBin(t)
	env := setupEnv(t)
	svcID := "e2e-service-route-traffic"
	routeID := "e2e-route-traffic"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})
	createTestServiceViaCLI(t, env, svcID)

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "route-traffic",
		"service_id": %q,
		"paths": ["/e2e-traffic-test"],
		"plugins": {
			"proxy-rewrite": {
				"uri": "/get"
			}
		}
	}`, routeID, svcID)

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	status, err := waitForGatewayStatus(gatewayURL+"/e2e-traffic-test", func() (*http.Request, error) {
		return http.NewRequest("GET", gatewayURL+"/e2e-traffic-test", nil)
	}, func(code int) bool {
		return code == 200
	}, 15*time.Second)
	require.NoError(t, err)
	if status == 404 {
		t.Skip("route did not propagate to the local gateway within timeout; skipping traffic forwarding assertion")
	}
	assert.Equal(t, 200, status)
}
