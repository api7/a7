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

	stdout, stderr, err := runA7WithEnv(env, "stream-route", "list", "-g", gatewayGroup)
	if err != nil {
		t.Skipf("stream-route list failed (may not be enabled): stderr=%s", stderr)
	}
	assert.NotEmpty(t, stdout)
}

func TestStreamRoute_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "stream-route", "list", "-g", gatewayGroup, "-o", "json")
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
	assert.Equal(t, svcID, streamRoute["service_id"])
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
