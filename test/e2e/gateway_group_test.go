//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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

// findDefaultGatewayGroupID uses the CLI to list gateway groups in JSON
// and returns the ID of the first group whose name contains "default".
// API7 EE uses UUID-style IDs, not names, so we need to resolve the real ID.
func findDefaultGatewayGroupID(t *testing.T, env []string) string {
	t.Helper()
	stdout, stderr, err := runA7WithEnv(env, "gateway-group", "list", "-o", "json")
	require.NoError(t, err, "list gateway groups failed: %s", stderr)

	var groups []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &groups), "should be valid JSON array")
	require.NotEmpty(t, groups, "no gateway groups found")

	for _, g := range groups {
		if id, ok := g["id"].(string); ok && id != "" {
			return id
		}
	}
	t.Fatal("no gateway group with a valid id found")
	return ""
}

func TestGatewayGroup_List(t *testing.T) {
	env := setupEnv(t)

	// Gateway groups use /api/gateway_groups — no -g flag needed.
	stdout, stderr, err := runA7WithEnv(env, "gateway-group", "list")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
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

	ggID := findDefaultGatewayGroupID(t, env)
	stdout, stderr, err := runA7WithEnv(env, "gateway-group", "get", ggID)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestGatewayGroup_GetJSON(t *testing.T) {
	env := setupEnv(t)

	ggID := findDefaultGatewayGroupID(t, env)
	stdout, stderr, err := runA7WithEnv(env, "gateway-group", "get", ggID, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)

	var group map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &group), "should be valid JSON")
	assert.Equal(t, ggID, group["id"])
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
	ggName := fmt.Sprintf("E2E Test GG %d", time.Now().UnixNano())

	// API7 EE generates UUIDs for gateway groups; custom IDs are not supported.
	ggJSON := fmt.Sprintf(`{
		"name": %q,
		"description": "Created by e2e tests"
	}`, ggName)

	tmpFile := filepath.Join(t.TempDir(), "gateway-group.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(ggJSON), 0644))

	// Create
	stdout, stderr, err := runA7WithEnv(env, "gateway-group", "create", "-f", tmpFile)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Parse created ID from response.
	var created map[string]interface{}
	var ggID string
	if json.Unmarshal([]byte(stdout), &created) == nil {
		if id, ok := created["id"]; ok {
			ggID = fmt.Sprintf("%v", id)
		}
	}
	if ggID == "" {
		t.Fatalf("failed to parse gateway group ID from create response: %s", stdout)
	}
	t.Cleanup(func() { deleteGatewayGroupViaAdmin(t, ggID) })

	// Get
	stdout, stderr, err = runA7WithEnv(env, "gateway-group", "get", ggID)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, ggName)

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "gateway-group", "get", ggID, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, ggName)

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "gateway-group", "delete", ggID, "--force")
	require.NoError(t, err, stderr)
}

func TestGatewayGroup_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "gateway-group", "delete", "nonexistent-gg-12345", "--force")
	assert.Error(t, err)
}
