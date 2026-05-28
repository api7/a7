package dump

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/httpmock"
	"github.com/api7/a7/pkg/iostreams"
)

type mockConfig struct {
	baseURL      string
	gatewayGroup string
}

func (m *mockConfig) BaseURL() string                                 { return m.baseURL }
func (m *mockConfig) Token() string                                   { return "" }
func (m *mockConfig) GatewayGroup() string                            { return m.gatewayGroup }
func (m *mockConfig) TLSSkipVerify() bool                             { return false }
func (m *mockConfig) CACert() string                                  { return "" }
func (m *mockConfig) CurrentContext() string                          { return "test" }
func (m *mockConfig) Contexts() []config.Context                      { return nil }
func (m *mockConfig) GetContext(name string) (*config.Context, error) { return nil, nil }
func (m *mockConfig) AddContext(ctx config.Context) error             { return nil }
func (m *mockConfig) RemoveContext(name string) error                 { return nil }
func (m *mockConfig) SetCurrentContext(name string) error             { return nil }
func (m *mockConfig) Save() error                                     { return nil }

// registerEmptyResources registers empty list responses for all resource endpoints.
// Note: /apisix/admin/routes is NOT registered here because routes are now fetched
// per-service via fetchRoutesForServices(). Tests that need routes must also register
// services and the routes endpoint separately.
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

func TestConfigDump_RoutesOnly(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/services": true})
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"svc-1","name":"svc"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(`{
		"total": 1,
		"list": [
			{
				"id": "1",
				"name": "hello-route",
				"uri": "/hello",
				"service_id": "svc-1",
				"create_time": 1714100000,
				"update_time": 1714200000
			}
		]
	}`))

	ios, _, stdout, _ := iostreams.Test()
	ios.SetStdoutTTY(true)

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{"--output", "yaml"})
	err := c.Execute()

	require.NoError(t, err)
	out := stdout.String()
	assert.Contains(t, out, "version: \"1\"")
	assert.Contains(t, out, "routes:")
	assert.Contains(t, out, "name: hello-route")
	assert.NotContains(t, out, "create_time")
	assert.NotContains(t, out, "update_time")
	reg.Verify(t)
}

func TestConfigDump_MultipleResources(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{
		"/apisix/admin/services":         true,
		"/apisix/admin/secret_providers": true,
		"/apisix/admin/plugins/list":     true,
	})

	reg.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"1","uri":"/hello"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"1","name":"svc-1","upstream_id":"1"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/secret_providers", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"vault/my-vault","uri":"https://vault.example.com"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/plugins/list", httpmock.JSONResponse(`["limit-count"]`))
	reg.Register(http.MethodGet, "/apisix/admin/plugin_metadata/limit-count", httpmock.JSONResponse(`{
		"policy":"local"
	}`))

	ios, _, stdout, _ := iostreams.Test()

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{"--output", "json"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(stdout.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "1", result["version"])
	routes := result["routes"].([]interface{})
	assert.Len(t, routes, 1)
	services := result["services"].([]interface{})
	assert.Len(t, services, 1)

	secrets := result["secrets"].([]interface{})
	secret0 := secrets[0].(map[string]interface{})
	assert.Equal(t, "vault/my-vault", secret0["id"])

	metadata := result["plugin_metadata"].([]interface{})
	meta0 := metadata[0].(map[string]interface{})
	assert.Equal(t, "limit-count", meta0["plugin_name"])
	assert.Equal(t, "local", meta0["policy"])

	reg.Verify(t)
}

func TestConfigDump_EmptyAPI7(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, nil)

	ios, _, stdout, _ := iostreams.Test()

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{"--output", "json"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(stdout.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "1", result["version"])
	assert.NotContains(t, result, "routes")
	assert.NotContains(t, result, "services")
	assert.NotContains(t, result, "plugin_metadata")
	reg.Verify(t)
}

func TestConfigDump_YAMLOutput(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, nil)

	ios, _, stdout, _ := iostreams.Test()

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = yaml.Unmarshal(stdout.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "1", result["version"])
	reg.Verify(t)
}

func TestConfigDump_FileFlag(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/services": true})
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"svc-1","name":"svc"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"1","uri":"/hello","service_id":"svc-1"}]
	}`))

	ios, _, stdout, _ := iostreams.Test()
	outFile := filepath.Join(t.TempDir(), "dump.yaml")

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{"-f", outFile})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "", stdout.String())

	content, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "version: \"1\"")
	assert.Contains(t, string(content), "uri: /hello")
	reg.Verify(t)
}

func TestConfigDump_IncludeOnly_RoutesAndServices(t *testing.T) {
	reg := &httpmock.Registry{}
	// Only register services and routes endpoints. If the command tries to
	// GET any other resource endpoint, httpmock will fail the request and
	// the command will return an error.
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"svc-1","name":"svc"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"r1","uri":"/hello","service_id":"svc-1"}]
	}`))

	ios, _, stdout, _ := iostreams.Test()

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{"--output", "json", "--include-resource-type", "routes,services"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))

	assert.Equal(t, "1", result["version"])
	routes := result["routes"].([]interface{})
	assert.Len(t, routes, 1)
	services := result["services"].([]interface{})
	assert.Len(t, services, 1)
	assert.NotContains(t, result, "consumers")
	assert.NotContains(t, result, "ssl")
	assert.NotContains(t, result, "global_rules")
	assert.NotContains(t, result, "stream_routes")
	assert.NotContains(t, result, "protos")
	assert.NotContains(t, result, "secrets")
	assert.NotContains(t, result, "plugin_metadata")

	reg.Verify(t)
}

func TestConfigDump_ExcludeSSL_SkipsSSLEndpoint(t *testing.T) {
	reg := &httpmock.Registry{}
	// Register everything except SSL. If the command tries to GET
	// /apisix/admin/ssls it will fail because no mock is registered.
	registerEmptyResources(reg, map[string]bool{
		"/apisix/admin/ssls":     true,
		"/apisix/admin/services": true,
	})
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"svc-1","name":"svc"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(`{
		"total": 0,
		"list": []
	}`))

	ios, _, stdout, _ := iostreams.Test()

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{"--output", "json", "--exclude-resource-type", "ssl"})
	err := c.Execute()

	require.NoError(t, err)
	// Defence-in-depth: the SSL endpoint must never have been called.
	assert.Equal(t, 0, reg.CallCount(http.MethodGet, "/apisix/admin/ssls"),
		"expected no GET to /apisix/admin/ssls when ssl is excluded")

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "1", result["version"])
	assert.NotContains(t, result, "ssl")

	reg.Verify(t)
}

func TestConfigDump_IncludeAndExcludeBothSet_Errors(t *testing.T) {
	reg := &httpmock.Registry{}

	ios, _, _, _ := iostreams.Test()

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{
		"--include-resource-type", "routes",
		"--exclude-resource-type", "ssl",
	})
	c.SilenceUsage = true
	c.SilenceErrors = true
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestConfigDump_UnknownResourceType_Errors(t *testing.T) {
	reg := &httpmock.Registry{}

	ios, _, _, _ := iostreams.Test()

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{"--include-resource-type", "not_a_real_type"})
	c.SilenceUsage = true
	c.SilenceErrors = true
	err := c.Execute()

	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "unknown resource type")
	assert.Contains(t, msg, "not_a_real_type")
	// The error should mention at least one valid type to guide the user.
	assert.Contains(t, msg, "routes")
}

func TestConfigDump_StreamRoutesDisabled(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/stream_routes": true})
	reg.Register(http.MethodGet, "/apisix/admin/stream_routes", httpmock.StringResponse(http.StatusBadRequest,
		`{"message":"stream mode is disabled, can not add stream routes"}`))

	ios, _, stdout, _ := iostreams.Test()

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{"--output", "json"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "1", result["version"])
	assert.NotContains(t, result, "stream_routes")
	reg.Verify(t)
}
