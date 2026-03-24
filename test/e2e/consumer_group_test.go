//go:build e2e

package e2e

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
	t.Skip("consumer_group is not exposed via API7 EE Admin API")
}

func TestConsumerGroup_ListJSON(t *testing.T) {
	t.Skip("consumer_group is not exposed via API7 EE Admin API")
}

func TestConsumerGroup_CRUD(t *testing.T) {
	t.Skip("consumer_group is not exposed via API7 EE Admin API")
}

func TestConsumerGroup_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "consumer-group", "delete", "nonexistent-cg-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}
