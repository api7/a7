//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
	var metadata map[string]interface{}
	runA7JSON(t, env, &metadata, "plugin-metadata", "get", pluginName, "-g", gatewayGroup, "-o", "json")
	logFormatValue := metadata["log_format"]
	if logFormatValue == nil {
		wrapped := requireJSONObject(t, metadata["metadata"], "plugin_metadata.metadata")
		logFormatValue = wrapped["log_format"]
	}
	logFormat := requireJSONObject(t, logFormatValue, "plugin_metadata.log_format")
	assert.Equal(t, "$remote_addr", logFormat["client_ip"])

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "plugin-metadata", "delete", pluginName, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	_, _, err = runA7WithEnv(env, "plugin-metadata", "get", pluginName, "-g", gatewayGroup)
	assert.Error(t, err)
}

// TestPluginMetadata_GetYAML guards against a regression where
// `plugin-metadata get -o yaml` serialized the raw response bytes as a YAML
// list of integers instead of the actual metadata map.
func TestPluginMetadata_GetYAML(t *testing.T) {
	env := setupEnv(t)
	pluginName := "http-logger"
	t.Cleanup(func() { deletePluginMetadataViaAdmin(t, pluginName) })

	pmJSON := `{"log_format":{"host":"$host"}}`
	tmpFile := filepath.Join(t.TempDir(), "plugin-metadata.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(pmJSON), 0644))
	_, stderr, err := runA7WithEnv(env, "plugin-metadata", "create", pluginName, "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	stdout, stderr, err := runA7WithEnv(env, "plugin-metadata", "get", pluginName, "-g", gatewayGroup, "-o", "yaml")
	require.NoError(t, err, stderr)

	var meta map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(stdout), &meta), "output should be a YAML map, got: %s", stdout)
	logFormat, ok := meta["log_format"].(map[string]interface{})
	require.True(t, ok, "expected log_format map in: %s", stdout)
	assert.Equal(t, "$host", logFormat["host"])
}
