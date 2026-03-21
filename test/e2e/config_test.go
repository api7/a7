//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDump_YAML(t *testing.T) {
	env := setupEnv(t)

	// Default output is YAML.
	stdout, stderr, err := runA7WithEnv(env, "config", "dump", "-g", gatewayGroup)
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
	// YAML output should contain "version:" or resource keys.
	assert.Contains(t, stdout, "version")
}

func TestConfigDump_JSON(t *testing.T) {
	env := setupEnv(t)

	stdout, stderr, err := runA7WithEnv(env, "config", "dump", "-g", gatewayGroup, "-o", "json")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)

	// Verify it's valid JSON.
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result), "should be valid JSON")
}

func TestConfigDump_ToFile(t *testing.T) {
	env := setupEnv(t)
	outFile := filepath.Join(t.TempDir(), "dumped-config.yaml")

	stdout, stderr, err := runA7WithEnv(env, "config", "dump", "-g", gatewayGroup, "-f", outFile)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	// File should exist and have content.
	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), "version")
}

func TestConfigDump_ToFileJSON(t *testing.T) {
	env := setupEnv(t)
	outFile := filepath.Join(t.TempDir(), "dumped-config.json")

	stdout, stderr, err := runA7WithEnv(env, "config", "dump", "-g", gatewayGroup, "-f", outFile, "-o", "json")
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)

	data, err := os.ReadFile(outFile)
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result), "file should contain valid JSON")
}

func TestConfigValidate_Valid(t *testing.T) {
	env := setupEnv(t)

	validYAML := `version: "1"
routes:
  - id: valid-route-1
    uri: /test
    upstream:
      type: roundrobin
      nodes:
        "127.0.0.1:8080": 1
consumers:
  - username: valid-consumer-1
`
	tmpFile := filepath.Join(t.TempDir(), "valid-config.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(validYAML), 0644))

	stdout, stderr, err := runA7WithEnv(env, "config", "validate", "-f", tmpFile)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	assert.Contains(t, stdout, "Config is valid")
}

func TestConfigValidate_ValidJSON(t *testing.T) {
	env := setupEnv(t)

	validJSON := `{
		"version": "1",
		"routes": [
			{
				"id": "valid-route-json",
				"uri": "/test-json",
				"upstream": {
					"type": "roundrobin",
					"nodes": {"127.0.0.1:8080": 1}
				}
			}
		]
	}`
	tmpFile := filepath.Join(t.TempDir(), "valid-config.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(validJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "config", "validate", "-f", tmpFile)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	assert.Contains(t, stdout, "Config is valid")
}

func TestConfigValidate_MissingVersion(t *testing.T) {
	env := setupEnv(t)

	invalidYAML := `routes:
  - id: no-version-route
    uri: /test
`
	tmpFile := filepath.Join(t.TempDir(), "invalid-config.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(invalidYAML), 0644))

	_, stderr, err := runA7WithEnv(env, "config", "validate", "-f", tmpFile)
	assert.Error(t, err)
	combined := stderr
	assert.True(t, strings.Contains(combined, "version is required") || strings.Contains(combined, "validation failed"),
		"expected validation error about version, got: %s", combined)
}

func TestConfigValidate_InvalidVersion(t *testing.T) {
	env := setupEnv(t)

	invalidYAML := `version: "99"
routes:
  - id: bad-version-route
    uri: /test
`
	tmpFile := filepath.Join(t.TempDir(), "invalid-version.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(invalidYAML), 0644))

	_, _, err := runA7WithEnv(env, "config", "validate", "-f", tmpFile)
	assert.Error(t, err)
}

func TestConfigValidate_MissingRouteURI(t *testing.T) {
	env := setupEnv(t)

	invalidYAML := `version: "1"
routes:
  - id: no-uri-route
`
	tmpFile := filepath.Join(t.TempDir(), "invalid-no-uri.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(invalidYAML), 0644))

	_, _, err := runA7WithEnv(env, "config", "validate", "-f", tmpFile)
	assert.Error(t, err)
}

func TestConfigValidate_DuplicateIDs(t *testing.T) {
	env := setupEnv(t)

	invalidYAML := `version: "1"
routes:
  - id: dup-route
    uri: /test-1
  - id: dup-route
    uri: /test-2
`
	tmpFile := filepath.Join(t.TempDir(), "invalid-dup-ids.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(invalidYAML), 0644))

	_, _, err := runA7WithEnv(env, "config", "validate", "-f", tmpFile)
	assert.Error(t, err)
}

func TestConfigValidate_EmptyConsumerUsername(t *testing.T) {
	env := setupEnv(t)

	invalidYAML := `version: "1"
consumers:
  - username: ""
`
	tmpFile := filepath.Join(t.TempDir(), "invalid-consumer.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(invalidYAML), 0644))

	_, _, err := runA7WithEnv(env, "config", "validate", "-f", tmpFile)
	assert.Error(t, err)
}

func TestConfigValidate_MissingFileFlag(t *testing.T) {
	env := setupEnv(t)

	// config validate without -f should error.
	_, _, err := runA7WithEnv(env, "config", "validate")
	assert.Error(t, err)
}

func TestConfigDiff_MatchingConfig(t *testing.T) {
	env := setupEnv(t)

	// First, dump the current config.
	dumpFile := filepath.Join(t.TempDir(), "current.yaml")
	_, stderr, err := runA7WithEnv(env, "config", "dump", "-g", gatewayGroup, "-f", dumpFile)
	require.NoError(t, err, stderr)

	// Diff against same config — should show no differences (exit code 0).
	stdout, stderr, err := runA7WithEnv(env, "config", "diff", "-f", dumpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
}

func TestConfigDiff_WithDifferences(t *testing.T) {
	env := setupEnv(t)

	// Create a config with an extra route that doesn't exist remotely.
	diffYAML := `version: "1"
routes:
  - id: e2e-diff-extra-route
    uri: /diff-extra
    upstream:
      type: roundrobin
      nodes:
        "127.0.0.1:8080": 1
`
	tmpFile := filepath.Join(t.TempDir(), "diff-config.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(diffYAML), 0644))

	// Diff should detect differences (exit code non-zero due to SilentError).
	_, _, err := runA7WithEnv(env, "config", "diff", "-f", tmpFile, "-g", gatewayGroup)
	assert.Error(t, err, "diff should report differences")
}

func TestConfigDiff_JSONOutput(t *testing.T) {
	env := setupEnv(t)

	diffYAML := `version: "1"
routes:
  - id: e2e-diff-json-route
    uri: /diff-json
    upstream:
      type: roundrobin
      nodes:
        "127.0.0.1:8080": 1
`
	tmpFile := filepath.Join(t.TempDir(), "diff-json-config.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(diffYAML), 0644))

	stdout, _, err := runA7WithEnv(env, "config", "diff", "-f", tmpFile, "-g", gatewayGroup, "-o", "json")
	// Even if diff has differences, -o json should produce valid JSON.
	_ = err // diff with differences returns non-zero exit
	if stdout != "" {
		var result interface{}
		assert.NoError(t, json.Unmarshal([]byte(stdout), &result), "JSON output should be valid")
	}
}

func TestConfigDiff_MissingFileFlag(t *testing.T) {
	env := setupEnv(t)

	// config diff without -f should error.
	_, _, err := runA7WithEnv(env, "config", "diff", "-g", gatewayGroup)
	assert.Error(t, err)
}
