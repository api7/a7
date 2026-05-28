package configutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api7/a7/pkg/api"
)

func TestApplyEnvSubstitution_ServiceName(t *testing.T) {
	t.Setenv("NAME", "name")

	cfg := &api.ConfigFile{
		Services: []api.Service{
			{Name: "Test ${NAME}"},
		},
	}
	require.NoError(t, applyEnvSubstitution(cfg))

	assert.Equal(t, "Test name", cfg.Services[0].Name)
}

func TestApplyEnvSubstitution_EscapeIsPreserved(t *testing.T) {
	t.Setenv("NAME", "name")

	cfg := &api.ConfigFile{
		Services: []api.Service{
			{Name: `Test escape \${NAME}`},
		},
	}
	require.NoError(t, applyEnvSubstitution(cfg))

	// The backslash is consumed, the `${NAME}` is preserved as a literal.
	assert.Equal(t, "Test escape ${NAME}", cfg.Services[0].Name)
}

func TestApplyEnvSubstitution_PluginConfigValue(t *testing.T) {
	t.Setenv("SECRET", "secret")

	cfg := &api.ConfigFile{
		Consumers: []api.Consumer{
			{
				Username: "alice",
				Plugins: map[string]interface{}{
					"key-auth": map[string]interface{}{
						"key": "${SECRET}",
					},
				},
			},
		},
	}
	require.NoError(t, applyEnvSubstitution(cfg))

	keyAuth, ok := cfg.Consumers[0].Plugins["key-auth"].(map[string]interface{})
	require.True(t, ok, "expected key-auth plugin config to be map[string]interface{}")
	assert.Equal(t, "secret", keyAuth["key"])
}

func TestApplyEnvSubstitution_SSLCertAndKey(t *testing.T) {
	t.Setenv("CERT", "-----BEGIN CERT-----")
	t.Setenv("KEY", "-----BEGIN KEY-----")

	cfg := &api.ConfigFile{
		SSL: []api.SSL{
			{
				Cert: "${CERT}",
				Key:  "${KEY}",
				SNIs: []string{"test.com", "${CERT}"},
			},
		},
	}
	require.NoError(t, applyEnvSubstitution(cfg))

	assert.Equal(t, "-----BEGIN CERT-----", cfg.SSL[0].Cert)
	assert.Equal(t, "-----BEGIN KEY-----", cfg.SSL[0].Key)
	assert.Equal(t, []string{"test.com", "-----BEGIN CERT-----"}, cfg.SSL[0].SNIs)
}

func TestApplyEnvSubstitution_NestedGlobalRulePluginKeyValue(t *testing.T) {
	t.Setenv("SECRET", "secret")

	cfg := &api.ConfigFile{
		GlobalRules: []api.GlobalRule{
			{
				ID: "gr1",
				// The plugin name (the map key) is intentionally a variable
				// reference. Map keys are not substituted (matches adc).
				Plugins: map[string]interface{}{
					"${GLOBAL_PLUGIN}": map[string]interface{}{
						"key": "${SECRET}",
					},
				},
			},
		},
	}
	require.NoError(t, applyEnvSubstitution(cfg))

	plugin, ok := cfg.GlobalRules[0].Plugins["${GLOBAL_PLUGIN}"].(map[string]interface{})
	require.True(t, ok, "map keys must not be substituted")
	assert.Equal(t, "secret", plugin["key"])
}

func TestApplyEnvSubstitution_UnsetVarKeepsLiteral(t *testing.T) {
	// Make sure the variable is genuinely unset for this test.
	require.NoError(t, os.Unsetenv("DEFINITELY_NOT_SET_DAILY_VAR"))

	cfg := &api.ConfigFile{
		Services: []api.Service{
			{Name: "before-${DEFINITELY_NOT_SET_DAILY_VAR}-after"},
		},
	}
	require.NoError(t, applyEnvSubstitution(cfg))

	assert.Equal(t, "before-${DEFINITELY_NOT_SET_DAILY_VAR}-after", cfg.Services[0].Name)
}

func TestApplyEnvSubstitution_RoutesNestedFields(t *testing.T) {
	t.Setenv("NAME", "name")

	cfg := &api.ConfigFile{
		Services: []api.Service{
			{
				Name: "Test ${NAME}",
				Plugins: map[string]interface{}{
					"proxy-rewrite": map[string]interface{}{
						"headers": map[string]interface{}{
							"X-Trace": "${NAME}",
						},
					},
				},
			},
		},
		Routes: []api.Route{
			{
				Name: "Test ${NAME}",
				URIs: []string{"/test/${NAME}"},
			},
		},
	}
	require.NoError(t, applyEnvSubstitution(cfg))

	assert.Equal(t, "Test name", cfg.Services[0].Name)
	assert.Equal(t, []string{"/test/name"}, cfg.Routes[0].URIs)
	headers := cfg.Services[0].Plugins["proxy-rewrite"].(map[string]interface{})["headers"].(map[string]interface{})
	assert.Equal(t, "name", headers["X-Trace"])
}

func TestApplyEnvSubstitution_PluginMetadataNestedValue(t *testing.T) {
	t.Setenv("NOTE", "note")

	cfg := &api.ConfigFile{
		PluginMetadata: []api.PluginMetadataEntry{
			{
				"plugin_name": "file-logger",
				"log_format": map[string]interface{}{
					"note": "${NOTE}",
				},
			},
		},
	}
	require.NoError(t, applyEnvSubstitution(cfg))

	entry := cfg.PluginMetadata[0]
	format := entry["log_format"].(map[string]interface{})
	assert.Equal(t, "note", format["note"])
}

func TestApplyEnvSubstitution_NumbersAndBoolsUntouched(t *testing.T) {
	t.Setenv("NAME", "name")

	cfg := &api.ConfigFile{
		Routes: []api.Route{
			{
				Name:     "Test ${NAME}",
				Status:   1,
				Priority: 10,
				Plugins: map[string]interface{}{
					"limit-count": map[string]interface{}{
						"count":         100,
						"time_window":   60,
						"allow_degrade": true,
						"key":           "${NAME}",
					},
				},
			},
		},
	}
	require.NoError(t, applyEnvSubstitution(cfg))

	assert.Equal(t, "Test name", cfg.Routes[0].Name)
	assert.Equal(t, 1, cfg.Routes[0].Status)
	assert.Equal(t, 10, cfg.Routes[0].Priority)
	lc := cfg.Routes[0].Plugins["limit-count"].(map[string]interface{})
	assert.Equal(t, 100, lc["count"])
	assert.Equal(t, 60, lc["time_window"])
	assert.Equal(t, true, lc["allow_degrade"])
	assert.Equal(t, "name", lc["key"])
}

func TestApplyEnvSubstitution_StringLabels(t *testing.T) {
	t.Setenv("ENV", "prod")

	cfg := &api.ConfigFile{
		Routes: []api.Route{
			{
				Name: "r1",
				Labels: map[string]string{
					"env": "${ENV}",
				},
			},
		},
	}
	require.NoError(t, applyEnvSubstitution(cfg))

	assert.Equal(t, "prod", cfg.Routes[0].Labels["env"])
}

func TestReadConfigFile_AppliesSubstitution(t *testing.T) {
	t.Setenv("ROUTE_NAME", "from-env")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
version: "1"
routes:
  - id: r1
    name: "Hello ${ROUTE_NAME}"
    uri: /sync/${ROUTE_NAME}
`), 0o644))

	cfg, err := ReadConfigFile(path)
	require.NoError(t, err)
	require.Len(t, cfg.Routes, 1)
	assert.Equal(t, "Hello from-env", cfg.Routes[0].Name)
	assert.Equal(t, "/sync/from-env", cfg.Routes[0].URI)
}
