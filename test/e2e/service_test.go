//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deleteServiceViaCLI deletes a service using the a7 CLI.
func deleteServiceViaCLI(t testTB, env []string, id string) {
	t.Helper()
	_, _, _ = runA7WithEnv(env, "service", "delete", id, "--force", "-g", gatewayGroup)
}

// deleteServiceViaAdmin deletes a service via the Admin API (cleanup).
func deleteServiceViaAdmin(t testTB, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/services/%s", id), nil)
	if err != nil {
		t.Fatalf("delete service %s via admin API failed: %v", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("delete service %s via admin API returned %d: %s", id, resp.StatusCode, string(body))
	}
}

// createTestServiceViaCLI creates a service via CLI.
func createTestServiceViaCLI(t testTB, env []string, id string) {
	t.Helper()
	svcJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "e2e-svc-%s",
		"upstream": {
			"type": "roundrobin",
			"nodes": [{"host": %q, "port": %d, "weight": 1}]
		}
	}`, id, id, upstreamNodeHost(), upstreamNodePort())

	tmpFile := filepath.Join(t.TempDir(), "service.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(svcJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "service", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
}

func TestService_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "service", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestService_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "service", "list", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestService_CRUD(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-crud"
	t.Cleanup(func() { deleteServiceViaAdmin(t, svcID) })

	// Create
	createTestServiceViaCLI(t, env, svcID)

	// Get
	stdout, stderr, err := runA7WithEnv(env, "service", "get", svcID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, svcID)

	// Get JSON
	var service map[string]interface{}
	runA7JSON(t, env, &service, "service", "get", svcID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, svcID, service["id"])
	assert.Equal(t, "e2e-svc-"+svcID, service["name"])

	// Update
	updateJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "e2e-svc-updated",
		"upstream": {
			"type": "roundrobin",
			"nodes": [{"host": %q, "port": %d, "weight": 2}]
		}
	}`, svcID, upstreamNodeHost(), upstreamNodePort())
	tmpFile := filepath.Join(t.TempDir(), "service-update.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(updateJSON), 0644))

	stdout, stderr, err = runA7WithEnv(env, "service", "update", svcID, "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	// Verify update
	runA7JSON(t, env, &service, "service", "get", svcID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, "e2e-svc-updated", service["name"])

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "service", "delete", svcID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	_, _, err = runA7WithEnv(env, "service", "get", svcID, "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestService_UpdateFlagsPreservesName(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-update-flags"
	t.Cleanup(func() { deleteServiceViaAdmin(t, svcID) })
	createTestServiceViaCLI(t, env, svcID)

	stdout, stderr, err := runA7WithEnv(env, "service", "update", svcID,
		"--desc", "updated by flag mode",
		"--host", "flag-service.example.com",
		"--labels", "mode=flag",
		"-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	var service map[string]interface{}
	runA7JSON(t, env, &service, "service", "get", svcID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, svcID, service["id"])
	assert.Equal(t, "e2e-svc-"+svcID, service["name"])
	assert.Equal(t, "updated by flag mode", service["desc"])
	hosts := requireJSONArray(t, service["hosts"], "service.hosts")
	assert.Contains(t, hosts, "flag-service.example.com")
	labels := requireJSONObject(t, service["labels"], "service.labels")
	assert.Equal(t, "flag", labels["mode"])
}

func TestService_Export(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-export"
	t.Cleanup(func() { deleteServiceViaAdmin(t, svcID) })

	createTestServiceViaCLI(t, env, svcID)

	// export is batch-only (cobra.NoArgs); use "get -o json" for single-resource export.
	var service map[string]interface{}
	runA7JSON(t, env, &service, "service", "get", svcID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, svcID, service["id"])
}

func TestService_WithPlugins(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-plugins"
	t.Cleanup(func() { deleteServiceViaAdmin(t, svcID) })

	svcJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "svc-with-plugins",
		"upstream": {
			"type": "roundrobin",
			"nodes": [{"host": %q, "port": %d, "weight": 1}]
		},
		"plugins": {
			"proxy-rewrite": {
				"uri": "/get"
			}
		}
	}`, svcID, upstreamNodeHost(), upstreamNodePort())

	tmpFile := filepath.Join(t.TempDir(), "service.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(svcJSON), 0644))

	_, stderr, err := runA7WithEnv(env, "service", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	var service map[string]interface{}
	runA7JSON(t, env, &service, "service", "get", svcID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, svcID, service["id"])
	plugins := requireJSONObject(t, service["plugins"], "service.plugins")
	assert.Contains(t, plugins, "proxy-rewrite")
}

func TestService_RouteWithServiceID(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-svc-ref"
	routeID := "e2e-route-svc-ref"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})

	createTestServiceViaCLI(t, env, svcID)

	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "route-svc-ref",
		"paths": ["/test-svc-ref"],
		"service_id": %q
	}`, routeID, svcID)
	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	_, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	var route map[string]interface{}
	runA7JSON(t, env, &route, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, routeID, route["id"])
	assert.Equal(t, svcID, route["service_id"])
}
