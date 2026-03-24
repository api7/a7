//go:build e2e

package e2e

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
	t.Skip("plugin_config is not exposed via API7 EE Admin API")
}

func TestPluginConfig_ListJSON(t *testing.T) {
	t.Skip("plugin_config is not exposed via API7 EE Admin API")
}

func TestPluginConfig_CRUD(t *testing.T) {
	t.Skip("plugin_config is not exposed via API7 EE Admin API")
}

func TestPluginConfig_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "plugin-config", "delete", "nonexistent-pc-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}
