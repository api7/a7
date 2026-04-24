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

func TestListServices_Table(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	responseBody := `{
		"total": 2,
		"list": [
			{"id":"s1","name":"svc-1","desc":"service one","upstream_id":"u1"},
			{"id":"s2","name":"svc-2","desc":"service two","upstream_id":"u2"}
		]
	}`
	registry.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(responseBody))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
	}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "DESCRIPTION") || !strings.Contains(output, "UPSTREAM_ID") {
		t.Fatal("table should contain expected headers")
	}
	if !strings.Contains(output, "s1") || !strings.Contains(output, "svc-1") || !strings.Contains(output, "service one") || !strings.Contains(output, "u1") {
		t.Fatal("table should contain first row data")
	}

	registry.Verify(t)
}

func TestListServices_JSON(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	responseBody := `{
		"total": 1,
		"list": [
			{"id":"s1","name":"svc-1","desc":"service one","upstream_id":"u1"}
		]
	}`
	registry.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(responseBody))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
		Output:       "json",
		GatewayGroup: "gg1",
	}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	var items []api.Service
	if err := json.Unmarshal([]byte(out.String()), &items); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if len(items) != 1 || items[0].ID != "s1" {
		t.Fatalf("unexpected JSON output: %+v", items)
	}

	registry.Verify(t)
}

func TestListServices_MissingGatewayGroup(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: ""}, nil
		},
	}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "gateway group is required") {
		t.Fatalf("expected gateway group error, got: %v", err)
	}
}

func TestListServices_APIError(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/services", httpmock.StringResponse(http.StatusInternalServerError, `{"message":"boom"}`))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
	}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("expected API 500 error, got: %v", err)
	}

	registry.Verify(t)
}
