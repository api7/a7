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

// deleteSecretViaAdmin deletes a secret provider via the Admin API.
func deleteSecretViaAdmin(t *testing.T, secretManager, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/secrets/%s/%s", secretManager, id), nil)
	if err == nil {
		resp.Body.Close()
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
	if err != nil {
		t.Skipf("secret create failed (vault may not be configured): %s %s", stdout, stderr)
	}

	// Get
	stdout, stderr, err = runA7WithEnv(env, "secret", "get", secretID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "vault")

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "secret", "get", secretID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "vault.example.com")

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "secret", "delete", secretID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
}

func TestSecret_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "secret", "delete", "vault/nonexistent-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}
