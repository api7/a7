package update

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
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

func TestUpdateStreamRoute_Success(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPut, "/apisix/admin/stream_routes/sr1", httpmock.JSONResponse(`{"id":"sr1","desc":"mysql-updated","service_id":"svc2"}`))

	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return registry.GetClient(), nil },
		GatewayGroup: "gg1",
		ID:           "sr1",
		Desc:         "mysql-updated",
		ServiceID:    "svc2",
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
	})
	if err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	var item api.StreamRoute
	if err := json.Unmarshal(out.Bytes(), &item); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if item.ID != "sr1" {
		t.Fatalf("unexpected response: %+v", item)
	}

	registry.Verify(t)
}

func TestUpdateStreamRoute_MissingGatewayGroup(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	err := actionRun(&Options{
		IO:        ios,
		Client:    func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil },
		ID:        "sr1",
		ServiceID: "svc1",
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: ""}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "gateway group is required") {
		t.Fatalf("expected gateway group error, got: %v", err)
	}
}

func TestUpdateStreamRoute_MissingServiceID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil },
		GatewayGroup: "gg1",
		ID:           "sr1",
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "--service-id is required") {
		t.Fatalf("expected service-id required error, got: %v", err)
	}
}

func TestUpdateStreamRoute_FileMissingServiceID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	path := filepath.Join(t.TempDir(), "stream-route.json")
	if err := os.WriteFile(path, []byte(`{"desc":"mysql-updated"}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil },
		GatewayGroup: "gg1",
		ID:           "sr1",
		File:         path,
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "--service-id is required") {
		t.Fatalf("expected service-id required error, got: %v", err)
	}
}

func TestUpdateStreamRoute_FileNullServiceID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	path := filepath.Join(t.TempDir(), "stream-route.json")
	if err := os.WriteFile(path, []byte(`{"desc":"mysql-updated","service_id":null}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil },
		GatewayGroup: "gg1",
		ID:           "sr1",
		File:         path,
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "--service-id is required") {
		t.Fatalf("expected service-id required error, got: %v", err)
	}
}

func TestUpdateStreamRoute_APIError(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPut, "/apisix/admin/stream_routes/sr1", httpmock.StringResponse(http.StatusInternalServerError, `{"message":"boom"}`))

	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return registry.GetClient(), nil },
		GatewayGroup: "gg1",
		ID:           "sr1",
		ServiceID:    "svc1",
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("expected API error with status 500, got: %v", err)
	}

	registry.Verify(t)
}
