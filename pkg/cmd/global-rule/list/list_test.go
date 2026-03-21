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

func TestListGlobalRules_Table(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	responseBody := `{"total":2,"list":[{"id":"1","plugins":{"limit-count":{},"cors":{}}},{"id":"2","plugins":{"proxy-rewrite":{}}}]}`
	registry.Register(http.MethodGet, "/apisix/admin/global_rules", httpmock.JSONResponse(responseBody))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
	}

	if err := listRun(opts); err != nil {
		t.Fatalf("listRun failed: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "ID") || !strings.Contains(output, "PLUGINS") || !strings.Contains(output, "1") || !strings.Contains(output, "2") {
		t.Fatalf("unexpected table output: %s", output)
	}
	registry.Verify(t)
}

func TestListGlobalRules_JSON(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	responseBody := `{"total":1,"list":[{"id":"1","plugins":{"limit-count":{}}}]}`
	registry.Register(http.MethodGet, "/apisix/admin/global_rules", httpmock.JSONResponse(responseBody))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		Output:       "json",
		GatewayGroup: "gg1",
	}

	if err := listRun(opts); err != nil {
		t.Fatalf("listRun failed: %v", err)
	}
	var items []api.GlobalRule
	if err := json.Unmarshal([]byte(out.String()), &items); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if len(items) != 1 || items[0].ID != "1" {
		t.Fatalf("unexpected JSON output: %+v", items)
	}
	registry.Verify(t)
}

func TestListGlobalRules_MissingGatewayGroup(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	err := listRun(&Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) { return &mockConfig{baseURL: "http://api.local", gatewayGroup: ""}, nil },
	})
	if err == nil || !strings.Contains(err.Error(), "gateway group is required") {
		t.Fatalf("expected gateway group error, got: %v", err)
	}
}

func TestListGlobalRules_APIError(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/global_rules", httpmock.StringResponse(http.StatusInternalServerError, `{"message":"boom"}`))

	err := listRun(&Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
	})
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("expected API 500 error, got: %v", err)
	}
	registry.Verify(t)
}
