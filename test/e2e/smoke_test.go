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

	stdout, stderr, err := runA7WithEnv(env, "route", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)

	resp, err := runtimeAdminAPI(http.MethodGet, "/apisix/admin/routes", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
