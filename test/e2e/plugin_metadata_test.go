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

// deletePluginMetadataViaAdmin deletes plugin metadata via the Admin API.
func deletePluginMetadataViaAdmin(t *testing.T, pluginName string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/plugin_metadata/%s", pluginName), nil)
	if err == nil {
		resp.Body.Close()
	}
}

func TestPluginMetadata_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "plugin-metadata", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestPluginMetadata_CRUD(t *testing.T) {
	env := setupEnv(t)
	pluginName := "http-logger"
	t.Cleanup(func() { deletePluginMetadataViaAdmin(t, pluginName) })

	pmJSON := fmt.Sprintf(`{
		"log_format": {
			"host": "$host",
			"client_ip": "$remote_addr"
		}
	}`)

	tmpFile := filepath.Join(t.TempDir(), "plugin-metadata.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(pmJSON), 0644))

	// Create (plugin-metadata uses plugin name as identifier)
	stdout, stderr, err := runA7WithEnv(env, "plugin-metadata", "create", pluginName, "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Get
	stdout, stderr, err = runA7WithEnv(env, "plugin-metadata", "get", pluginName, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, pluginName)

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "plugin-metadata", "get", pluginName, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "log_format")

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "plugin-metadata", "delete", pluginName, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
}
