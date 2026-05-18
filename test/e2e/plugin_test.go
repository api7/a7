//go:build e2e

package e2e

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPlugin_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "plugin", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
	// Should contain well-known plugins.
	assert.Contains(t, stdout, "proxy-rewrite")
}

func TestPlugin_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "plugin", "list", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestPlugin_Get(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "plugin", "get", "proxy-rewrite", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestPlugin_GetJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "plugin", "get", "proxy-rewrite", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
	// The response is a JSON schema; verify it's valid JSON.
	var schema map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(stdout), &schema))
}

func TestPlugin_GetNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "plugin", "get", "nonexistent-plugin-12345", "-g", gatewayGroup)
	assert.Error(t, err)
}

// TestPlugin_GetYAML guards against a regression where `plugin get -o yaml`
// ignored the requested format and emitted JSON instead.
func TestPlugin_GetYAML(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "plugin", "get", "proxy-rewrite", "-g", gatewayGroup, "-o", "yaml")
	require.NoError(t, err, stderr)
	assert.False(t, strings.HasPrefix(strings.TrimSpace(stdout), "{"), "yaml output must not be JSON: %s", stdout)
	var schema map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(stdout), &schema), "output should be valid YAML")
	assert.NotEmpty(t, schema)
}
