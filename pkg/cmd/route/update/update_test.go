package update

import (
	"encoding/json"
	"io"
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
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.Method {
		case http.MethodGet:
			if req.URL.Path != "/apisix/admin/routes/r1" {
				t.Fatalf("unexpected GET path: %s", req.URL.Path)
			}
			return jsonHTTPResponse(`{"id":"r1","name":"old-name","paths":["/old"],"service_id":"svc1","status":1}`), nil
		case http.MethodPut:
			var payload map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if _, ok := payload["uri"]; ok {
				t.Fatalf("route update should not send uri to API7 EE: %#v", payload)
			}
			paths, ok := payload["paths"].([]interface{})
			if !ok || len(paths) != 1 || paths[0] != "/new" {
				t.Fatalf("expected uri to map to paths, got payload: %#v", payload)
			}
			if payload["name"] != "old-name" || payload["service_id"] != "svc1" {
				t.Fatalf("expected current route fields to be preserved, got payload: %#v", payload)
			}
			if payload["status"] != float64(0) {
				t.Fatalf("expected explicit status 0 to be sent, got payload: %#v", payload)
			}
			return jsonHTTPResponse(`{"id":"r1","name":"old-name","paths":["/new"],"service_id":"svc1","status":0}`), nil
		default:
			t.Fatalf("unexpected method: %s", req.Method)
			return nil, nil
		}
	})}

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return client, nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, ID: "r1", URI: "/new", Status: 0, StatusSet: true, GatewayGroup: "gg1"}
	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), "/new") {
		t.Fatalf("expected updated route output: %s", out.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
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
