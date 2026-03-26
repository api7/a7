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
	}`, id, upstreamNode())

	tmpFile := filepath.Join(t.TempDir(), "upstream.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(upstreamJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "upstream", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
}

func TestUpstream_List(t *testing.T) {
	t.Skip("standalone upstreams are not exposed via API7 EE Admin API; upstreams exist only as inline objects within services")
}

func TestUpstream_ListJSON(t *testing.T) {
	t.Skip("standalone upstreams are not exposed via API7 EE Admin API; upstreams exist only as inline objects within services")
}

func TestUpstream_CRUD(t *testing.T) {
	t.Skip("standalone upstreams are not exposed via API7 EE Admin API; upstreams exist only as inline objects within services")
}

func TestUpstream_MultiNode(t *testing.T) {
	t.Skip("standalone upstreams are not exposed via API7 EE Admin API; upstreams exist only as inline objects within services")
}

func TestUpstream_Export(t *testing.T) {
	t.Skip("standalone upstreams are not exposed via API7 EE Admin API; upstreams exist only as inline objects within services")
}

func TestUpstream_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "upstream", "delete", "nonexistent-upstream-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestUpstream_RouteWithUpstreamID(t *testing.T) {
	t.Skip("standalone upstreams are not exposed via API7 EE Admin API; upstreams exist only as inline objects within services")
}

func TestUpstream_ListWithLabel(t *testing.T) {
	t.Skip("standalone upstreams are not exposed via API7 EE Admin API; upstreams exist only as inline objects within services")
}
