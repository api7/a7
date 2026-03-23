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

// deleteStreamRouteViaAdmin deletes a stream route via the Admin API.
func deleteStreamRouteViaAdmin(t *testing.T, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/stream_routes/%s", id), nil)
	if err == nil {
		resp.Body.Close()
	}
}

func TestStreamRoute_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "stream-route", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestStreamRoute_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "stream-route", "list", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestStreamRoute_CRUD(t *testing.T) {
	// Stream routes may not be enabled in all API7 EE setups.
	env := setupEnv(t)
	srID := "e2e-stream-route-crud"
	t.Cleanup(func() { deleteStreamRouteViaAdmin(t, srID) })

	srJSON := fmt.Sprintf(`{
		"id": %q,
		"server_port": 19090,
		"upstream": {
			"type": "roundrobin",
			"nodes": {"%s": 1}
		}
	}`, srID, upstreamNode())

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
	stdout, stderr, err = runA7WithEnv(env, "stream-route", "get", srID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "19090")

	// Export
	stdout, stderr, err = runA7WithEnv(env, "stream-route", "export", srID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, srID)

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "stream-route", "delete", srID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
}

func TestStreamRoute_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "stream-route", "delete", "nonexistent-sr-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}
