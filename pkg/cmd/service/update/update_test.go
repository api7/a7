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

func TestUpdateService_Success(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/services/s1", httpmock.JSONResponse(`{"id":"s1","name":"svc-1","desc":"d1","upstream_id":"u1"}`))
	registry.Register(http.MethodPut, "/apisix/admin/services/s1", httpmock.JSONResponse(`{"id":"s1","name":"svc-1-updated","desc":"d2","upstream_id":"u2"}`))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		ID:           "s1",
		Name:         "svc-1-updated",
		Desc:         "d2",
		UpstreamID:   "u2",
		Labels:       []string{"k=v"},
	}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	var item api.Service
	if err := json.Unmarshal(out.Bytes(), &item); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if item.ID != "s1" {
		t.Fatalf("unexpected response: %+v", item)
	}

	registry.Verify(t)
}

func TestUpdateService_PreservesCurrentNameWhenOmitted(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.Method {
		case http.MethodGet:
			return jsonHTTPResponse(`{"id":"s1","name":"svc-1","desc":"old"}`), nil
		case http.MethodPut:
			var payload map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if payload["name"] != "svc-1" || payload["desc"] != "new" {
				t.Fatalf("expected update to preserve existing name and apply desc, got payload: %#v", payload)
			}
			return jsonHTTPResponse(`{"id":"s1","name":"svc-1","desc":"new"}`), nil
		default:
			t.Fatalf("unexpected method: %s", req.Method)
			return nil, nil
		}
	})}

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return client, nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, ID: "s1", Desc: "new", GatewayGroup: "gg1"}
	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), `"name": "svc-1"`) {
		t.Fatalf("expected preserved service name in output: %s", out.String())
	}
}

func TestUpdateService_MissingGatewayGroup(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	err := actionRun(&Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) { return &mockConfig{baseURL: "http://api.local", gatewayGroup: ""}, nil },
		ID:     "s1",
	})
	if err == nil || !strings.Contains(err.Error(), "gateway group is required") {
		t.Fatalf("expected gateway group error, got: %v", err)
	}
}

func TestUpdateService_APIError(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/services/s1", httpmock.JSONResponse(`{"id":"s1","name":"svc-1"}`))
	registry.Register(http.MethodPut, "/apisix/admin/services/s1", httpmock.StringResponse(http.StatusInternalServerError, `{"message":"boom"}`))

	err := actionRun(&Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		ID:           "s1",
	})
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("expected API 500 error, got: %v", err)
	}
	registry.Verify(t)
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

func TestUpdateService_ValidationError(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	err := actionRun(&Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		ID:           "s1",
		Labels:       []string{"broken"},
	})
	if err == nil || !strings.Contains(err.Error(), "expected key=value") {
		t.Fatalf("expected labels validation error, got: %v", err)
	}
}
