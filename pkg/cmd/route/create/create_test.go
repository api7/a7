package create

import (
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

func TestCreateRoute_Success(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPost, "/apisix/admin/routes", httpmock.JSONResponse(`{"id":"r1","name":"demo","uri":"/demo","service_id":"svc1"}`))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		URI:          "/demo",
		Name:         "demo",
		ServiceID:    "svc1",
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

func TestCreateRoute_MissingServiceID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		URI:          "/demo",
		Name:         "demo",
	}
	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "--service-id is required") {
		t.Fatalf("expected service-id required error, got: %v", err)
	}
}

func TestCreateRoute_FromFile(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPost, "/apisix/admin/routes", httpmock.JSONResponse(`{"id":"r-file","name":"demo-file","uri":"/demo-file","service_id":"svc1"}`))

	tmp := t.TempDir()
	path := filepath.Join(tmp, "route.json")
	if err := os.WriteFile(path, []byte(`{"name":"demo-file","uri":"/demo-file","service_id":"svc1"}`), 0o600); err != nil {
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

func TestCreateRoute_FileMissingServiceID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "route.json")
	if err := os.WriteFile(path, []byte(`{"name":"demo-file","uri":"/demo-file"}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		File:         path,
	}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "--service-id is required") {
		t.Fatalf("expected service-id required error, got: %v", err)
	}
}

func TestCreateRoute_FileNullServiceID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "route.json")
	if err := os.WriteFile(path, []byte(`{"name":"demo-file","uri":"/demo-file","service_id":null}`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		File:         path,
	}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "--service-id is required") {
		t.Fatalf("expected service-id required error, got: %v", err)
	}
}

func TestCreateRoute_FileServiceIDFlag(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.RegisterResponder(http.MethodPost, "/apisix/admin/routes", func(r *http.Request) (httpmock.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return httpmock.Response{}, err
		}
		if !strings.Contains(string(body), `"service_id":"svc-flag"`) {
			t.Fatalf("expected injected service_id in request body, got: %s", string(body))
		}
		return httpmock.JSONResponse(`{"id":"r-file","service_id":"svc-flag"}`), nil
	})

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
		ServiceID:    "svc-flag",
	}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), "\"service_id\": \"svc-flag\"") {
		t.Fatalf("expected created route in output, got: %s", out.String())
	}
	registry.Verify(t)
}

func TestCreateRoute_FromYAMLFile(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPost, "/apisix/admin/routes", httpmock.JSONResponse(`{"id":"r-yaml","name":"demo-yaml","uri":"/demo-yaml","service_id":"svc1"}`))

	tmp := t.TempDir()
	path := filepath.Join(tmp, "route.yaml")
	if err := os.WriteFile(path, []byte("name: demo-yaml\nuri: /demo-yaml\nservice_id: svc1\n"), 0o600); err != nil {
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

// TestCreateRoute_DescFlag guards the regression where flag-based `route create`
// had no `--desc` flag despite the README documenting one. The description must
// reach the API request body.
func TestCreateRoute_DescFlag(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.RegisterResponder(http.MethodPost, "/apisix/admin/routes", func(r *http.Request) (httpmock.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return httpmock.Response{}, err
		}
		if !strings.Contains(string(body), `"desc":"my description"`) {
			t.Fatalf("expected --desc to land in request body, got: %s", string(body))
		}
		return httpmock.JSONResponse(`{"id":"r-desc","name":"demo","desc":"my description","service_id":"svc1"}`), nil
	})

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		URI:          "/demo",
		Name:         "demo",
		Desc:         "my description",
		ServiceID:    "svc1",
	}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), "my description") {
		t.Fatalf("expected desc in output, got: %s", out.String())
	}
	registry.Verify(t)
}
