//go:build e2e

package e2e

import (
	"encoding/json"
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
	require.NoError(t, err, "credential create failed")

	var created map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &created), "credential create should return JSON")
	actualID, ok := created["id"].(string)
	require.True(t, ok && actualID != "", "credential create response should contain generated id: %v", created)
	assert.Equal(t, credID, created["name"])
	t.Cleanup(func() {
		_, _, cleanupErr := runA7WithEnv(env, "credential", "delete", actualID,
			"--consumer", username, "--force", "-g", gatewayGroup)
		if cleanupErr != nil {
			t.Logf("credential cleanup failed for %s", actualID)
		}
	})

	var credential map[string]interface{}
	runA7JSON(t, env, &credential, "credential", "get", actualID,
		"--consumer", username, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, actualID, credential["id"])
	assert.Equal(t, credID, credential["name"])
	plugins := requireJSONObject(t, credential["plugins"], "credential.plugins")
	assert.Contains(t, plugins, "key-auth")
}

func TestCredential_UpdateFlagsPreserveExistingFields(t *testing.T) {
	env := setupEnv(t)
	username := uniqueResourceID("e2e-cred-update-consumer")
	credName := uniqueResourceID("e2e-cred-update")
	t.Cleanup(func() { deleteConsumerViaAdmin(t, username) })

	createTestConsumerViaCLI(t, env, username)

	stdout, _, err := runA7WithEnv(env, "credential", "create", credName,
		"--consumer", username,
		"--plugins-json", `{"key-auth":{"key":"e2e-update-key-12345"}}`,
		"--desc", "old credential desc",
		"-g", gatewayGroup)
	require.NoError(t, err, "credential create failed")

	var created map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &created), "credential create should return JSON")
	actualID, ok := created["id"].(string)
	require.True(t, ok && actualID != "", "credential create response should contain generated id: %v", created)
	t.Cleanup(func() {
		_, _, cleanupErr := runA7WithEnv(env, "credential", "delete", actualID,
			"--consumer", username, "--force", "-g", gatewayGroup)
		if cleanupErr != nil {
			t.Logf("credential cleanup failed for %s", actualID)
		}
	})

	stdout, _, err = runA7WithEnv(env, "credential", "update", actualID,
		"--consumer", username,
		"--desc", "updated credential desc",
		"-g", gatewayGroup,
		"-o", "json")
	require.NoError(t, err, "credential update failed")

	var credential map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &credential), "credential update should return JSON")
	assert.Equal(t, "updated credential desc", credential["desc"])
	assert.Equal(t, credName, credential["name"])
	plugins := requireJSONObject(t, credential["plugins"], "credential.plugins")
	assert.Contains(t, plugins, "key-auth")
	keyAuth := requireJSONObject(t, plugins["key-auth"], "credential.plugins.key-auth")
	assert.Equal(t, "e2e-update-key-12345", keyAuth["key"])

	runA7JSON(t, env, &credential, "credential", "get", actualID,
		"--consumer", username, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, "updated credential desc", credential["desc"])
	assert.Equal(t, credName, credential["name"])
	plugins = requireJSONObject(t, credential["plugins"], "credential.plugins")
	assert.Contains(t, plugins, "key-auth")
	keyAuth = requireJSONObject(t, plugins["key-auth"], "credential.plugins.key-auth")
	assert.Equal(t, "e2e-update-key-12345", keyAuth["key"])
}

func TestCredential_RequiresConsumerFlag(t *testing.T) {
	env := setupEnv(t)

	// Should fail without --consumer
	_, stderr, err := runA7WithEnv(env, "credential", "list", "-g", gatewayGroup)
	assert.Error(t, err)
	assert.Contains(t, stderr, "consumer")
}
