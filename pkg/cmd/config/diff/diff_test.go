package diff

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmdutil"
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

func registerEmptyResources(reg *httpmock.Registry, skip map[string]bool) {
	resources := []string{
		"/apisix/admin/services",
		"/apisix/admin/consumers",
		"/apisix/admin/ssls",
		"/apisix/admin/global_rules",
		"/apisix/admin/stream_routes",
		"/apisix/admin/protos",
		"/apisix/admin/secret_providers",
	}
	for _, path := range resources {
		if skip != nil && skip[path] {
			continue
		}
		reg.Register(http.MethodGet, path, httpmock.JSONResponse(`{"total":0,"list":[]}`))
	}
	if skip == nil || !skip["/apisix/admin/plugins/list"] {
		reg.Register(http.MethodGet, "/apisix/admin/plugins/list", httpmock.JSONResponse(`[]`))
	}
}

func newFactory(reg *httpmock.Registry, ios *iostreams.IOStreams) *cmd.Factory {
	return &cmd.Factory{
		IOStreams:  ios,
		HttpClient: func() (*http.Client, error) { return reg.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://localhost:9180"}, nil
		},
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
	return file
}

func TestConfigDiff_CreateUpdateDelete(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/services": true})
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 2,
		"list": [{"id":"svc-1","name":"svc-1"},{"id":"svc-2","name":"svc-2"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(`{
		"total": 2,
		"list": [
			{"id":"r1","uri":"/a","name":"old"},
			{"id":"r3","uri":"/c","name":"gone"}
		]
	}`))

	local := writeConfig(t, `
version: "1"
routes:
  - id: r1
    uri: /a
    name: new
  - id: r2
    uri: /b
    name: created
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdDiff(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	err := c.Execute()

	require.Error(t, err)
	assert.True(t, cmdutil.IsSilent(err))
	out := stdout.String()
	assert.Contains(t, out, "Differences found")
	assert.Contains(t, out, "routes: create=1 update=1 delete=1")
	assert.Contains(t, out, "CREATE r2")
	assert.Contains(t, out, "UPDATE r1")
	assert.Contains(t, out, "DELETE r3")
	reg.Verify(t)
}

func TestConfigDiff_NoDiff(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/services": true})
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"svc-1","name":"svc"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"r1","uri":"/same","name":"same"}]
	}`))

	local := writeConfig(t, `
version: "1"
services:
  - id: svc-1
    name: svc
routes:
  - id: r1
    uri: /same
    name: same
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdDiff(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "No differences found.")
	reg.Verify(t)
}

func TestConfigDiff_EmptyLocal(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/services": true})
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"svc-1","name":"svc"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"r1","uri":"/same","name":"same"}]
	}`))

	local := writeConfig(t, `
version: "1"
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdDiff(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	err := c.Execute()

	require.Error(t, err)
	assert.True(t, cmdutil.IsSilent(err))
	assert.Contains(t, stdout.String(), "DELETE r1")
	reg.Verify(t)
}

func TestConfigDiff_EmptyRemote(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, nil)

	local := writeConfig(t, `
version: "1"
routes:
  - id: r1
    uri: /same
    name: same
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdDiff(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	err := c.Execute()

	require.Error(t, err)
	assert.True(t, cmdutil.IsSilent(err))
	assert.Contains(t, stdout.String(), "CREATE r1")
	reg.Verify(t)
}

func TestConfigDiff_RejectsUnsupportedSections(t *testing.T) {
	reg := &httpmock.Registry{}

	local := writeConfig(t, `
version: "1"
upstreams:
  - id: u1
    nodes:
      127.0.0.1:8080: 1
consumer_groups:
  - id: group1
`)

	ios, _, _, _ := iostreams.Test()
	c := NewCmdDiff(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported declarative config sections")
	assert.Contains(t, err.Error(), "upstreams")
	assert.Contains(t, err.Error(), "consumer_groups")
	reg.Verify(t)
}

func TestConfigDiff_JSONOutput(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, nil)

	local := writeConfig(t, `
version: "1"
routes:
  - id: r1
    uri: /json
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdDiff(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local, "--output", "json"})
	err := c.Execute()

	require.Error(t, err)
	assert.True(t, cmdutil.IsSilent(err))

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	routes := result["routes"].(map[string]interface{})
	create := routes["create"].([]interface{})
	assert.Len(t, create, 1)
	reg.Verify(t)
}

func TestConfigDiff_CredentialCreateUpdateDelete(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/consumers": true})
	// Two consumers: alice (kept, credentials change) and bob (kept too).
	reg.Register(http.MethodGet, "/apisix/admin/consumers", httpmock.JSONResponse(`{
		"total": 2,
		"list": [{"username":"alice"},{"username":"bob"}]
	}`))
	// alice has two credentials remotely: k-update (will be updated) and
	// k-delete (will be removed).
	reg.Register(http.MethodGet, "/apisix/admin/consumers/alice/credentials", httpmock.JSONResponse(`{
		"total": 2,
		"list": [
			{"name":"k-update","plugins":{"key-auth":{"key":"old"}}},
			{"name":"k-delete","plugins":{"key-auth":{"key":"gone"}}}
		]
	}`))
	// bob has no credentials remotely; local config will add one.
	reg.Register(http.MethodGet, "/apisix/admin/consumers/bob/credentials", httpmock.JSONResponse(`{
		"total": 0,
		"list": []
	}`))

	local := writeConfig(t, `
version: "1"
consumers:
  - username: alice
    credentials:
      - name: k-update
        plugins:
          key-auth:
            key: new
      - name: k-create
        plugins:
          key-auth:
            key: brand-new
  - username: bob
    credentials:
      - name: bob-key
        plugins:
          key-auth:
            key: bob-secret
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdDiff(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	err := c.Execute()

	require.Error(t, err)
	assert.True(t, cmdutil.IsSilent(err))
	out := stdout.String()
	assert.Contains(t, out, "credentials: create=2 update=1 delete=1")
	assert.Contains(t, out, "CREATE alice/k-create")
	assert.Contains(t, out, "CREATE bob/bob-key")
	assert.Contains(t, out, "UPDATE alice/k-update")
	assert.Contains(t, out, "DELETE alice/k-delete")
	// Consumers themselves should be unchanged because we strip credentials
	// out of the consumer payload before diffing.
	assert.Contains(t, out, "consumers: create=0 update=0 delete=0")
	reg.Verify(t)
}

func TestConfigDiff_CredentialDeleteCascadeWithConsumer(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/consumers": true})
	// Consumer alice exists remotely with credentials, but local config has
	// no consumers. The consumer DELETE cascades to credentials server-side,
	// so the credential delete entries should be filtered out.
	reg.Register(http.MethodGet, "/apisix/admin/consumers", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"username":"alice"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/consumers/alice/credentials", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"name":"cascaded","plugins":{"key-auth":{"key":"x"}}}]
	}`))

	local := writeConfig(t, `
version: "1"
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdDiff(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	err := c.Execute()

	require.Error(t, err)
	assert.True(t, cmdutil.IsSilent(err))
	out := stdout.String()
	assert.Contains(t, out, "consumers: create=0 update=0 delete=1")
	assert.Contains(t, out, "DELETE alice")
	// Credential delete must be filtered out by cascade logic.
	assert.Contains(t, out, "credentials: create=0 update=0 delete=0")
	assert.NotContains(t, out, "DELETE alice/cascaded")
	reg.Verify(t)
}

func TestConfigDiff_StreamRoutesDisabled(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/stream_routes": true})
	reg.Register(http.MethodGet, "/apisix/admin/stream_routes", httpmock.StringResponse(http.StatusBadRequest,
		`{"message":"stream mode is disabled, can not add stream routes"}`))

	local := writeConfig(t, `
version: "1"
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdDiff(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "No differences found.")
	reg.Verify(t)
}
