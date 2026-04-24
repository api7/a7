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

func TestListPluginConfigs_Table(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/plugin_configs", httpmock.JSONResponse(`{
		"total": 2,
		"list": [
			{"id":"pc1","desc":"auth", "plugins":{"key-auth":{},"limit-count":{}}},
			{"id":"pc2","desc":"cors", "plugins":{"cors":{}}}
		]
	}`))

	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return registry.GetClient(), nil },
		GatewayGroup: "gg1",
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
	})
	if err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	output := out.String()
	for _, expect := range []string{"ID", "DESCRIPTION", "PLUGINS", "pc1", "auth", "2", "pc2", "cors", "1"} {
		if !strings.Contains(output, expect) {
			t.Fatalf("expected %q in output: %q", expect, output)
		}
	}

	registry.Verify(t)
}

func TestListPluginConfigs_JSON(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/plugin_configs", httpmock.JSONResponse(`{"total":1,"list":[{"id":"pc1","desc":"auth","plugins":{"key-auth":{}}}]}`))

	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return registry.GetClient(), nil },
		Output:       "json",
		GatewayGroup: "gg1",
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
	})
	if err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	var items []api.PluginConfig
	if err := json.Unmarshal([]byte(out.String()), &items); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if len(items) != 1 || items[0].ID != "pc1" {
		t.Fatalf("unexpected items: %+v", items)
	}

	registry.Verify(t)
}

func TestListPluginConfigs_MissingGatewayGroup(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	err := actionRun(&Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: ""}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "gateway group is required") {
		t.Fatalf("expected gateway group error, got: %v", err)
	}
}

func TestListPluginConfigs_APIError(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/plugin_configs", httpmock.StringResponse(http.StatusInternalServerError, `{"message":"boom"}`))

	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return registry.GetClient(), nil },
		GatewayGroup: "gg1",
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("expected API error with status 500, got: %v", err)
	}

	registry.Verify(t)
}
