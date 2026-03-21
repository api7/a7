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

// deletePluginConfigViaAdmin deletes a plugin config via the Admin API.
func deletePluginConfigViaAdmin(t *testing.T, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/plugin_configs/%s", id), nil)
	if err == nil {
		resp.Body.Close()
	}
}

func TestPluginConfig_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "plugin-config", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestPluginConfig_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "plugin-config", "list", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestPluginConfig_CRUD(t *testing.T) {
	env := setupEnv(t)
	pcID := "e2e-plugin-config-crud"
	t.Cleanup(func() { deletePluginConfigViaAdmin(t, pcID) })

	pcJSON := fmt.Sprintf(`{
		"id": %q,
		"plugins": {
			"proxy-rewrite": {
				"uri": "/get"
			}
		}
	}`, pcID)

	tmpFile := filepath.Join(t.TempDir(), "plugin-config.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(pcJSON), 0644))

	// Create
	stdout, stderr, err := runA7WithEnv(env, "plugin-config", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Get
	stdout, stderr, err = runA7WithEnv(env, "plugin-config", "get", pcID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, pcID)

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "plugin-config", "get", pcID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "proxy-rewrite")

	// Update
	updateJSON := fmt.Sprintf(`{
		"id": %q,
		"plugins": {
			"proxy-rewrite": {
				"uri": "/anything"
			}
		}
	}`, pcID)
	tmpFile2 := filepath.Join(t.TempDir(), "plugin-config-update.json")
	require.NoError(t, os.WriteFile(tmpFile2, []byte(updateJSON), 0644))

	stdout, stderr, err = runA7WithEnv(env, "plugin-config", "update", pcID, "-f", tmpFile2, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	// Export
	stdout, stderr, err = runA7WithEnv(env, "plugin-config", "export", pcID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, pcID)

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "plugin-config", "delete", pcID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
}

func TestPluginConfig_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "plugin-config", "delete", "nonexistent-pc-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}
