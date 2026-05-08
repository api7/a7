//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requirePluginConfigCompatibilityError(t *testing.T, stdout, stderr string, err error) {
	t.Helper()
	require.Error(t, err)
	combined := strings.ToLower(stdout + "\n" + stderr)
	assert.Contains(t, combined, "apisix compatibility")
	assert.Contains(t, combined, "api7 ee admin api")
}

func TestPluginConfig_ListUnsupportedInAPI7EE(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "plugin-config", "list", "-g", gatewayGroup)
	requirePluginConfigCompatibilityError(t, stdout, stderr, err)
}

func TestPluginConfig_CreateUnsupportedInAPI7EE(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "plugin-config", "create",
		"--plugins-json", `{"key-auth":{}}`,
		"-g", gatewayGroup)
	requirePluginConfigCompatibilityError(t, stdout, stderr, err)
}
