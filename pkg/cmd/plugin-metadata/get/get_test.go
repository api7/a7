package get

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

func TestGetPluginMetadata_Table(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/plugin_metadata/key-auth", httpmock.JSONResponse(`{"id":"key-auth","metadata":{"header":"x-api-key"}}`))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		PluginName:   "key-auth",
	}
	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	outStr := out.String()
	if !strings.Contains(outStr, "FIELD") || !strings.Contains(outStr, "VALUE") || !strings.Contains(outStr, "key-auth") {
		t.Fatalf("unexpected output: %s", outStr)
	}
	registry.Verify(t)
}

func TestGetPluginMetadata_JSON(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/plugin_metadata/key-auth", httpmock.JSONResponse(`{"id":"key-auth","metadata":{"header":"x-api-key"}}`))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		Output:       "json",
		GatewayGroup: "gg1",
		PluginName:   "key-auth",
	}
	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	var item api.PluginMetadata
	if err := json.Unmarshal([]byte(out.String()), &item); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if item.ID != "key-auth" {
		t.Fatalf("unexpected item: %+v", item)
	}
	registry.Verify(t)
}
