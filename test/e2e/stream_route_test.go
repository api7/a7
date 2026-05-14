//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deleteStreamRouteViaAdmin deletes a stream route via the Admin API.
func deleteStreamRouteViaAdmin(t testTB, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/stream_routes/%s", id), nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("delete stream route %s via admin API failed: %v", id, err)
	}
	if resp.StatusCode == 404 {
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("delete stream route %s via admin API returned %d: %s", id, resp.StatusCode, string(body))
	}
}

func TestStreamRoute_List(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-stream-route-list-svc"
	t.Cleanup(func() { deleteServiceViaAdmin(t, svcID) })
	createTestServiceViaCLI(t, env, svcID)

	stdout, stderr, err := runA7WithEnv(env, "stream-route", "list", "-g", gatewayGroup, "--service-id", svcID)
	if err != nil {
		t.Skipf("stream-route list failed (may not be enabled): stderr=%s", stderr)
	}
	assert.NotEmpty(t, stdout)
}

func TestStreamRoute_ListJSON(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-stream-route-list-json-svc"
	t.Cleanup(func() { deleteServiceViaAdmin(t, svcID) })
	createTestServiceViaCLI(t, env, svcID)

	stdout, stderr, err := runA7WithEnv(env, "stream-route", "list", "-g", gatewayGroup, "--service-id", svcID, "-o", "json")
	if err != nil {
		t.Skipf("stream-route list JSON failed (may not be enabled): stderr=%s", stderr)
	}
	assert.NotEmpty(t, stdout)
}

func TestStreamRoute_CRUD(t *testing.T) {
	// Stream routes may not be enabled in all API7 EE setups.
	env := setupEnv(t)
	svcID := "e2e-stream-route-svc"
	srID := "e2e-stream-route-crud"
	t.Cleanup(func() {
		deleteStreamRouteViaAdmin(t, srID)
		deleteServiceViaAdmin(t, svcID)
	})

	createTestServiceViaCLI(t, env, svcID)

	srJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "e2e-stream-route-crud",
		"service_id": %q,
		"server_port": 19090,
		"desc": "stream route e2e"
	}`, srID, svcID)

	tmpFile := filepath.Join(t.TempDir(), "stream-route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(srJSON), 0644))

	// Create
	stdout, stderr, err := runA7WithEnv(env, "stream-route", "create", "-f", tmpFile, "-g", gatewayGroup)
	if err != nil {
		t.Skipf("stream-route create failed (may not be enabled): %s %s", stdout, stderr)
	}

	// Get
	stdout, stderr, err = runA7WithEnv(env, "stream-route", "get", srID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, srID)

	// Get JSON
	var streamRoute map[string]interface{}
	runA7JSON(t, env, &streamRoute, "stream-route", "get", srID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, srID, streamRoute["id"])
	assert.Equal(t, float64(19090), streamRoute["server_port"])
	assert.Equal(t, "stream route e2e", streamRoute["desc"])

	// Update and verify readback.
	updateJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "e2e-stream-route-updated",
		"service_id": %q,
		"server_port": 19091,
		"desc": "stream route e2e updated"
	}`, srID, svcID)
	updateFile := filepath.Join(t.TempDir(), "stream-route-update.json")
	require.NoError(t, os.WriteFile(updateFile, []byte(updateJSON), 0644))
	stdout, stderr, err = runA7WithEnv(env, "stream-route", "update", srID, "-f", updateFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	runA7JSON(t, env, &streamRoute, "stream-route", "get", srID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, float64(19091), streamRoute["server_port"])
	assert.Equal(t, "stream route e2e updated", streamRoute["desc"])

	var exported []map[string]interface{}
	runA7JSON(t, env, &exported, "stream-route", "export", "-g", gatewayGroup, "--service-id", svcID, "-o", "json")
	found := false
	for _, item := range exported {
		if item["id"] == srID {
			found = true
			assert.Equal(t, svcID, item["service_id"])
		}
	}
	assert.True(t, found, "expected exported stream routes to contain %s", srID)

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "stream-route", "delete", srID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	_, _, err = runA7WithEnv(env, "stream-route", "get", srID, "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestStreamRoute_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "stream-route", "delete", "nonexistent-sr-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}

// TestStreamRoute_CreateRequiresName guards against the gap where flag-based
// `stream-route create` had no --name flag, while API7 EE requires `name`.
func TestStreamRoute_CreateRequiresName(t *testing.T) {
	env := setupEnv(t)

	_, stderr, err := runA7WithEnv(env, "stream-route", "create",
		"--service-id", "any-service-id", "--server-port", "19099", "-g", gatewayGroup)
	require.Error(t, err, "stream-route create without --name must error")
	assert.Contains(t, stderr, "name is required")
}

// createStreamServiceViaCLI creates a `type: stream` service; stream routes
// can only be attached to stream-typed services in API7 EE.
func createStreamServiceViaCLI(t testTB, env []string, id string) {
	t.Helper()
	svcJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "e2e-stream-svc-%s",
		"type": "stream",
		"upstream": {
			"type": "roundrobin",
			"nodes": [{"host": "127.0.0.1", "port": 3306, "weight": 1}]
		}
	}`, id, id)
	tmpFile := filepath.Join(t.TempDir(), "stream-service.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(svcJSON), 0644))
	stdout, stderr, err := runA7WithEnv(env, "service", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
}

// TestStreamRoute_CreateWithFlags exercises the flag-based create path,
// including the --name flag added to satisfy API7 EE's required field.
func TestStreamRoute_CreateWithFlags(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-stream-route-flags-svc"
	t.Cleanup(func() { deleteServiceViaAdmin(t, svcID) })
	createStreamServiceViaCLI(t, env, svcID)

	srName := "e2e-stream-route-flags"
	stdout, stderr, err := runA7WithEnv(env, "stream-route", "create",
		"--name", srName, "--service-id", svcID, "--server-port", "19098", "-g", gatewayGroup)
	if err != nil {
		t.Skipf("stream-route create failed (may not be enabled): %s %s", stdout, stderr)
	}

	var created map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &created), "create should return JSON: %s", stdout)
	assert.Equal(t, srName, created["name"])
	srID, ok := created["id"].(string)
	require.True(t, ok && srID != "", "create response should contain an id: %v", created)
	t.Cleanup(func() { deleteStreamRouteViaAdmin(t, srID) })

	var got map[string]interface{}
	runA7JSON(t, env, &got, "stream-route", "get", srID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, srName, got["name"])
	assert.Equal(t, float64(19098), got["server_port"])
}
