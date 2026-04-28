//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deleteSecretViaAdmin deletes a secret provider via the Admin API.
func deleteSecretViaAdmin(t testTB, secretManager, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/secret_providers/%s/%s", secretManager, id), nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("delete secret provider %s/%s via admin API failed: %v", secretManager, id, err)
	}
	if resp.StatusCode == 404 {
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("delete secret provider %s/%s via admin API returned %d: %s", secretManager, id, resp.StatusCode, string(body))
	}
}

func TestSecret_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "secret", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestSecret_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "secret", "list", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestSecret_CRUD(t *testing.T) {
	env := setupEnv(t)
	// Secret IDs use format: manager/id (e.g., vault/test-secret)
	secretID := "vault/e2e-secret-crud"
	t.Cleanup(func() { deleteSecretViaAdmin(t, "vault", "e2e-secret-crud") })

	secretJSON := fmt.Sprintf(`{
		"uri": "https://vault.example.com",
		"prefix": "kv/apisix",
		"token": "test-vault-token"
	}`)

	tmpFile := filepath.Join(t.TempDir(), "secret.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(secretJSON), 0644))

	// Create
	stdout, stderr, err := runA7WithEnv(env, "secret", "create", secretID, "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "secret create failed: stdout=%s stderr=%s", stdout, stderr)

	// Get
	stdout, stderr, err = runA7WithEnv(env, "secret", "get", secretID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "vault")

	// Get JSON
	var secret map[string]interface{}
	runA7JSON(t, env, &secret, "secret", "get", secretID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, secretID, secret["id"])
	assert.Equal(t, "https://vault.example.com", secret["uri"])
	assert.Equal(t, "kv/apisix", secret["prefix"])

	// Update and verify readback.
	updateJSON := `{
		"uri": "https://vault-updated.example.com",
		"prefix": "kv/apisix-updated",
		"token": "updated-vault-token"
	}`
	updateFile := filepath.Join(t.TempDir(), "secret-update.json")
	require.NoError(t, os.WriteFile(updateFile, []byte(updateJSON), 0644))
	stdout, stderr, err = runA7WithEnv(env, "secret", "update", secretID, "-f", updateFile, "-g", gatewayGroup)
	require.NoError(t, err, "secret update failed")

	runA7JSON(t, env, &secret, "secret", "get", secretID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, "https://vault-updated.example.com", secret["uri"])
	assert.Equal(t, "kv/apisix-updated", secret["prefix"])

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "secret", "delete", secretID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	_, _, err = runA7WithEnv(env, "secret", "get", secretID, "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestSecret_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "secret", "delete", "vault/nonexistent-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}
