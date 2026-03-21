//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deleteGatewayGroupViaAdmin deletes a gateway group via the control-plane API.
func deleteGatewayGroupViaAdmin(t *testing.T, id string) {
	t.Helper()
	resp, err := adminAPI("DELETE", fmt.Sprintf("/api/gateway_groups/%s", id), nil)
	if err == nil {
		resp.Body.Close()
	}
}

func TestGatewayGroup_List(t *testing.T) {
	env := setupEnv(t)

	// Gateway groups use /api/gateway_groups — no -g flag needed.
	stdout, stderr, err := runA7WithEnv(env, "gateway-group", "list")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
	// The "default" gateway group should always exist.
	assert.Contains(t, stdout, "default")
}

func TestGatewayGroup_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "gateway-group", "list", "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)

	var groups []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &groups), "should be valid JSON array")
	assert.NotEmpty(t, groups)
}

func TestGatewayGroup_ListYAML(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "gateway-group", "list", "-o", "yaml")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestGatewayGroup_Get(t *testing.T) {
	env := setupEnv(t)

	// The "default" gateway group should exist in API7 EE.
	stdout, stderr, err := runA7WithEnv(env, "gateway-group", "get", "default")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "default")
}

func TestGatewayGroup_GetJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "gateway-group", "get", "default", "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)

	var group map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &group), "should be valid JSON")
	assert.Equal(t, "default", group["id"])
}

func TestGatewayGroup_GetNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "gateway-group", "get", "nonexistent-gg-12345")
	assert.Error(t, err)
}

func TestGatewayGroup_Alias(t *testing.T) {
	env := setupEnv(t)

	// Test the "gg" alias.
	stdout, stderr, err := runA7WithEnv(env, "gg", "list")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestGatewayGroup_CRUD(t *testing.T) {
	env := setupEnv(t)
	ggID := "e2e-gateway-group-crud"
	t.Cleanup(func() { deleteGatewayGroupViaAdmin(t, ggID) })

	ggJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "E2E Test Gateway Group",
		"description": "Created by e2e tests"
	}`, ggID)

	tmpFile := filepath.Join(t.TempDir(), "gateway-group.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(ggJSON), 0644))

	// Create
	stdout, stderr, err := runA7WithEnv(env, "gateway-group", "create", "-f", tmpFile)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Get
	stdout, stderr, err = runA7WithEnv(env, "gateway-group", "get", ggID)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, ggID)

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "gateway-group", "get", ggID, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "E2E Test Gateway Group")

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "gateway-group", "delete", ggID, "--force")
	require.NoError(t, err, stderr)
}

func TestGatewayGroup_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "gateway-group", "delete", "nonexistent-gg-12345", "--force")
	assert.Error(t, err)
}
