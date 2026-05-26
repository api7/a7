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

// deleteGlobalRuleViaAdmin deletes a global rule via the Admin API.
func deleteGlobalRuleViaAdmin(t *testing.T, id string) {
	t.Helper()
	resp, err := runtimeAdminAPI("DELETE", fmt.Sprintf("/apisix/admin/global_rules/%s", id), nil)
	if err == nil {
		resp.Body.Close()
	}
}

func TestGlobalRule_List(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "global-rule", "list", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestGlobalRule_ListJSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "global-rule", "list", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestGlobalRule_CRUD(t *testing.T) {
	env := setupEnv(t)
	// API7 EE derives the global rule id from its single plugin name.
	grID := "prometheus"
	t.Cleanup(func() { deleteGlobalRuleViaAdmin(t, grID) })

	grJSON := `{
		"plugins": {
			"prometheus": {}
		}
	}`

	tmpFile := filepath.Join(t.TempDir(), "global-rule.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(grJSON), 0644))

	// Create
	stdout, stderr, err := runA7WithEnv(env, "global-rule", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// Get
	stdout, stderr, err = runA7WithEnv(env, "global-rule", "get", grID, "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, grID)

	// Get JSON
	var globalRule map[string]interface{}
	runA7JSON(t, env, &globalRule, "global-rule", "get", grID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, grID, globalRule["id"])
	plugins := requireJSONObject(t, globalRule["plugins"], "global_rule.plugins")
	assert.Contains(t, plugins, "prometheus")

	// Update — API7 EE requires exactly one plugin per global rule.
	updateJSON := fmt.Sprintf(`{
		"id": %q,
		"plugins": {
			"prometheus": {"prefer_name": true}
		}
	}`, grID)
	tmpFile2 := filepath.Join(t.TempDir(), "global-rule-update.json")
	require.NoError(t, os.WriteFile(tmpFile2, []byte(updateJSON), 0644))

	stdout, stderr, err = runA7WithEnv(env, "global-rule", "update", grID, "-f", tmpFile2, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	// Export (use get -o json; export is batch-only with cobra.NoArgs)
	runA7JSON(t, env, &globalRule, "global-rule", "get", grID, "-g", gatewayGroup, "-o", "json")
	plugins = requireJSONObject(t, globalRule["plugins"], "global_rule.plugins")
	prometheus := requireJSONObject(t, plugins["prometheus"], "global_rule.plugins.prometheus")
	assert.Equal(t, true, prometheus["prefer_name"])

	// Delete
	stdout, stderr, err = runA7WithEnv(env, "global-rule", "delete", grID, "--force", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	_, _, err = runA7WithEnv(env, "global-rule", "get", grID, "-g", gatewayGroup)
	assert.Error(t, err)
}

func TestGlobalRule_DeleteNonexistent(t *testing.T) {
	env := setupEnv(t)

	_, _, err := runA7WithEnv(env, "global-rule", "delete", "nonexistent-gr-12345", "--force", "-g", gatewayGroup)
	assert.Error(t, err)
}

// TestGlobalRule_SkillExample covers the skill file pattern:
// a7 global-rule create -f - <<'EOF'
func TestGlobalRule_SkillExample(t *testing.T) {
	env := setupEnv(t)
	// API7 EE derives the global rule id from its single plugin name.
	grID := "ip-restriction"
	t.Cleanup(func() { deleteGlobalRuleViaAdmin(t, grID) })

	grJSON := `{
		"plugins": {
			"ip-restriction": {
				"whitelist": ["10.0.0.0/8"]
			}
		}
	}`

	tmpFile := filepath.Join(t.TempDir(), "gr.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(grJSON), 0644))

	_, stderr, err := runA7WithEnv(env, "global-rule", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, stderr)

	// Verify
	var globalRule map[string]interface{}
	runA7JSON(t, env, &globalRule, "global-rule", "get", grID, "-g", gatewayGroup, "-o", "json")
	assert.Equal(t, grID, globalRule["id"])
	plugins := requireJSONObject(t, globalRule["plugins"], "global_rule.plugins")
	ipRestriction := requireJSONObject(t, plugins["ip-restriction"], "global_rule.plugins.ip-restriction")
	whitelist := requireJSONArray(t, ipRestriction["whitelist"], "global_rule.plugins.ip-restriction.whitelist")
	assert.Contains(t, whitelist, "10.0.0.0/8")
}
