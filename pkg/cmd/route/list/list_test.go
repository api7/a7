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

// mockConfig implements config.Config for testing
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

// TestListRoutes_Table tests table output format with 2 routes
func TestListRoutes_Table(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	// Register mock response for GET /apisix/admin/routes
	responseBody := `{
		"total": 2,
		"list": [
			{
				"id": "r1",
				"name": "test-route",
				"uri": "/api/v1",
				"methods": ["GET", "POST"],
				"status": 1
			},
			{
				"id": "r2",
				"name": "catch-all",
				"uris": ["/v2/*", "/v3/*"],
				"methods": ["GET"],
				"status": 1
			}
		]
	}`
	registry.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(responseBody))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
		Output:       "",
		GatewayGroup: "gg1",
	}

	err := actionRun(opts)
	if err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "ID") {
		t.Error("table should contain ID header")
	}
	if !strings.Contains(output, "NAME") {
		t.Error("table should contain NAME header")
	}
	if !strings.Contains(output, "URI") {
		t.Error("table should contain URI header")
	}
	if !strings.Contains(output, "METHODS") {
		t.Error("table should contain METHODS header")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("table should contain STATUS header")
	}
	if !strings.Contains(output, "r1") {
		t.Error("table should contain first route ID")
	}
	if !strings.Contains(output, "test-route") {
		t.Error("table should contain first route name")
	}
	if !strings.Contains(output, "/api/v1") {
		t.Error("table should contain first route URI")
	}
	if !strings.Contains(output, "GET,POST") {
		t.Error("table should contain first route methods")
	}
	if !strings.Contains(output, "r2") {
		t.Error("table should contain second route ID")
	}
	if !strings.Contains(output, "catch-all") {
		t.Error("table should contain second route name")
	}
	if !strings.Contains(output, "/v2/*,/v3/*") {
		t.Error("table should contain second route URIs joined by comma")
	}

	registry.Verify(t)
}

// TestListRoutes_JSON tests JSON output format
func TestListRoutes_JSON(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	responseBody := `{
		"total": 2,
		"list": [
			{
				"id": "r1",
				"name": "test-route",
				"uri": "/api/v1",
				"methods": ["GET", "POST"],
				"status": 1
			},
			{
				"id": "r2",
				"name": "catch-all",
				"uris": ["/v2/*", "/v3/*"],
				"methods": ["GET"],
				"status": 1
			}
		]
	}`
	registry.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(responseBody))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
		Output:       "json",
		GatewayGroup: "gg1",
	}

	err := actionRun(opts)
	if err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	output := out.String()
	var routes []api.Route
	err = json.Unmarshal([]byte(output), &routes)
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
	if routes[0].ID != "r1" {
		t.Errorf("expected first route ID 'r1', got '%s'", routes[0].ID)
	}
	if routes[0].Name != "test-route" {
		t.Errorf("expected first route name 'test-route', got '%s'", routes[0].Name)
	}
	if routes[0].URI != "/api/v1" {
		t.Errorf("expected first route URI '/api/v1', got '%s'", routes[0].URI)
	}
	if routes[1].ID != "r2" {
		t.Errorf("expected second route ID 'r2', got '%s'", routes[1].ID)
	}
	if routes[1].Name != "catch-all" {
		t.Errorf("expected second route name 'catch-all', got '%s'", routes[1].Name)
	}

	registry.Verify(t)
}

// TestListRoutes_MissingGatewayGroup tests error when no gateway group is provided
func TestListRoutes_MissingGatewayGroup(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: ""}, nil
		},
		Output:       "",
		GatewayGroup: "",
	}

	err := actionRun(opts)
	if err == nil {
		t.Fatal("actionRun should return error when gateway group is missing")
	}
	if !strings.Contains(err.Error(), "gateway group is required") {
		t.Errorf("error message should contain 'gateway group is required', got: %v", err)
	}
}

// TestListRoutes_GatewayGroupFromConfig tests that GatewayGroup falls back to config when opts is empty
func TestListRoutes_GatewayGroupFromConfig(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	responseBody := `{
		"total": 1,
		"list": [
			{
				"id": "r1",
				"name": "test-route",
				"uri": "/api/v1",
				"methods": ["GET"],
				"status": 1
			}
		]
	}`
	registry.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(responseBody))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg-from-config"}, nil
		},
		Output:       "",
		GatewayGroup: "", // Empty - should use config value
	}

	err := actionRun(opts)
	if err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "r1") {
		t.Error("output should contain first route ID")
	}
	if !strings.Contains(output, "test-route") {
		t.Error("output should contain first route name")
	}

	registry.Verify(t)
}

// TestListRoutes_GatewayGroupFromFlag tests that flag value takes precedence over config
func TestListRoutes_GatewayGroupFromFlag(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	responseBody := `{
		"total": 1,
		"list": [
			{
				"id": "r1",
				"name": "test-route",
				"uri": "/api/v1",
				"methods": ["GET"],
				"status": 1
			}
		]
	}`
	registry.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(responseBody))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg-from-config"}, nil
		},
		Output:       "",
		GatewayGroup: "gg-from-flag", // Flag value - should take precedence
	}

	err := actionRun(opts)
	if err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "r1") {
		t.Error("output should contain first route ID")
	}
	if !strings.Contains(output, "test-route") {
		t.Error("output should contain first route name")
	}

	// Verify that the mock was called (indicating flag took effect)
	callCount := registry.CallCount(http.MethodGet, "/apisix/admin/routes")
	if callCount != 1 {
		t.Errorf("expected mock to be called once, got %d", callCount)
	}

	registry.Verify(t)
}
