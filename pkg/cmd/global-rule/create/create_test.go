package create

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

func TestGlobalRule_CreateWithoutID(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPost, "/apisix/admin/global_rules", httpmock.JSONResponse(`{"id":"cors","plugins":{"cors":{}}}`))

	err := actionRun(&Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		PluginsJSON:  `{"cors":{}}`,
		Output:       "json",
	})
	if err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	var created api.GlobalRule
	if err := json.Unmarshal(out.Bytes(), &created); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if created.ID != "cors" {
		t.Fatalf("expected id derived from plugin name 'cors', got: %q", created.ID)
	}
	registry.Verify(t)
}

func TestGlobalRule_CreateFromFileRejectsID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "global-rule.json")
	if err := os.WriteFile(path, []byte(`{"id":"smoke-gr-X","plugins":{"cors":{}}}`), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	err := actionRun(&Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		File:         path,
	})
	if err == nil || !strings.Contains(err.Error(), `"id" must not be set`) {
		t.Fatalf(`expected "id" must not be set error, got: %v`, err)
	}
	registry.Verify(t)
}

func TestGlobalRule_CreateRequiresPayload(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	err := actionRun(&Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
	})
	if err == nil || !strings.Contains(err.Error(), "one of --file or --plugins-json is required") {
		t.Fatalf("expected payload required error, got: %v", err)
	}
}

func TestGlobalRule_CreateMissingGatewayGroup(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return registry.GetClient(), nil },
		Config:       func() (config.Config, error) { return &mockConfig{baseURL: "http://api.local"}, nil },
		GatewayGroup: "",
		PluginsJSON:  `{"cors":{}}`,
	})
	if err == nil || !strings.Contains(err.Error(), "gateway group is required") {
		t.Fatalf("expected gateway group error, got: %v", err)
	}
}
