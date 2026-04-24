package create

import (
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

func TestCreateRoute_Success(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPost, "/apisix/admin/routes", httpmock.JSONResponse(`{"id":"r1","name":"demo","uri":"/demo"}`))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		URI:          "/demo",
		Name:         "demo",
	}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), "\"id\": \"r1\"") {
		t.Fatalf("expected created route in output, got: %s", out.String())
	}
	registry.Verify(t)
}

func TestCreateRoute_MissingURI(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
	}
	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "--uri is required") {
		t.Fatalf("expected uri required error, got: %v", err)
	}
}

func TestCreateRoute_FromFile(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPost, "/apisix/admin/routes", httpmock.JSONResponse(`{"id":"r-file","name":"demo-file","uri":"/demo-file"}`))

	tmp := t.TempDir()
	path := filepath.Join(tmp, "route.json")
	if err := os.WriteFile(path, []byte(`{"name":"demo-file","uri":"/demo-file"}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		File:         path,
	}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), "\"id\": \"r-file\"") {
		t.Fatalf("expected created route in output, got: %s", out.String())
	}
	registry.Verify(t)
}

func TestCreateRoute_FromYAMLFile(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPost, "/apisix/admin/routes", httpmock.JSONResponse(`{"id":"r-yaml","name":"demo-yaml","uri":"/demo-yaml"}`))

	tmp := t.TempDir()
	path := filepath.Join(tmp, "route.yaml")
	if err := os.WriteFile(path, []byte("name: demo-yaml\nuri: /demo-yaml\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		File:         path,
	}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), "\"id\": \"r-yaml\"") {
		t.Fatalf("expected created route in output, got: %s", out.String())
	}
	registry.Verify(t)
}
