//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deleteUpstreamViaCLI deletes an upstream using the a7 CLI.
func deleteUpstreamViaCLI(t *testing.T, env []string, id string) {
	t.Helper()
	_, _, _ = runA7WithEnv(env, "upstream", "delete", id, "--force", "-g", gatewayGroup)
}

// deleteUpstreamViaAdmin deletes an upstream via the Admin API (cleanup).
func deleteUpstreamViaAdmin(t *testing.T, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/upstreams/%s", id), nil)
	if err == nil {
		resp.Body.Close()
	}
}

// createTestUpstreamViaCLI creates an upstream via CLI.
func createTestUpstreamViaCLI(t *testing.T, env []string, id string) {
	t.Helper()
	upstreamJSON := fmt.Sprintf(`{
		"id": %q,
		"type": "roundrobin",
		"nodes": {"%s": 1}
	}`, id, strings.TrimPrefix(httpbinURL, "http://"))

	tmpFile := filepath.Join(t.TempDir(), "upstream.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(upstreamJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "upstream", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
}

func TestUpstream_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "upstream", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestUpstream_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "upstream", "list", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestUpstream_CRUD(t *testing.T) {
	env := setupEnv(t)
	usID := "e2e-upstream-crud"
	t.Cleanup(func() { deleteUpstreamViaAdmin(t, usID) })

	// Create
	createTestUpstreamViaCLI(t, env, usID)

	// Get
	stdout, stderr, err := runA7WithEnv(env, "upstream", "get", usID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, usID)

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "upstream", "get", usID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, usID)

	// Update
	updateJSON := fmt.Sprintf(`{
		"id": %q,
		"type": "roundrobin",
		"nodes": {"%s": 2}
	}`, usID, strings.TrimPrefix(httpbinURL, "http://"))
	tmpFile := filepath.Join(t.TempDir(), "upstream-update.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(updateJSON), 0644))

	stdout, stderr, err = runA7WithEnv(env, "upstream", "update", usID, "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "upstream", "delete", usID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
}

func TestUpstream_MultiNode(t *testing.T) {
	env := setupEnv(t)
	usID := "e2e-upstream-multi"
	t.Cleanup(func() { deleteUpstreamViaAdmin(t, usID) })

	upstreamJSON := fmt.Sprintf(`{
		"id": %q,
		"type": "roundrobin",
		"nodes": {
			"%s": 3,
			"127.0.0.1:9999": 1
		}
	}`, usID, strings.TrimPrefix(httpbinURL, "http://"))

	tmpFile := filepath.Join(t.TempDir(), "upstream.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(upstreamJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "upstream", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Verify via get
	stdout, stderr, err = runA7WithEnv(env, "upstream", "get", usID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "roundrobin")
}

func TestUpstream_Export(t *testing.T) {
	env := setupEnv(t)
	usID := "e2e-upstream-export"
	t.Cleanup(func() { deleteUpstreamViaAdmin(t, usID) })

	createTestUpstreamViaCLI(t, env, usID)

	stdout, stderr, err := runA7WithEnv(env, "upstream", "export", usID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, usID)
}

func TestUpstream_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "upstream", "delete", "nonexistent-upstream-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestUpstream_RouteWithUpstreamID(t *testing.T) {
	env := setupEnv(t)
	usID := "e2e-us-ref"
	routeID := "e2e-route-us-ref"
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteUpstreamViaAdmin(t, usID)
	})

	// Create upstream
	createTestUpstreamViaCLI(t, env, usID)

	// Create route referencing upstream
	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"uri": "/test-us-ref",
		"upstream_id": %q
	}`, routeID, usID)

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Verify route references the upstream
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, usID)
}

func TestUpstream_ListWithLabel(t *testing.T) {
	env := setupEnv(t)
	usID := "e2e-upstream-label"
	t.Cleanup(func() { deleteUpstreamViaAdmin(t, usID) })

	upstreamJSON := fmt.Sprintf(`{
		"id": %q,
		"type": "roundrobin",
		"nodes": {"%s": 1},
		"labels": {"env": "e2e-test"}
	}`, usID, strings.TrimPrefix(httpbinURL, "http://"))

	tmpFile := filepath.Join(t.TempDir(), "upstream.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(upstreamJSON), 0644))

	_, stderr, err := runA7WithEnv(env, "upstream", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	stdout, stderr, err := runA7WithEnv(env, "upstream", "list", "-g", gatewayGroup, "--label", "env=e2e-test")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, usID)
}
