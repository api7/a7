package list

import (
	"encoding/json"
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

func TestListConsumerGroups_Table(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/consumer_groups", httpmock.JSONResponse(`{"total":2,"list":[{"id":"cg1","desc":"first","plugins":{"key-auth":{} }},{"id":"cg2","desc":"second","plugins":{}}]}`))

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, GatewayGroup: "gg1"}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "ID") || !strings.Contains(output, "DESCRIPTION") || !strings.Contains(output, "PLUGINS") {
		t.Fatalf("table headers missing: %s", output)
	}
	if !strings.Contains(output, "cg1") || !strings.Contains(output, "first") || !strings.Contains(output, "1") {
		t.Fatalf("first row missing: %s", output)
	}
	if !strings.Contains(output, "cg2") || !strings.Contains(output, "second") || !strings.Contains(output, "0") {
		t.Fatalf("second row missing: %s", output)
	}

	registry.Verify(t)
}

func TestListConsumerGroups_JSON(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/consumer_groups", httpmock.JSONResponse(`{"total":1,"list":[{"id":"cg1","desc":"first"}]}`))

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, Output: "json", GatewayGroup: "gg1"}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	var items []api.ConsumerGroup
	if err := json.Unmarshal([]byte(out.String()), &items); err != nil {
		t.Fatalf("failed to parse json output: %v", err)
	}
	if len(items) != 1 || items[0].ID != "cg1" {
		t.Fatalf("unexpected output: %+v", items)
	}

	registry.Verify(t)
}

func TestListConsumerGroups_MissingGatewayGroup(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local"}, nil
	}}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "gateway group is required") {
		t.Fatalf("expected missing gateway group error, got: %v", err)
	}
}

func TestListConsumerGroups_APIError(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/consumer_groups", httpmock.StringResponse(http.StatusInternalServerError, `{"message":"boom"}`))

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, GatewayGroup: "gg1"}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "API error") {
		t.Fatalf("expected api error, got: %v", err)
	}

	registry.Verify(t)
}
