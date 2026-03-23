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

// deleteConsumerGroupViaAdmin deletes a consumer group via the Admin API.
func deleteConsumerGroupViaAdmin(t *testing.T, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/consumer_groups/%s", id), nil)
	if err == nil {
		resp.Body.Close()
	}
}

func TestConsumerGroup_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "consumer-group", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestConsumerGroup_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "consumer-group", "list", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestConsumerGroup_CRUD(t *testing.T) {
	env := setupEnv(t)
	cgID := "e2e-consumer-group-crud"
	t.Cleanup(func() { deleteConsumerGroupViaAdmin(t, cgID) })

	cgJSON := fmt.Sprintf(`{
		"id": %q,
		"plugins": {
			"limit-count": {
				"count": 100,
				"time_window": 60,
				"key": "remote_addr",
				"rejected_code": 429
			}
		}
	}`, cgID)

	tmpFile := filepath.Join(t.TempDir(), "consumer-group.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(cgJSON), 0644))

	// Create
	stdout, stderr, err := runA7WithEnv(env, "consumer-group", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Get
	stdout, stderr, err = runA7WithEnv(env, "consumer-group", "get", cgID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, cgID)

	// Get JSON
	stdout, stderr, err = runA7WithEnv(env, "consumer-group", "get", cgID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "limit-count")

	// Export (use get -o json; export is batch-only with cobra.NoArgs)
	stdout, stderr, err = runA7WithEnv(env, "consumer-group", "get", cgID, "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "limit-count")

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "consumer-group", "delete", cgID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
}

func TestConsumerGroup_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "consumer-group", "delete", "nonexistent-cg-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}
