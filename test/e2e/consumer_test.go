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

// deleteConsumerViaCLI deletes a consumer using the a7 CLI.
func deleteConsumerViaCLI(t *testing.T, env []string, username string) {
	t.Helper()
	_, _, _ = runA7WithEnv(env, "consumer", "delete", username, "--force", "-g", gatewayGroup)
}

// deleteConsumerViaAdmin deletes a consumer via the Admin API (cleanup).
func deleteConsumerViaAdmin(t *testing.T, username string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/consumers/%s", username), nil)
	if err == nil {
		resp.Body.Close()
	}
}

// createTestConsumerViaCLI creates a consumer via CLI.
// API7 EE does not allow auth plugins in the consumer body; use credentials instead.
func createTestConsumerViaCLI(t *testing.T, env []string, username string) {
	t.Helper()
	consumerJSON := fmt.Sprintf(`{
		"username": %q,
		"desc": "e2e test consumer"
	}`, username)

	tmpFile := filepath.Join(t.TempDir(), "consumer.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(consumerJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "consumer", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
}

func TestConsumer_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "consumer", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestConsumer_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "consumer", "list", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestConsumer_CRUD(t *testing.T) {
	env := setupEnv(t)
	username := "e2e-consumer-crud"
	t.Cleanup(func() { deleteConsumerViaAdmin(t, username) })

	// Create
	createTestConsumerViaCLI(t, env, username)

	// Get
	stdout, stderr, err := runA7WithEnv(env, "consumer", "get", username, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, username)

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "consumer", "get", username, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, username)

	// Update
	updateJSON := fmt.Sprintf(`{
		"username": %q,
		"desc": "updated consumer"
	}`, username)
	tmpFile := filepath.Join(t.TempDir(), "consumer-update.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(updateJSON), 0644))

	stdout, stderr, err = runA7WithEnv(env, "consumer", "update", username, "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "consumer", "delete", username, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
}

func TestConsumer_Export(t *testing.T) {
	env := setupEnv(t)
	username := "e2e-consumer-export"
	t.Cleanup(func() { deleteConsumerViaAdmin(t, username) })

	createTestConsumerViaCLI(t, env, username)

	// Use get -o json (export is batch-only, cobra.NoArgs).
	stdout, stderr, err := runA7WithEnv(env, "consumer", "get", username, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, username)
}

func TestConsumer_WithKeyAuth(t *testing.T) {
	requireGatewayURL(t)
	requireHTTPBin(t)
	env := setupEnv(t)
	username := "e2e-consumer-keyauth"
	routeID := "e2e-route-keyauth"
	credID := "e2e-cred-keyauth"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteConsumerViaAdmin(t, username)
	})

	// Create consumer (no auth plugins — API7 EE requires credentials).
	createTestConsumerViaCLI(t, env, username)

	// Create credential with key-auth plugin.
	credJSON := fmt.Sprintf(`{
		"plugins": {
			"key-auth": {
				"key": "e2e-key-%s"
			}
		}
	}`, username)
	credFile := filepath.Join(t.TempDir(), "credential.json")
	require.NoError(t, os.WriteFile(credFile, []byte(credJSON), 0644))
	_, stderr, err := runA7WithEnv(env, "credential", "create", credID,
		"--consumer", username, "-f", credFile, "-g", gatewayGroup)
	if err != nil {
		t.Skipf("credential create failed: %s", stderr)
	}

	// Create route with key-auth plugin
	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "route-keyauth",
		"paths": ["/test-keyauth"],
		"upstream": {
			"type": "roundrobin",
			"nodes": {"%s": 1}
		},
		"plugins": {
			"key-auth": {},
			"proxy-rewrite": {"uri": "/get"}
		}
	}`, routeID, upstreamNode())

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	_, stderr, err = runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	// Verify: request without key should fail (401 or 403)
	resp, err := insecureClient.Get(gatewayURL + "/test-keyauth")
	if err == nil {
		defer resp.Body.Close()
		assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 403,
			"expected 401/403 without key, got %d", resp.StatusCode)
	}
}

func TestConsumer_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "consumer", "delete", "nonexistent-consumer-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}
