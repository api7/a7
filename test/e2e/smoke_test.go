//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmoke_BinaryRuns(t *testing.T) {
	stdout, stderr, err := runA7("--help")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "a7")
	assert.Contains(t, stdout, "API7")
}

func TestSmoke_API7Reachable(t *testing.T) {
	_ = setupEnv(t)

	resp, err := adminAPI(http.MethodGet, "/api/gateway_groups", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSmoke_GatewayReachable(t *testing.T) {
	env := setupEnv(t)

	_, stderr, err := runA7WithEnv(env, "route", "list", "-g", gatewayGroup)
	if err != nil {
		t.Logf("route list returned error (API7 EE may require service_id): %s", stderr)
	}

	resp, err := runtimeAdminAPI(http.MethodGet, "/apisix/admin/routes", nil)
	if err != nil {
		t.Skipf("runtime admin API not reachable: %v", err)
	}
	defer resp.Body.Close()

	// Accept 200 or 400 (API7 EE may require service_id for route listing).
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest,
		"expected 200 or 400, got %d", resp.StatusCode)
}
