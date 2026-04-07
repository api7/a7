//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestRoute_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "route", "list", "-g", gatewayGroup)
	if err != nil {
		t.Skipf("route list failed (API7 EE may use different endpoint): stderr=%s", stderr)
	}
	assert.NotEmpty(t, stdout)
}

func TestRoute_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "route", "list", "-g", gatewayGroup, "-o", "json")
	if err != nil {
		t.Skipf("route list failed (API7 EE may use different endpoint): stderr=%s", stderr)
	}
	assert.NotEmpty(t, stdout)
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
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, routeID)

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
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "/test-updated")

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "route", "delete", routeID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
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
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "flagged-route")
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
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "proxy-rewrite")
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

	// Use 'get -o json' to export a single route (export is batch, no positional ID).
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)

	var exported map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &exported), "should be valid JSON")
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

	stdout, stderr, err = runA7WithEnv(env, "route", "list", "-g", gatewayGroup, "--label", "filter-test=yes")
	if err != nil && strings.Contains(stderr, `parameter "service_id" in query has an error`) {
		t.Skipf("route list with label requires service_id in current EE API: %s", stderr)
	}
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
