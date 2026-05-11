package validate

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/iostreams"
)

type mockConfig struct {
	baseURL string
}

func (m *mockConfig) BaseURL() string                                 { return m.baseURL }
func (m *mockConfig) Token() string                                   { return "" }
func (m *mockConfig) GatewayGroup() string                            { return "" }
func (m *mockConfig) TLSSkipVerify() bool                             { return false }
func (m *mockConfig) CACert() string                                  { return "" }
func (m *mockConfig) CurrentContext() string                          { return "test" }
func (m *mockConfig) Contexts() []config.Context                      { return nil }
func (m *mockConfig) GetContext(name string) (*config.Context, error) { return nil, nil }
func (m *mockConfig) AddContext(ctx config.Context) error             { return nil }
func (m *mockConfig) RemoveContext(name string) error                 { return nil }
func (m *mockConfig) SetCurrentContext(name string) error             { return nil }
func (m *mockConfig) Save() error                                     { return nil }

func factoryWithIO(ios *iostreams.IOStreams) *cmd.Factory {
	return &cmd.Factory{
		IOStreams: ios,
		HttpClient: func() (*http.Client, error) {
			return nil, nil
		},
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://localhost:9180"}, nil
		},
	}
}

func TestConfigValidate_ValidYAML(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()

	filePath := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(filePath, []byte(`
version: "1"
routes:
  - id: "route-1"
    uri: /hello
    service_id: service-1
services:
  - id: service-1
    name: service-1
    upstream:
      type: roundrobin
      nodes:
        - host: 127.0.0.1
          port: 8080
          weight: 1
consumers:
  - username: jack
`), 0o644)
	require.NoError(t, err)

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{"-f", filePath})
	err = c.Execute()

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Config is valid")
}

func TestConfigValidate_ValidJSON(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()

	filePath := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(filePath, []byte(`{
		"version": "1",
		"routes": [{"id": "route-1", "uri": "/hello", "service_id": "service-1"}],
		"services": [{"id": "service-1", "name": "service-1", "upstream": {"type": "roundrobin", "nodes": [{"host": "127.0.0.1", "port": 8080, "weight": 1}]}}],
		"consumers": [{"username": "jack"}]
	}`), 0o644)
	require.NoError(t, err)

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{"-f", filePath})
	err = c.Execute()

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Config is valid")
}

func TestConfigValidate_MissingVersion(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	filePath := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(filePath, []byte(`
routes:
  - id: "route-1"
    uri: /hello
`), 0o644)
	require.NoError(t, err)

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{"-f", filePath})
	err = c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestConfigValidate_InvalidVersion(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	filePath := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(filePath, []byte(`
version: "2"
`), 0o644)
	require.NoError(t, err)

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{"-f", filePath})
	err = c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "version must be \"1\"")
}

func TestConfigValidate_DuplicateIDs(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	filePath := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(filePath, []byte(`
version: "1"
routes:
  - id: "route-1"
    uri: /hello
    service_id: service-1
  - id: "route-1"
    uri: /hello2
    service_id: service-1
`), 0o644)
	require.NoError(t, err)

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{"-f", filePath})
	err = c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate id \"route-1\"")
}

func TestConfigValidate_MissingRouteURI(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	filePath := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(filePath, []byte(`
version: "1"
routes:
  - id: "route-1"
    service_id: service-1
`), 0o644)
	require.NoError(t, err)

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{"-f", filePath})
	err = c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "either uri or uris is required")
}

func TestConfigValidate_MissingRouteServiceID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	filePath := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(filePath, []byte(`
version: "1"
routes:
  - id: "route-1"
    uri: /hello
`), 0o644)
	require.NoError(t, err)

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{"-f", filePath})
	err = c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "service_id is required by current API7 EE")
}

func TestConfigValidate_MissingConsumerUsername(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	filePath := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(filePath, []byte(`
version: "1"
consumers:
  - plugins:
      key-auth:
        key: foo
`), 0o644)
	require.NoError(t, err)

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{"-f", filePath})
	err = c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

func TestConfigValidate_RejectsUnsupportedTopLevelUpstreams(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	filePath := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(filePath, []byte(`
version: "1"
upstreams:
  - id: backend
    nodes:
      127.0.0.1:8080: 1
`), 0o644)
	require.NoError(t, err)

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{"-f", filePath})
	err = c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "upstreams are not supported as top-level API7 EE resources")
}

func TestConfigValidate_RejectsUnsupportedConsumerGroups(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	filePath := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(filePath, []byte(`
version: "1"
consumer_groups:
  - id: tenants
`), 0o644)
	require.NoError(t, err)

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{"-f", filePath})
	err = c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "consumer_groups are not supported by current API7 EE")
}

func TestConfigValidate_MissingFileFlag(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "required flag \"file\" not set")
}
