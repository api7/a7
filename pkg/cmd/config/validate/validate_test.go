package validate

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/httpmock"
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

// factoryWithHTTP wires an httpmock.Registry into the Factory so tests can
// stub the remote-validate endpoint. Mirrors dump_test.newFactory.
func factoryWithHTTP(reg *httpmock.Registry, ios *iostreams.IOStreams) *cmd.Factory {
	return &cmd.Factory{
		IOStreams:  ios,
		HttpClient: func() (*http.Client, error) { return reg.GetClient(), nil },
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

func TestConfigValidate_RejectsStreamRouteWithoutServiceID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	filePath := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(filePath, []byte(`
version: "1"
stream_routes:
  - id: tcp-route
    server_port: 9100
`), 0o644)
	require.NoError(t, err)

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{"-f", filePath})
	err = c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "stream_routes[0]: service_id is required by API7 EE")
}

func TestConfigValidate_MissingFileFlag(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	c := NewCmdValidate(factoryWithIO(ios))
	c.SetArgs([]string{})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "required flag \"file\" not set")
}

// TestConfigValidate_EmptyUnsupportedSections asserts that declaring an
// unsupported top-level section (upstreams, consumer_groups, service_templates)
// is rejected even when the section is explicitly empty. Presence alone is
// enough — the user is asserting an unsupported resource type.
func TestConfigValidate_EmptyUnsupportedSections(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "upstreams",
			body:    "version: \"1\"\nupstreams: []\n",
			wantErr: "upstreams are not supported",
		},
		{
			name:    "consumer_groups",
			body:    "version: \"1\"\nconsumer_groups: []\n",
			wantErr: "consumer_groups are not supported",
		},
		{
			name:    "plugin_configs",
			body:    "version: \"1\"\nplugin_configs: []\n",
			wantErr: "plugin_configs are not supported",
		},
		{
			name:    "service_templates",
			body:    "version: \"1\"\nservice_templates: []\n",
			wantErr: "service_templates are not supported",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			filePath := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(filePath, []byte(tc.body), 0o644))

			c := NewCmdValidate(factoryWithIO(ios))
			c.SetArgs([]string{"-f", filePath})
			err := c.Execute()

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

const validRemoteConfig = `
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
`

func TestConfigValidate_Remote_HappyPath(t *testing.T) {
	reg := &httpmock.Registry{}
	var receivedBody map[string]interface{}
	reg.RegisterResponder(http.MethodPost, "/apisix/admin/configs/validate", func(r *http.Request) (httpmock.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return httpmock.Response{}, err
		}
		if err := json.Unmarshal(body, &receivedBody); err != nil {
			return httpmock.Response{}, err
		}
		return httpmock.JSONResponse(`{}`), nil
	})

	ios, _, stdout, _ := iostreams.Test()
	filePath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(validRemoteConfig), 0o644))

	c := NewCmdValidate(factoryWithHTTP(reg, ios))
	c.SetArgs([]string{"-f", filePath, "--remote"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Config is valid")
	assert.Equal(t, 1, reg.CallCount(http.MethodPost, "/apisix/admin/configs/validate"))
	// The request body should carry the local routes and services in the
	// flat shape adc's validator uses.
	require.NotNil(t, receivedBody)
	assert.Contains(t, receivedBody, "routes")
	assert.Contains(t, receivedBody, "services")
	reg.Verify(t)
}

func TestConfigValidate_Remote_CollectsErrors(t *testing.T) {
	reg := &httpmock.Registry{}
	reg.Register(http.MethodPost, "/apisix/admin/configs/validate", httpmock.StringResponse(http.StatusBadRequest, `{
		"error_msg": "schema validation failed",
		"errors": [
			{"resource_type": "routes", "index": 0, "field": "plugins.limit-count.count", "message": "required field missing"},
			{"resource_type": "services", "index": 0, "message": "invalid upstream scheme"}
		]
	}`))

	ios, _, _, _ := iostreams.Test()
	filePath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(validRemoteConfig), 0o644))

	c := NewCmdValidate(factoryWithHTTP(reg, ios))
	c.SetArgs([]string{"-f", filePath, "--remote"})
	err := c.Execute()

	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "[remote]")
	assert.Contains(t, msg, "routes[0].plugins.limit-count.count")
	assert.Contains(t, msg, "required field missing")
	assert.Contains(t, msg, "services[0]")
	assert.Contains(t, msg, "invalid upstream scheme")
	reg.Verify(t)
}

func TestConfigValidate_Remote_LocalErrorsSkipRemote(t *testing.T) {
	reg := &httpmock.Registry{}
	// If the remote endpoint is hit, the mock responder returns an error
	// that surfaces through httpmock's RoundTrip — test failure.
	reg.RegisterResponder(http.MethodPost, "/apisix/admin/configs/validate", func(_ *http.Request) (httpmock.Response, error) {
		t.Fatalf("remote validate should not be called when local validation fails")
		return httpmock.Response{}, nil
	})

	ios, _, _, _ := iostreams.Test()
	filePath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(`
version: "1"
routes:
  - id: bad-route
`), 0o644))

	c := NewCmdValidate(factoryWithHTTP(reg, ios))
	c.SetArgs([]string{"-f", filePath, "--remote"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "either uri or uris is required")
	assert.NotContains(t, err.Error(), "[remote]")
	assert.Equal(t, 0, reg.CallCount(http.MethodPost, "/apisix/admin/configs/validate"))
}

func TestConfigValidate_NoRemoteFlag_BehavesAsBefore(t *testing.T) {
	reg := &httpmock.Registry{}
	reg.RegisterResponder(http.MethodPost, "/apisix/admin/configs/validate", func(_ *http.Request) (httpmock.Response, error) {
		t.Fatalf("remote validate should not be called without --remote")
		return httpmock.Response{}, nil
	})

	ios, _, stdout, _ := iostreams.Test()
	filePath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(validRemoteConfig), 0o644))

	c := NewCmdValidate(factoryWithHTTP(reg, ios))
	c.SetArgs([]string{"-f", filePath})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Config is valid")
	assert.Equal(t, 0, reg.CallCount(http.MethodPost, "/apisix/admin/configs/validate"))
}
