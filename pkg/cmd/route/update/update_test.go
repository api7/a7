package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/api7/a7/internal/config"
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

func TestUpdateRoute_Success(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/routes/r1", httpmock.JSONResponse(`{"id":"r1","name":"old-name","paths":["/old"],"service_id":"svc1"}`))
	registry.Register(http.MethodPut, "/apisix/admin/routes/r1", httpmock.JSONResponse(`{"id":"r1","name":"new-name"}`))
	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, ID: "r1", Name: "new-name", GatewayGroup: "gg1"}
	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), "new-name") {
		t.Fatalf("expected updated route output: %s", out.String())
	}
	registry.Verify(t)
}

func TestUpdateRoute_URIMapsToPathsAndPreservesCurrentRoute(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/routes/r1", httpmock.JSONResponse(`{"id":"r1","name":"old-name","uris":["/old"],"service_id":"svc1","status":1}`))
	registry.RegisterResponder(http.MethodPut, "/apisix/admin/routes/r1", func(req *http.Request) (httpmock.Response, error) {
		var payload map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			return httpmock.Response{}, fmt.Errorf("decode request body: %w", err)
		}
		if _, ok := payload["uri"]; ok {
			return httpmock.Response{}, fmt.Errorf("route update should not send uri to API7 EE: %#v", payload)
		}
		if _, ok := payload["uris"]; ok {
			return httpmock.Response{}, fmt.Errorf("route update should not preserve uris when --uri maps to paths: %#v", payload)
		}
		paths, ok := payload["paths"].([]interface{})
		if !ok || len(paths) != 1 || paths[0] != "/new" {
			return httpmock.Response{}, fmt.Errorf("expected uri to map to paths, got payload: %#v", payload)
		}
		if payload["name"] != "old-name" || payload["service_id"] != "svc1" {
			return httpmock.Response{}, fmt.Errorf("expected current route fields to be preserved, got payload: %#v", payload)
		}
		if payload["status"] != float64(0) {
			return httpmock.Response{}, fmt.Errorf("expected explicit status 0 to be sent, got payload: %#v", payload)
		}
		return httpmock.JSONResponse(`{"id":"r1","name":"old-name","paths":["/new"],"service_id":"svc1","status":0}`), nil
	})

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, ID: "r1", URI: "/new", Status: 0, StatusSet: true, GatewayGroup: "gg1"}
	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), "/new") {
		t.Fatalf("expected updated route output: %s", out.String())
	}
	registry.Verify(t)
}

func TestUpdateRoute_InvalidLabel(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, ID: "r1", GatewayGroup: "gg1", Labels: []string{"bad"}}
	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "invalid label") {
		t.Fatalf("expected invalid label error, got: %v", err)
	}
}

func TestUpdateRoute_FromFile(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPut, "/apisix/admin/routes/r2", httpmock.JSONResponse(`{"id":"r2","name":"from-file"}`))

	filePath := filepath.Join(t.TempDir(), "route.json")
	if err := os.WriteFile(filePath, []byte(`{"name":"from-file","uri":"/from-file"}`), 0o644); err != nil {
		t.Fatalf("failed to write temp route file: %v", err)
	}

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		ID:           "r2",
		File:         filePath,
		GatewayGroup: "gg1",
	}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), "from-file") {
		t.Fatalf("expected file-based updated route output: %s", out.String())
	}
	registry.Verify(t)
}

// TestUpdateRoute_DescFlag guards the regression where `route update` had no
// `--desc` flag despite the README documenting one. The new description must
// reach the PUT body.
func TestUpdateRoute_DescFlag(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/routes/r1", httpmock.JSONResponse(`{"id":"r1","name":"old-name","desc":"old desc","paths":["/demo"],"service_id":"svc1","status":1}`))
	registry.RegisterResponder(http.MethodPut, "/apisix/admin/routes/r1", func(req *http.Request) (httpmock.Response, error) {
		var payload map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			return httpmock.Response{}, fmt.Errorf("decode request body: %w", err)
		}
		if payload["desc"] != "updated desc" {
			return httpmock.Response{}, fmt.Errorf("expected updated desc in payload, got desc=%v", payload["desc"])
		}
		return httpmock.JSONResponse(`{"id":"r1","name":"old-name","desc":"updated desc","paths":["/demo"],"service_id":"svc1","status":1}`), nil
	})

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		ID:           "r1",
		GatewayGroup: "gg1",
		Desc:         "updated desc",
		DescSet:      true,
	}
	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), "updated desc") {
		t.Fatalf("expected updated desc in output: %s", out.String())
	}
	registry.Verify(t)
}

// TestUpdateRoute_DescFlagCanClear verifies that passing --desc "" explicitly
// clears the existing description (rather than being treated as "unset").
func TestUpdateRoute_DescFlagCanClear(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/routes/r1", httpmock.JSONResponse(`{"id":"r1","name":"old-name","desc":"old desc","paths":["/demo"],"service_id":"svc1","status":1}`))
	registry.RegisterResponder(http.MethodPut, "/apisix/admin/routes/r1", func(req *http.Request) (httpmock.Response, error) {
		var payload map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			return httpmock.Response{}, fmt.Errorf("decode request body: %w", err)
		}
		// With api.Route's `omitempty` desc tag, clearing should drop the field
		// from the payload entirely; an empty string would also be acceptable.
		if d, present := payload["desc"]; present && d != "" {
			return httpmock.Response{}, fmt.Errorf("expected desc to be cleared, got desc=%v", d)
		}
		return httpmock.JSONResponse(`{"id":"r1","name":"old-name","paths":["/demo"],"service_id":"svc1","status":1}`), nil
	})

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		ID:           "r1",
		GatewayGroup: "gg1",
		Desc:         "",
		DescSet:      true,
	}
	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	registry.Verify(t)
}
