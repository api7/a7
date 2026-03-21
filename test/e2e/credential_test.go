//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredential_List(t *testing.T) {
	env := setupEnv(t)
	username := "e2e-cred-consumer"
	t.Cleanup(func() { deleteConsumerViaAdmin(t, username) })

	// Create consumer first
	createTestConsumerViaCLI(t, env, username)

	stdout, stderr, err := runA7WithEnv(env, "credential", "list", "--consumer", username, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestCredential_ListJSON(t *testing.T) {
	env := setupEnv(t)
	username := "e2e-cred-consumer-json"
	t.Cleanup(func() { deleteConsumerViaAdmin(t, username) })

	createTestConsumerViaCLI(t, env, username)

	stdout, stderr, err := runA7WithEnv(env, "credential", "list", "--consumer", username, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestCredential_CRUD(t *testing.T) {
	env := setupEnv(t)
	username := "e2e-cred-crud-consumer"
	credID := "e2e-cred-crud"
	t.Cleanup(func() { deleteConsumerViaAdmin(t, username) })

	// Create consumer
	createTestConsumerViaCLI(t, env, username)

	// Create credential
	credJSON := `{
		"plugins": {
			"key-auth": {
				"key": "e2e-cred-key-12345"
			}
		}
	}`
	tmpFile := filepath.Join(t.TempDir(), "credential.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(credJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "credential", "create", credID,
		"--consumer", username, "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Get
	stdout, stderr, err = runA7WithEnv(env, "credential", "get", credID,
		"--consumer", username, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, credID)

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "credential", "get", credID,
		"--consumer", username, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "key-auth")

	// Delete credential
	stdout, stderr, err = runA7WithEnv(env, "credential", "delete", credID,
		"--consumer", username, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
}

func TestCredential_RequiresConsumerFlag(t *testing.T) {
	env := setupEnv(t)

	// Should fail without --consumer
	_, stderr, err := runA7WithEnv(env, "credential", "list", "-g", gatewayGroup)
	assert.Error(t, err)
	assert.Contains(t, stderr, "consumer")
}
