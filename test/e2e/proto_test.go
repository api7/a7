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

// deleteProtoViaAdmin deletes a proto via the Admin API.
func deleteProtoViaAdmin(t *testing.T, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/protos/%s", id), nil)
	if err == nil {
		resp.Body.Close()
	}
}

func TestProto_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "proto", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestProto_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "proto", "list", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestProto_CRUD(t *testing.T) {
	env := setupEnv(t)
	protoID := "e2e-proto-crud"
	t.Cleanup(func() { deleteProtoViaAdmin(t, protoID) })

	protoJSON := fmt.Sprintf(`{
		"id": %q,
		"desc": "e2e test proto",
		"content": "syntax = \"proto3\";\npackage helloworld;\nservice Greeter {\n  rpc SayHello (HelloRequest) returns (HelloReply) {}\n}\nmessage HelloRequest { string name = 1; }\nmessage HelloReply { string message = 1; }"
	}`, protoID)

	tmpFile := filepath.Join(t.TempDir(), "proto.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(protoJSON), 0644))

	// Create
	stdout, stderr, err := runA7WithEnv(env, "proto", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Get
	stdout, stderr, err = runA7WithEnv(env, "proto", "get", protoID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, protoID)

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "proto", "get", protoID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "helloworld")

	// Export (use get -o json; export is batch-only with cobra.NoArgs)
	stdout, stderr, err = runA7WithEnv(env, "proto", "get", protoID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "helloworld")

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "proto", "delete", protoID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
}

func TestProto_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "proto", "delete", "nonexistent-proto-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}
