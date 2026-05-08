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
		"name": "e2e-cred-crud",
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
	require.NoError(t, err, "credential create failed")

	// Get
	stdout, stderr, err = runA7WithEnv(env, "credential", "get", credID,
		"--consumer", username, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, credID)

	// Get JSON
	var credential map[string]interface{}
	runA7JSON(t, env, &credential, "credential", "get", credID,
		"--consumer", username, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, credID, credential["id"])
	plugins := requireJSONObject(t, credential["plugins"], "credential.plugins")
	assert.Contains(t, plugins, "key-auth")

	// Delete credential
	stdout, stderr, err = runA7WithEnv(env, "credential", "delete", credID,
		"--consumer", username, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	_, _, err = runA7WithEnv(env, "credential", "get", credID,
		"--consumer", username, "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestCredential_CreateWithPositionalID(t *testing.T) {
	env := setupEnv(t)
	username := "e2e-cred-positional-consumer"
	credID := "e2e-cred-positional"
	t.Cleanup(func() { deleteConsumerViaAdmin(t, username) })

	createTestConsumerViaCLI(t, env, username)

	stdout, stderr, err := runA7WithEnv(env, "credential", "create", credID,
		"--consumer", username,
		"--plugins-json", `{"key-auth":{"key":"e2e-positional-key-12345"}}`,
		"-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	var credential map[string]interface{}
	runA7JSON(t, env, &credential, "credential", "get", credID,
		"--consumer", username, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, credID, credential["id"])
	assert.Equal(t, credID, credential["name"])
	plugins := requireJSONObject(t, credential["plugins"], "credential.plugins")
	assert.Contains(t, plugins, "key-auth")

	stdout, stderr, err = runA7WithEnv(env, "credential", "delete", credID,
		"--consumer", username, "--force", "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
}

func TestCredential_RequiresConsumerFlag(t *testing.T) {
	env := setupEnv(t)

	// Should fail without --consumer
	_, stderr, err := runA7WithEnv(env, "credential", "list", "-g", gatewayGroup)
	assert.Error(t, err)
	assert.Contains(t, stderr, "consumer")
}
