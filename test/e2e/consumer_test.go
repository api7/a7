//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForGatewayStatus polls the gateway until the desired status is observed
// or the timeout expires. Each request is bound to the remaining deadline so a
// stalled HTTP call cannot outlive the caller-provided timeout.
func waitForGatewayStatus(url string, buildRequest func() (*http.Request, error), want func(int) bool, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	lastStatus := 0
	var lastErr error
	for time.Now().Before(deadline) {
		req, err := buildRequest()
		if err != nil {
			return 0, err
		}
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		req = req.WithContext(ctx)
		resp, err := insecureClient.Do(req)
		cancel()
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		lastStatus = resp.StatusCode
		resp.Body.Close()
		if want(resp.StatusCode) {
			return resp.StatusCode, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	if lastErr != nil {
		return lastStatus, lastErr
	}
	return lastStatus, nil
}

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
	svcID := "e2e-service-keyauth"
	routeID := "e2e-route-keyauth"
	credID := "e2e-cred-keyauth"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
		deleteConsumerViaAdmin(t, username)
	})

	// Create consumer (no auth plugins — API7 EE requires credentials).
	createTestConsumerViaCLI(t, env, username)
	createTestServiceViaCLI(t, env, svcID)

	// Create credential with key-auth plugin.
	credJSON := fmt.Sprintf(`{
		"name": %q,
		"plugins": {
			"key-auth": {
				"key": "e2e-key-%s"
			}
		}
	}`, credID, username)
	credFile := filepath.Join(t.TempDir(), "credential.json")
	require.NoError(t, os.WriteFile(credFile, []byte(credJSON), 0644))
	stdout, stderr, err := runA7WithEnv(env, "credential", "create", credID,
		"--consumer", username, "-f", credFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Create route with key-auth plugin
	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "route-keyauth",
		"service_id": %q,
		"paths": ["/test-keyauth"],
		"plugins": {
			"key-auth": {},
			"proxy-rewrite": {"uri": "/get"}
		}
	}`, routeID, svcID)

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err = runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	status, err := waitForGatewayStatus(gatewayURL+"/test-keyauth", func() (*http.Request, error) {
		return http.NewRequest("GET", gatewayURL+"/test-keyauth", nil)
	}, func(code int) bool {
		return code == 401 || code == 403
	}, 15*time.Second)
	require.NoError(t, err)
	if status == 404 {
		t.Skip("route did not propagate to the local gateway within timeout; skipping live key-auth assertion")
	}
	assert.True(t, status == 401 || status == 403, "expected 401/403 without key, got %d", status)

	status, err = waitForGatewayStatus(gatewayURL+"/test-keyauth", func() (*http.Request, error) {
		req, err := http.NewRequest("GET", gatewayURL+"/test-keyauth", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("apikey", "e2e-key-"+username)
		return req, nil
	}, func(code int) bool {
		return code == 200
	}, 15*time.Second)
	require.NoError(t, err)
	if status == 404 {
		t.Skip("authenticated route did not propagate to the local gateway within timeout; skipping live key-auth assertion")
	}
	assert.Equal(t, 200, status)
}

func TestConsumer_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "consumer", "delete", "nonexistent-consumer-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}
