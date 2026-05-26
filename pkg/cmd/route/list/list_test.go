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

// TestListRoutes_Table tests table output format with 2 routes when --service-id is set
func TestListRoutes_Table(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	responseBody := `{
		"total": 2,
		"list": [
			{
				"id": "r1",
				"name": "test-route",
				"service_id": "svc1",
				"uri": "/api/v1",
				"methods": ["GET", "POST"],
				"status": 1
			},
			{
				"id": "r2",
				"name": "catch-all",
				"service_id": "svc1",
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
		ServiceID:    "svc1",
	}

	err := actionRun(opts)
	if err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	output := out.String()
	for _, header := range []string{"ID", "NAME", "SERVICE_ID", "PATHS", "METHODS", "STATUS"} {
		if !strings.Contains(output, header) {
			t.Errorf("table should contain %q header", header)
		}
	}
	for _, want := range []string{"r1", "test-route", "svc1", "/api/v1", "GET,POST", "r2", "catch-all", "/v2/*,/v3/*"} {
		if !strings.Contains(output, want) {
			t.Errorf("table should contain %q", want)
		}
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
		ServiceID:    "svc1",
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
	if routes[1].ID != "r2" {
		t.Errorf("expected second route ID 'r2', got '%s'", routes[1].ID)
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

// TestListRoutes_AcrossServices verifies that omitting --service-id triggers
// a /services lookup followed by per-service /routes calls, and that the
// merged result is rendered as a single table.
func TestListRoutes_AcrossServices(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	registry.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 2,
		"list": [
			{"id": "svc-a", "name": "service-a"},
			{"id": "svc-b", "name": "service-b"}
		]
	}`))

	registry.RegisterResponder(http.MethodGet, "/apisix/admin/routes", func(r *http.Request) (httpmock.Response, error) {
		switch r.URL.Query().Get("service_id") {
		case "svc-a":
			return httpmock.JSONResponse(`{
				"total": 1,
				"list": [
					{"id": "r-a", "name": "route-a", "service_id": "svc-a", "uri": "/a", "methods": ["GET"], "status": 1}
				]
			}`), nil
		case "svc-b":
			return httpmock.JSONResponse(`{
				"total": 1,
				"list": [
					{"id": "r-b", "name": "route-b", "service_id": "svc-b", "uri": "/b", "methods": ["POST"], "status": 1}
				]
			}`), nil
		default:
			return httpmock.JSONResponse(`{"total": 0, "list": []}`), nil
		}
	})

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		// ServiceID intentionally empty
	}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	output := out.String()
	for _, want := range []string{"r-a", "route-a", "svc-a", "/a", "r-b", "route-b", "svc-b", "/b"} {
		if !strings.Contains(output, want) {
			t.Errorf("aggregated output should contain %q\noutput:\n%s", want, output)
		}
	}

	if got := registry.CallCount(http.MethodGet, "/apisix/admin/services"); got != 1 {
		t.Errorf("expected services to be listed once, got %d calls", got)
	}
	if got := registry.CallCount(http.MethodGet, "/apisix/admin/routes"); got != 2 {
		t.Errorf("expected routes to be fetched once per service (2 calls), got %d", got)
	}
}

// TestListRoutes_AcrossServices_JSON verifies the aggregated path also feeds
// the JSON exporter correctly.
func TestListRoutes_AcrossServices_JSON(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	registry.Register(http.MethodGet, "/apisix/admin/services", httpmock.JSONResponse(`{
		"total": 1,
		"list": [{"id": "svc-a", "name": "service-a"}]
	}`))
	registry.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse(`{
		"total": 1,
		"list": [
			{"id": "r-a", "name": "route-a", "service_id": "svc-a", "uri": "/a", "methods": ["GET"], "status": 1}
		]
	}`))

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

	var routes []api.Route
	if err := json.Unmarshal([]byte(out.String()), &routes); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if len(routes) != 1 || routes[0].ID != "r-a" || routes[0].ServiceID != "svc-a" {
		t.Errorf("unexpected JSON routes: %+v", routes)
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
		ServiceID:    "svc1",
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
		GatewayGroup: "gg-from-flag",
		ServiceID:    "svc1",
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

	if callCount := registry.CallCount(http.MethodGet, "/apisix/admin/routes"); callCount != 1 {
		t.Errorf("expected mock to be called once, got %d", callCount)
	}

	registry.Verify(t)
}
