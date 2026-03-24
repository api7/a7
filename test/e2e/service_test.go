//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deleteServiceViaCLI deletes a service using the a7 CLI.
func deleteServiceViaCLI(t *testing.T, env []string, id string) {
	t.Helper()
	_, _, _ = runA7WithEnv(env, "service", "delete", id, "--force", "-g", gatewayGroup)
}

// deleteServiceViaAdmin deletes a service via the Admin API (cleanup).
func deleteServiceViaAdmin(t *testing.T, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/services/%s", id), nil)
	if err == nil {
		resp.Body.Close()
	}
}

// createTestServiceViaCLI creates a service via CLI.
func createTestServiceViaCLI(t *testing.T, env []string, id string) {
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
	stdout, stderr, err = runA7WithEnv(env, "service", "get", svcID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, svcID)

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
	stdout, stderr, err = runA7WithEnv(env, "service", "get", svcID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "e2e-svc-updated")

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "service", "delete", svcID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
}

func TestService_Export(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-service-export"
	t.Cleanup(func() { deleteServiceViaAdmin(t, svcID) })

	createTestServiceViaCLI(t, env, svcID)

	stdout, stderr, err := runA7WithEnv(env, "service", "export", svcID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, svcID)
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

	stdout, stderr, err := runA7WithEnv(env, "service", "get", svcID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "proxy-rewrite")
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

	stdout, stderr, err := runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, svcID)
}
