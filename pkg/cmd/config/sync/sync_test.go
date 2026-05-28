package sync

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func TestConfigSync_CreatesNewResources(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, nil)
	reg.RegisterResponder(http.MethodPut, "/apisix/admin/routes/r1", func(r *http.Request) (httpmock.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return httpmock.Response{}, err
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			return httpmock.Response{}, err
		}
		if payload["service_id"] != "svc-1" {
			t.Fatalf("expected service_id svc-1 in route payload, got: %v", payload["service_id"])
		}
		return httpmock.JSONResponse(`{"id":"r1"}`), nil
	})

	local := writeConfig(t, `
version: "1"
routes:
  - id: r1
    uri: /sync
    service_id: svc-1
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdSync(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, 1, reg.CallCount(http.MethodPut, "/apisix/admin/routes/r1"))
	assert.Contains(t, stdout.String(), "Sync completed")
	reg.Verify(t)
}

func TestConfigSync_UpdatesExistingResources(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/services": true})
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"svc-1","name":"svc"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(`{
		"total":1,
		"list":[{"id":"r1","uri":"/old","name":"old","service_id":"svc-1"}]
	}`))
	reg.Register(http.MethodPut, "/apisix/admin/routes/r1", httpmock.JSONResponse(`{"id":"r1"}`))

	local := writeConfig(t, `
version: "1"
services:
  - id: svc-1
    name: svc
routes:
  - id: r1
    uri: /new
    name: new
    service_id: svc-1
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdSync(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, 1, reg.CallCount(http.MethodPut, "/apisix/admin/routes/r1"))
	assert.Contains(t, stdout.String(), "updated=1")
	reg.Verify(t)
}

func TestConfigSync_DeletesRemoteOnlyResources(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/services": true})
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"svc-1","name":"svc"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(`{
		"total":1,
		"list":[{"id":"r-del","uri":"/gone"}]
	}`))
	reg.Register(http.MethodDelete, "/apisix/admin/routes/r-del", httpmock.JSONResponse(`{"message":"deleted"}`))
	reg.Register(http.MethodDelete, "/apisix/admin/services/svc-1", httpmock.JSONResponse(`{"message":"deleted"}`))

	local := writeConfig(t, `
version: "1"
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdSync(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, 1, reg.CallCount(http.MethodDelete, "/apisix/admin/routes/r-del"))
	assert.Contains(t, stdout.String(), "deleted=1")
	reg.Verify(t)
}

func TestConfigSync_DryRunDoesNotApply(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, nil)

	local := writeConfig(t, `
version: "1"
routes:
  - id: r1
    uri: /sync
    service_id: svc-1
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdSync(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local, "--dry-run"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, 0, reg.CallCount(http.MethodPut, "/apisix/admin/routes/r1"))
	assert.Contains(t, stdout.String(), "Differences found")
	assert.Contains(t, stdout.String(), "CREATE r1")
	reg.Verify(t)
}

func TestConfigSync_DeleteFalseSkipsDeletion(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/services": true})
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id":"svc-1","name":"svc"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(`{
		"total":1,
		"list":[{"id":"r-del","uri":"/gone"}]
	}`))

	local := writeConfig(t, `
version: "1"
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdSync(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local, "--delete=false"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, 0, reg.CallCount(http.MethodDelete, "/apisix/admin/routes/r-del"))
	assert.Contains(t, stdout.String(), "deleted=0")
	reg.Verify(t)
}

func TestConfigSync_Credential_RoundTrip(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/consumers": true})
	// Remote: consumer alice with two credentials; one will be updated, one
	// will be deleted because it is not in the local config.
	reg.Register(http.MethodGet, "/apisix/admin/consumers", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"username":"alice"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/consumers/alice/credentials", httpmock.JSONResponse(`{
		"total": 2,
		"list": [
			{"name":"k-update","plugins":{"key-auth":{"key":"old"}}},
			{"name":"k-delete","plugins":{"key-auth":{"key":"gone"}}}
		]
	}`))

	// Capture the PUT bodies so we can assert the payload is clean and the
	// path follows /apisix/admin/consumers/{user}/credentials/{name}.
	reg.RegisterResponder(http.MethodPut, "/apisix/admin/consumers/alice/credentials/k-create", func(r *http.Request) (httpmock.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return httpmock.Response{}, err
		}
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.NotContains(t, payload, "_consumer_username")
		assert.NotContains(t, payload, "_diff_key")
		assert.Equal(t, "k-create", payload["name"])
		return httpmock.JSONResponse(`{"name":"k-create"}`), nil
	})
	reg.RegisterResponder(http.MethodPut, "/apisix/admin/consumers/alice/credentials/k-update", func(r *http.Request) (httpmock.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return httpmock.Response{}, err
		}
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &payload))
		plugins, ok := payload["plugins"].(map[string]interface{})
		require.True(t, ok)
		keyAuth, ok := plugins["key-auth"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "new", keyAuth["key"])
		return httpmock.JSONResponse(`{"name":"k-update"}`), nil
	})
	reg.Register(http.MethodDelete, "/apisix/admin/consumers/alice/credentials/k-delete", httpmock.JSONResponse(`{"message":"deleted"}`))
	// Consumers themselves are unchanged because credentials are stripped
	// out of the consumer payload before diffing, so no consumer PUT/DELETE
	// is expected here.

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
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdSync(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	require.NoError(t, c.Execute())

	assert.Equal(t, 1, reg.CallCount(http.MethodPut, "/apisix/admin/consumers/alice/credentials/k-create"))
	assert.Equal(t, 1, reg.CallCount(http.MethodPut, "/apisix/admin/consumers/alice/credentials/k-update"))
	assert.Equal(t, 1, reg.CallCount(http.MethodDelete, "/apisix/admin/consumers/alice/credentials/k-delete"))
	assert.Contains(t, stdout.String(), "credentials: created=1 updated=1 deleted=1")
	reg.Verify(t)
}

func TestConfigSync_CredentialCascadeOnConsumerDelete(t *testing.T) {
	reg := &httpmock.Registry{}
	registerEmptyResources(reg, map[string]bool{"/apisix/admin/consumers": true})
	// Remote consumer alice with one credential. Local config has nothing,
	// so alice (and her credential) must be removed. The server cascades the
	// credential delete, so the sync layer should NOT issue an explicit
	// DELETE for the credential.
	reg.Register(http.MethodGet, "/apisix/admin/consumers", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"username":"alice"}]
	}`))
	reg.Register(http.MethodGet, "/apisix/admin/consumers/alice/credentials", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"name":"cascaded","plugins":{"key-auth":{"key":"x"}}}]
	}`))
	reg.Register(http.MethodDelete, "/apisix/admin/consumers/alice", httpmock.JSONResponse(`{"message":"deleted"}`))

	local := writeConfig(t, `
version: "1"
`)

	ios, _, stdout, _ := iostreams.Test()
	c := NewCmdSync(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	require.NoError(t, c.Execute())

	assert.Equal(t, 1, reg.CallCount(http.MethodDelete, "/apisix/admin/consumers/alice"))
	assert.Equal(t, 0, reg.CallCount(http.MethodDelete, "/apisix/admin/consumers/alice/credentials/cascaded"))
	assert.Contains(t, stdout.String(), "consumers: created=0 updated=0 deleted=1")
	assert.Contains(t, stdout.String(), "credentials: created=0 updated=0 deleted=0")
	reg.Verify(t)
}

func TestConfigSync_ValidationFailureStopsSync(t *testing.T) {
	reg := &httpmock.Registry{}
	ios, _, _, _ := iostreams.Test()

	local := writeConfig(t, `
version: "1"
routes:
  - id: bad-route
`)

	c := NewCmdSync(newFactory(reg, ios))
	c.SetArgs([]string{"-f", local})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "validation failed")
	assert.Contains(t, err.Error(), "either uri or uris is required")
}
