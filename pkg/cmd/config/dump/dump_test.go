package dump

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

// captureLabelsResponder returns an httpmock responder that records the
// `labels[*]` query parameters of each request into the given map (keyed by
// the inner label name) and returns an empty paginated list response. The
// responder is safe to register on multiple endpoints because all writes go
// through a shared map keyed by label name, so the assertion below is on the
// set of labels API7 EE saw, not on which endpoint asked first.
func captureLabelsResponder(captured map[string]string) func(*http.Request) (httpmock.Response, error) {
	return func(req *http.Request) (httpmock.Response, error) {
		for k, vs := range req.URL.Query() {
			if !strings.HasPrefix(k, "labels[") || !strings.HasSuffix(k, "]") {
				continue
			}
			inner := strings.TrimSuffix(strings.TrimPrefix(k, "labels["), "]")
			if len(vs) > 0 {
				captured[inner] = vs[0]
			}
		}
		return httpmock.JSONResponse(`{"total":0,"list":[]}`), nil
	}
}

func TestConfigDump_LabelSelector_PassesQueryParam(t *testing.T) {
	reg := &httpmock.Registry{}
	captured := map[string]string{}

	// Register every list endpoint with a label-capturing responder so we can
	// confirm the flag is plumbed through to *all* of them, not just one.
	for _, path := range []string{
		"/apisix/admin/services",
		"/apisix/admin/consumers",
		"/apisix/admin/ssls",
		"/apisix/admin/global_rules",
		"/apisix/admin/stream_routes",
		"/apisix/admin/protos",
		"/apisix/admin/secret_providers",
	} {
		reg.RegisterResponder(http.MethodGet, path, captureLabelsResponder(captured))
	}
	reg.Register(http.MethodGet, "/apisix/admin/plugins/list", httpmock.JSONResponse(`[]`))

	ios, _, _, _ := iostreams.Test()

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{"--output", "json", "--label-selector", "team=alpha"})
	err := c.Execute()
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"team": "alpha"}, captured)
}

func TestConfigDump_LabelSelector_Repeatable(t *testing.T) {
	reg := &httpmock.Registry{}
	captured := map[string]string{}

	for _, path := range []string{
		"/apisix/admin/services",
		"/apisix/admin/consumers",
		"/apisix/admin/ssls",
		"/apisix/admin/global_rules",
		"/apisix/admin/stream_routes",
		"/apisix/admin/protos",
		"/apisix/admin/secret_providers",
	} {
		reg.RegisterResponder(http.MethodGet, path, captureLabelsResponder(captured))
	}
	reg.Register(http.MethodGet, "/apisix/admin/plugins/list", httpmock.JSONResponse(`[]`))

	ios, _, _, _ := iostreams.Test()

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{
		"--output", "json",
		"--label-selector", "a=1",
		"--label-selector", "b=2",
	})
	err := c.Execute()
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, captured)
}

func TestConfigDump_LabelSelector_InvalidFormat(t *testing.T) {
	reg := &httpmock.Registry{}
	// No mocks should be hit: we expect dumpRun to fail before any HTTP call.

	ios, _, _, _ := iostreams.Test()

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{"--label-selector", "noequals"})
	c.SilenceErrors = true
	c.SilenceUsage = true
	err := c.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "label-selector")
	assert.Contains(t, err.Error(), "noequals")
}

// TestConfigDump_LabelSelector_PerServiceRouteFetch asserts that the label
// selector is also threaded into the per-service /apisix/admin/routes call,
// which is fetched via FetchRoutesForServices rather than the generic list
// helper. Without this, dump would over-fetch routes that don't match the
// requested label set.
func TestConfigDump_LabelSelector_PerServiceRouteFetch(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/services": true})
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"svc-1","name":"svc"}]
	}`))

	var routesLabel string
	reg.RegisterResponder(http.MethodGet, "/apisix/admin/routes", func(req *http.Request) (httpmock.Response, error) {
		if v := req.URL.Query().Get("labels[team]"); v != "" {
			routesLabel = v
		}
		return httpmock.JSONResponse(`{"total":0,"list":[]}`), nil
	})

	ios, _, _, _ := iostreams.Test()

	c := NewCmdDump(newFactory(reg, ios))
	c.SetArgs([]string{"--output", "json", "--label-selector", "team=alpha"})
	err := c.Execute()
	require.NoError(t, err)
	assert.Equal(t, "alpha", routesLabel)
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
