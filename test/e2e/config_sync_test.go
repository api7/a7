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

	// Create a config with a new route to sync.
	syncYAML := `version: "1"
routes:
  - id: e2e-sync-dryrun-route
    name: e2e-sync-dryrun-route
    paths:
      - /sync-dryrun
    upstream:
      type: roundrobin
      nodes:
        "127.0.0.1:8080": 1
`
	tmpFile := filepath.Join(t.TempDir(), "sync-dryrun.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(syncYAML), 0644))

	// --dry-run should show what would change without applying.
	stdout, stderr, err := runA7WithEnv(env, "config", "sync", "-f", tmpFile, "--dry-run", "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// The route should NOT exist after dry-run.
	_, _, getErr := runA7WithEnv(env, "route", "get", "e2e-sync-dryrun-route", "-g", gatewayGroup)
	assert.Error(t, getErr, "dry-run should not create resources")
}

func TestConfigSync_CreateAndCleanup(t *testing.T) {
	env := setupEnv(t)
	routeID := "e2e-sync-create-route"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	// Create a minimal config with just the new route
	// and use --delete=false to avoid removing existing resources.
	minimalYAML := fmt.Sprintf(`version: "1"
routes:
  - id: %s
    name: %s
    paths:
      - /sync-create-test
    upstream:
      type: roundrobin
      nodes:
        "127.0.0.1:8080": 1
`, routeID, routeID)

	syncFile := filepath.Join(t.TempDir(), "sync-create.yaml")
	require.NoError(t, os.WriteFile(syncFile, []byte(minimalYAML), 0644))

	// Sync with --delete=false so we only create, not remove existing resources.
	stdout, stderr, err := runA7WithEnv(env, "config", "sync", "-f", syncFile, "--delete=false", "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	assert.Contains(t, stdout, "Sync completed")

	// Verify route was created.
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, routeID)
}

func TestConfigSync_DeleteFalse(t *testing.T) {
	env := setupEnv(t)
	routeID := "e2e-sync-nodelete-route"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	// First create a route via CLI.
	createTestRouteViaCLI(t, env, routeID)

	// Now sync an empty config with --delete=false.
	emptyYAML := `version: "1"
`
	syncFile := filepath.Join(t.TempDir(), "sync-empty.yaml")
	require.NoError(t, os.WriteFile(syncFile, []byte(emptyYAML), 0644))

	stdout, stderr, err := runA7WithEnv(env, "config", "sync", "-f", syncFile, "--delete=false", "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Route should STILL exist because --delete=false.
	stdout, stderr, err = runA7WithEnv(env, "route", "get", routeID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, routeID)
}

func TestConfigSync_FullRoundtrip(t *testing.T) {
	env := setupEnv(t)
	routeID := "e2e-sync-roundtrip-route"
	t.Cleanup(func() { deleteRouteViaAdmin(t, routeID) })

	// Step 1: Create route via CLI.
	createTestRouteViaCLI(t, env, routeID)

	// Step 2: Dump current config.
	dumpFile := filepath.Join(t.TempDir(), "roundtrip-dump.yaml")
	_, stderr, err := runA7WithEnv(env, "config", "dump", "-g", gatewayGroup, "-f", dumpFile)
	require.NoError(t, err, stderr)

	data, err := os.ReadFile(dumpFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), routeID)

	// Step 3: Diff should show no differences.
	stdout, stderr, err := runA7WithEnv(env, "config", "diff", "-f", dumpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Step 4: Sync same config — should be a no-op.
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
