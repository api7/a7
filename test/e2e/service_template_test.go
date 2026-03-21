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

// deleteServiceTemplateViaAdmin deletes a service template via the control-plane API.
func deleteServiceTemplateViaAdmin(t *testing.T, id string) {
	t.Helper()
	resp, err := adminAPI("DELETE", fmt.Sprintf("/api/services/template/%s", id), nil)
	if err == nil {
		resp.Body.Close()
	}
}

// createTestServiceTemplateViaCLI creates a service template via CLI and returns its ID.
func createTestServiceTemplateViaCLI(t *testing.T, env []string, id string) string {
	t.Helper()
	stJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "e2e-template-%s",
		"description": "Created by e2e tests",
		"upstream": {
			"type": "roundrobin",
			"nodes": {
				"127.0.0.1:8080": 1
			}
		}
	}`, id, id)

	tmpFile := filepath.Join(t.TempDir(), "service-template.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(stJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "service-template", "create", "-f", tmpFile)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	return id
}

func TestServiceTemplate_List(t *testing.T) {
	env := setupEnv(t)

	// Service templates use /api/services/template — no -g flag.
	stdout, stderr, err := runA7WithEnv(env, "service-template", "list")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestServiceTemplate_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "service-template", "list", "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestServiceTemplate_Alias(t *testing.T) {
	env := setupEnv(t)

	// Test the "st" alias.
	stdout, stderr, err := runA7WithEnv(env, "st", "list")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestServiceTemplate_CRUD(t *testing.T) {
	env := setupEnv(t)
	stID := "e2e-service-template-crud"
	t.Cleanup(func() { deleteServiceTemplateViaAdmin(t, stID) })

	// Create
	createTestServiceTemplateViaCLI(t, env, stID)

	// Get
	stdout, stderr, err := runA7WithEnv(env, "service-template", "get", stID)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, stID)

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "service-template", "get", stID, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "e2e-template-"+stID)

	// Update via file
	updateJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "e2e-template-updated",
		"description": "Updated by e2e tests",
		"upstream": {
			"type": "roundrobin",
			"nodes": {
				"127.0.0.1:8080": 1
			}
		}
	}`, stID)
	tmpFile := filepath.Join(t.TempDir(), "service-template-update.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(updateJSON), 0644))

	stdout, stderr, err = runA7WithEnv(env, "service-template", "update", stID, "-f", tmpFile)
	require.NoError(t, err, stderr)

	// Verify update
	stdout, stderr, err = runA7WithEnv(env, "service-template", "get", stID, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "e2e-template-updated")

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "service-template", "delete", stID, "--force")
	require.NoError(t, err, stderr)
}

func TestServiceTemplate_CreateWithName(t *testing.T) {
	env := setupEnv(t)
	// When using --name flag (no -f), the API auto-generates the ID.
	// We need to capture the ID from the JSON response to clean up.
	stdout, stderr, err := runA7WithEnv(env, "service-template", "create", "--name", "e2e-named-template")
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Parse ID from response for cleanup.
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &resp); err == nil {
		if id, ok := resp["id"]; ok {
			t.Cleanup(func() { deleteServiceTemplateViaAdmin(t, fmt.Sprintf("%v", id)) })
		}
	}
}

func TestServiceTemplate_Publish(t *testing.T) {
	env := setupEnv(t)
	stID := "e2e-service-template-publish"
	t.Cleanup(func() { deleteServiceTemplateViaAdmin(t, stID) })

	createTestServiceTemplateViaCLI(t, env, stID)

	// Publish to the default gateway group.
	stdout, stderr, err := runA7WithEnv(env, "service-template", "publish", stID,
		"--gateway-group-id", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
}

func TestServiceTemplate_PublishMissingFlag(t *testing.T) {
	env := setupEnv(t)

	// publish without --gateway-group-id should fail.
	_, _, err := runA7WithEnv(env, "service-template", "publish", "some-id")
	assert.Error(t, err)
}

func TestServiceTemplate_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "service-template", "delete", "nonexistent-st-12345", "--force")
	assert.Error(t, err)
}

func TestServiceTemplate_GetNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "service-template", "get", "nonexistent-st-12345")
	assert.Error(t, err)
}
