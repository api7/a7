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

	// Update credential and verify readback.
	updateJSON := `{
		"id": "e2e-cred-crud",
		"desc": "updated credential",
		"plugins": {
			"key-auth": {
				"key": "e2e-cred-key-updated"
			}
		}
	}`
	updateFile := filepath.Join(t.TempDir(), "credential-update.json")
	require.NoError(t, os.WriteFile(updateFile, []byte(updateJSON), 0644))
	stdout, stderr, err = runA7WithEnv(env, "credential", "update", credID,
		"--consumer", username, "-f", updateFile, "-g", gatewayGroup)
	require.NoError(t, err, "credential update failed")

	runA7JSON(t, env, &credential, "credential", "get", credID,
		"--consumer", username, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, "updated credential", credential["desc"])
	plugins = requireJSONObject(t, credential["plugins"], "credential.plugins")
	assert.Contains(t, plugins, "key-auth")

	// Delete credential
	stdout, stderr, err = runA7WithEnv(env, "credential", "delete", credID,
		"--consumer", username, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	_, _, err = runA7WithEnv(env, "credential", "get", credID,
		"--consumer", username, "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestCredential_RequiresConsumerFlag(t *testing.T) {
	env := setupEnv(t)

	// Should fail without --consumer
	_, stderr, err := runA7WithEnv(env, "credential", "list", "-g", gatewayGroup)
	assert.Error(t, err)
	assert.Contains(t, stderr, "consumer")
}
