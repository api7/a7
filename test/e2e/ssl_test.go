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
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Get
	stdout, stderr, err = runA7WithEnv(env, "ssl", "get", sslID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, sslID)

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "ssl", "get", sslID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "e2e-test.example.com")

	// Export (use get -o json; export is batch-only with cobra.NoArgs)
	stdout, stderr, err = runA7WithEnv(env, "ssl", "get", sslID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "e2e-test.example.com")

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "ssl", "delete", sslID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
}

func TestSSL_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "ssl", "delete", "nonexistent-ssl-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}
