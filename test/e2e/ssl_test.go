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

// readTestCert reads a test certificate file from testdata.
func readTestCert(t *testing.T) (string, string) {
	t.Helper()
	modRoot, err := resolveModuleRoot()
	require.NoError(t, err)
	certPath := filepath.Join(modRoot, "test/e2e/testdata/test.crt")
	keyPath := filepath.Join(modRoot, "test/e2e/testdata/test.key")
	cert, err := os.ReadFile(certPath)
	require.NoError(t, err, "failed to read test.crt")
	key, err := os.ReadFile(keyPath)
	require.NoError(t, err, "failed to read test.key")
	return string(cert), string(key)
}

// deleteSSLViaAdmin deletes an SSL certificate via the Admin API.
func deleteSSLViaAdmin(t *testing.T, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/ssls/%s", id), nil)
	if err == nil {
		resp.Body.Close()
	}
}

func TestSSL_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "ssl", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestSSL_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "ssl", "list", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestSSL_CRUD(t *testing.T) {
	env := setupEnv(t)
	sslID := "e2e-ssl-crud"
	t.Cleanup(func() { deleteSSLViaAdmin(t, sslID) })

	cert, key := readTestCert(t)

	sslJSON := fmt.Sprintf(`{
		"id": %q,
		"cert": %q,
		"key": %q,
		"snis": ["e2e-test.example.com"]
	}`, sslID, cert, key)

	tmpFile := filepath.Join(t.TempDir(), "ssl.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(sslJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "ssl", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "ssl create failed")

	// Get
	stdout, stderr, err = runA7WithEnv(env, "ssl", "get", sslID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, sslID)

	// Get JSON
	var ssl map[string]interface{}
	runA7JSON(t, env, &ssl, "ssl", "get", sslID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, sslID, ssl["id"])
	snis := requireJSONArray(t, ssl["snis"], "ssl.snis")
	assert.Contains(t, snis, "e2e-test.example.com")

	// Export (use get -o json; export is batch-only with cobra.NoArgs)
	runA7JSON(t, env, &ssl, "ssl", "get", sslID, "-g", gatewayGroup, "-o", "json")
	snis = requireJSONArray(t, ssl["snis"], "ssl.snis")
	assert.Contains(t, snis, "e2e-test.example.com")

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "ssl", "delete", sslID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	_, _, err = runA7WithEnv(env, "ssl", "get", sslID, "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestSSL_UpdateFlagsWithCertificatePathsAndSNIs(t *testing.T) {
	env := setupEnv(t)
	sslID := "e2e-ssl-update-flags"
	t.Cleanup(func() { deleteSSLViaAdmin(t, sslID) })

	cert, key := readTestCert(t)
	sslJSON := fmt.Sprintf(`{
		"id": %q,
		"cert": %q,
		"key": %q,
		"snis": ["old-flags.example.com"]
	}`, sslID, cert, key)
	tmpFile := filepath.Join(t.TempDir(), "ssl.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(sslJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "ssl", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	modRoot, err := resolveModuleRoot()
	require.NoError(t, err)
	certPathArg := filepath.Join(modRoot, "test/e2e/testdata/test.crt")
	keyPathArg := filepath.Join(modRoot, "test/e2e/testdata/test.key")

	stdout, stderr, err = runA7WithEnv(env, "ssl", "update", sslID,
		"--cert", certPathArg,
		"--key", keyPathArg,
		"--sni", "new-flags.example.com",
		"--sni", "new-flags-alt.example.com",
		"--status", "0",
		"-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	var ssl map[string]interface{}
	runA7JSON(t, env, &ssl, "ssl", "get", sslID, "-g", gatewayGroup, "-o", "json")
	snis := requireJSONArray(t, ssl["snis"], "ssl.snis")
	assert.Contains(t, snis, "new-flags.example.com")
	assert.Contains(t, snis, "new-flags-alt.example.com")
	if status, ok := ssl["status"]; ok {
		assert.Equal(t, float64(0), status)
	}

	stdout, stderr, err = runA7WithEnv(env, "ssl", "delete", sslID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	_, _, err = runA7WithEnv(env, "ssl", "get", sslID, "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestSSL_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "ssl", "delete", "nonexistent-ssl-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}
