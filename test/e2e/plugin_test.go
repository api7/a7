//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Contains(t, stdout, "proxy-rewrite")
}

func TestPlugin_GetJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "plugin", "get", "proxy-rewrite", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "proxy-rewrite")
}

func TestPlugin_GetNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "plugin", "get", "nonexistent-plugin-12345", "-g", gatewayGroup)
	assert.Error(t, err)
}
