package update

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/api"
	"github.com/api7/a7/pkg/httpmock"
	"github.com/api7/a7/pkg/iostreams"
)

type mockConfig struct {
	baseURL      string
	token        string
	gatewayGroup string
}

func (m *mockConfig) BaseURL() string                                 { return m.baseURL }
func (m *mockConfig) Token() string                                   { return m.token }
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

func TestUpdateCredential_JSONOutput(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/consumers/alice/credentials/cred1", httpmock.JSONResponse(`{
		"id":"cred1",
		"name":"apikey",
		"desc":"old",
		"plugins":{"key-auth":{"key":"old-key"}},
		"labels":{"env":"dev"}
	}`))
	registry.RegisterResponder(http.MethodPut, "/apisix/admin/consumers/alice/credentials/cred1", func(r *http.Request) (httpmock.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return httpmock.Response{}, err
		}
		var payload api.Credential
		if err := json.Unmarshal(body, &payload); err != nil {
			return httpmock.Response{}, err
		}
		if payload.Name != "apikey" {
			t.Fatalf("expected existing credential name to be preserved, got %q", payload.Name)
		}
		if payload.Plugins["key-auth"] == nil {
			t.Fatalf("expected existing plugins to be preserved: %+v", payload.Plugins)
		}
		if payload.Desc != "updated" {
			t.Fatalf("expected desc to be updated, got %q", payload.Desc)
		}
		return httpmock.JSONResponse(`{"id":"cred1","name":"apikey","desc":"updated","plugins":{"key-auth":{"key":"old-key"}}}`), nil
	})

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, Consumer: "alice", ID: "cred1", GatewayGroup: "gg1", Desc: "updated", DescSet: true}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	var item api.Credential
	if err := json.Unmarshal(out.Bytes(), &item); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if item.ID != "cred1" || item.Desc != "updated" {
		t.Fatalf("unexpected output: %+v", item)
	}

	registry.Verify(t)
}

func TestUpdateCredential_MissingGatewayGroup(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local"}, nil
	}, Consumer: "alice", ID: "cred1"}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "gateway group is required") {
		t.Fatalf("expected missing gateway group error, got: %v", err)
	}
}

func TestUpdateCredential_MissingConsumer(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, ID: "cred1", GatewayGroup: "gg1"}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "--consumer is required") {
		t.Fatalf("expected missing consumer error, got: %v", err)
	}
}

func TestUpdateCredential_APIError(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/consumers/alice/credentials/cred1", httpmock.JSONResponse(`{"id":"cred1","name":"apikey","plugins":{"key-auth":{}}}`))
	registry.Register(http.MethodPut, "/apisix/admin/consumers/alice/credentials/cred1", httpmock.StringResponse(http.StatusInternalServerError, `{"message":"boom"}`))

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, Consumer: "alice", ID: "cred1", GatewayGroup: "gg1"}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "API error") {
		t.Fatalf("expected api error, got: %v", err)
	}

	registry.Verify(t)
}
