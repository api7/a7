//go:build e2e

package e2e

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContext_CreateAndUse(t *testing.T) {
	env := []string{"A7_CONFIG_DIR=" + t.TempDir()}

	_, stderr, err := runA7WithEnv(env,
		"context", "create", "local",
		"--server", "https://localhost:7443",
		"--token", "test123",
		"--gateway-group", "default",
		"--tls-skip-verify",
	)
	require.NoError(t, err, stderr)

	stdout, stderr, err := runA7WithEnv(env, "context", "current")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "local")

	_, stderr, err = runA7WithEnv(env,
		"context", "create", "staging",
		"--server", "https://localhost:7443",
		"--token", "test123",
		"--gateway-group", "default",
		"--tls-skip-verify",
	)
	require.NoError(t, err, stderr)

	_, stderr, err = runA7WithEnv(env, "context", "use", "staging")
	require.NoError(t, err, stderr)

	stdout, stderr, err = runA7WithEnv(env, "context", "current")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "staging")
}

func TestContext_List(t *testing.T) {
	env := []string{"A7_CONFIG_DIR=" + t.TempDir()}

	_, stderr, err := runA7WithEnv(env, "context", "create", "local", "--server", "https://localhost:7443")
	require.NoError(t, err, stderr)
	_, stderr, err = runA7WithEnv(env, "context", "create", "staging", "--server", "https://localhost:7443")
	require.NoError(t, err, stderr)

	stdout, stderr, err := runA7WithEnv(env, "context", "list")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "local")
	assert.Contains(t, stdout, "staging")

	stdout, stderr, err = runA7WithEnv(env, "context", "list", "--output", "json")
	require.NoError(t, err, stderr)
	var parsed any
	require.NoError(t, json.Unmarshal([]byte(stdout), &parsed))
}

func TestContext_Delete(t *testing.T) {
	env := []string{"A7_CONFIG_DIR=" + t.TempDir()}

	_, stderr, err := runA7WithEnv(env, "context", "create", "local", "--server", "https://localhost:7443")
	require.NoError(t, err, stderr)
	_, stderr, err = runA7WithEnv(env, "context", "create", "staging", "--server", "https://localhost:7443")
	require.NoError(t, err, stderr)

	_, stderr, err = runA7WithEnv(env, "context", "delete", "local")
	require.NoError(t, err, stderr)

	stdout, stderr, err := runA7WithEnv(env, "context", "list")
	require.NoError(t, err, stderr)
	assert.NotContains(t, stdout, "local")
	assert.Contains(t, stdout, "staging")
}

func TestContext_CreateDuplicate(t *testing.T) {
	env := []string{"A7_CONFIG_DIR=" + t.TempDir()}

	_, stderr, err := runA7WithEnv(env, "context", "create", "local", "--server", "https://localhost:7443")
	require.NoError(t, err, stderr)

	_, stderr, err = runA7WithEnv(env, "context", "create", "local", "--server", "https://localhost:7443")
	require.Error(t, err)
	assert.Contains(t, stderr, "already exists")
}

func TestContext_UseNonExistent(t *testing.T) {
	env := []string{"A7_CONFIG_DIR=" + t.TempDir()}

	_, stderr, err := runA7WithEnv(env, "context", "use", "ghost")
	require.Error(t, err)
	assert.Contains(t, stderr, "not found")
}

func TestContext_DeleteActive(t *testing.T) {
	env := []string{"A7_CONFIG_DIR=" + t.TempDir()}

	_, stderr, err := runA7WithEnv(env, "context", "create", "local", "--server", "https://localhost:7443")
	require.NoError(t, err, stderr)

	_, stderr, err = runA7WithEnv(env, "context", "delete", "local")
	require.NoError(t, err, stderr)

	stdout, stderr, err := runA7WithEnv(env, "context", "list")
	require.NoError(t, err, stderr)
	assert.Contains(t, stderr, "No contexts configured")
	assert.Empty(t, stdout)
}

func TestContext_CreateWithCACert(t *testing.T) {
	env := []string{"A7_CONFIG_DIR=" + t.TempDir()}

	fakeCA := filepath.Join(t.TempDir(), "fake-ca.pem")
	_, stderr, err := runA7WithEnv(env,
		"context", "create", "with-ca",
		"--server", "https://localhost:7443",
		"--token", "test123",
		"--gateway-group", "default",
		"--ca-cert", fakeCA,
	)
	require.NoError(t, err, stderr)
}
