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

func TestConfigSync_DryRun(t *testing.T) {
	env := setupEnv(t)

	svcID := "e2e-sync-dryrun-svc"
	createTestServiceViaCLI(t, env, svcID)
	t.Cleanup(func() { deleteServiceViaAdmin(t, svcID) })

	syncYAML := fmt.Sprintf(`version: "1"
routes:
  - id: e2e-sync-dryrun-route
    name: e2e-sync-dryrun-route
    service_id: %s
    paths:
      - /sync-dryrun
    upstream:
      type: roundrobin
      nodes:
        - host: "127.0.0.1"
          port: 8080
          weight: 1
`, svcID)
	tmpFile := filepath.Join(t.TempDir(), "sync-dryrun.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(syncYAML), 0644))

	stdout, stderr, err := runA7WithEnv(env, "config", "sync", "-f", tmpFile, "--dry-run", "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	_, _, getErr := runA7WithEnv(env, "route", "get", "e2e-sync-dryrun-route", "-g", gatewayGroup)
	assert.Error(t, getErr, "dry-run should not create resources")
}

func TestConfigSync_CreateAndCleanup(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-sync-create-svc"
	routeID := "e2e-sync-create-route"
	createTestServiceViaCLI(t, env, svcID)
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})

	minimalYAML := fmt.Sprintf(`version: "1"
routes:
  - id: %s
    name: %s
    service_id: %s
    paths:
      - /sync-create-test
    upstream:
      type: roundrobin
      nodes:
        - host: "127.0.0.1"
          port: 8080
          weight: 1
`, routeID, routeID, svcID)

	syncFile := filepath.Join(t.TempDir(), "sync-create.yaml")
	require.NoError(t, os.WriteFile(syncFile, []byte(minimalYAML), 0644))

	stdout, stderr, err := runA7WithEnv(env, "config", "sync", "-f", syncFile, "--delete=false", "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	assert.Contains(t, stdout, "Sync completed")

	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, routeID)
}

func TestConfigSync_DeleteFalse(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-sync-nodelete-svc"
	routeID := "e2e-sync-nodelete-route"
	createTestServiceViaCLI(t, env, svcID)
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})

	createTestRouteWithServiceViaCLI(t, env, routeID, svcID)

	emptyYAML := `version: "1"
`
	syncFile := filepath.Join(t.TempDir(), "sync-empty.yaml")
	require.NoError(t, os.WriteFile(syncFile, []byte(emptyYAML), 0644))

	stdout, stderr, err := runA7WithEnv(env, "config", "sync", "-f", syncFile, "--delete=false", "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, routeID)
}

func TestConfigSync_FullRoundtrip(t *testing.T) {
	env := setupEnv(t)
	svcID := "e2e-sync-roundtrip-svc"
	routeID := "e2e-sync-roundtrip-route"
	createTestServiceViaCLI(t, env, svcID)
	t.Cleanup(func() {
		deleteRouteViaAdmin(t, routeID)
		deleteServiceViaAdmin(t, svcID)
	})

	createTestRouteWithServiceViaCLI(t, env, routeID, svcID)

	dumpFile := filepath.Join(t.TempDir(), "roundtrip-dump.yaml")
	_, stderr, err := runA7WithEnv(env, "config", "dump", "-g", gatewayGroup, "-f", dumpFile)
	require.NoError(t, err, stderr)

	data, err := os.ReadFile(dumpFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), routeID)

	stdout, stderr, err := runA7WithEnv(env, "config", "diff", "-f", dumpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	stdout, stderr, err = runA7WithEnv(env, "config", "sync", "-f", dumpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	assert.Contains(t, stdout, "Sync completed")
}

func TestConfigSync_InvalidConfig(t *testing.T) {
	env := setupEnv(t)

	// Sync with an invalid config should fail at validation.
	invalidYAML := `version: "99"
routes:
  - id: bad
    uri: /bad
`
	syncFile := filepath.Join(t.TempDir(), "sync-invalid.yaml")
	require.NoError(t, os.WriteFile(syncFile, []byte(invalidYAML), 0644))

	_, _, err := runA7WithEnv(env, "config", "sync", "-f", syncFile, "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestConfigSync_MissingFileFlag(t *testing.T) {
	env := setupEnv(t)

	// config sync without -f should error.
	_, _, err := runA7WithEnv(env, "config", "sync", "-g", gatewayGroup)
	assert.Error(t, err)
}
