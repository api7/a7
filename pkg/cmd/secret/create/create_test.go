package create

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/api"
	cmd "github.com/api7/a7/pkg/cmd"
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

func TestCreateSecret_JSON(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPut, "/apisix/admin/secret_providers/vault/s1", httpmock.JSONResponse(`{"id":"vault/s1","uri":"http://vault","prefix":"kv","token":"tok"}`))

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		ID:           "vault/s1",
		URI:          "http://vault",
		Prefix:       "kv",
		Token:        "tok",
		Labels:       []string{"env=prod"},
	}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	var item api.Secret
	if err := json.Unmarshal(out.Bytes(), &item); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if item.ID != "vault/s1" || item.URI != "http://vault" || item.Token != api.RedactedSecretToken {
		t.Fatalf("unexpected item: %+v", item)
	}

	registry.Verify(t)
}

func TestCreateCommandUsesProviderTokenFlag(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	c := NewCmd(&cmd.Factory{IOStreams: ios})

	if c.Flags().Lookup("provider-token") == nil {
		t.Fatal("expected provider-token flag")
	}
	if c.Flags().Lookup("token") != nil {
		t.Fatal("secret create must not define a local token flag that shadows the global API token")
	}
}

func TestCreateSecret_FileUsesPositionalID(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPut, "/apisix/admin/secret_providers/vault/s1", httpmock.JSONResponse(`{"id":"vault/s1","uri":"http://vault","prefix":"kv"}`))

	file := filepath.Join(t.TempDir(), "secret.json")
	if err := os.WriteFile(file, []byte(`{"uri":"http://vault","prefix":"kv","token":"tok"}`), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		ID:           "vault/s1",
		File:         file,
	}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	var item api.Secret
	if err := json.Unmarshal(out.Bytes(), &item); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if item.ID != "vault/s1" {
		t.Fatalf("expected positional id in output, got: %+v", item)
	}

	registry.Verify(t)
}

func TestCreateSecret_FileRequiresID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	file := filepath.Join(t.TempDir(), "secret.json")
	if err := os.WriteFile(file, []byte(`{"uri":"http://vault","prefix":"kv","token":"tok"}`), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	err := actionRun(&Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		File:         file,
	})
	if err == nil || err.Error() != "secret provider id is required; use a positional arg or --id" {
		t.Fatalf("expected --id required error, got: %v", err)
	}

	registry.Verify(t)
}

func TestCreateSecret_FileRejectsWhitespaceID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	file := filepath.Join(t.TempDir(), "secret.json")
	if err := os.WriteFile(file, []byte(`{"id":"   ","uri":"http://vault","prefix":"kv","token":"tok"}`), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	err := actionRun(&Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		File:         file,
	})
	if err == nil || err.Error() != "secret provider id is required; use a positional arg or --id" {
		t.Fatalf("expected --id required error, got: %v", err)
	}

	registry.Verify(t)
}

func TestCreateSecret_RequiresID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}

	err := actionRun(&Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
	})
	if err == nil || err.Error() != "secret provider id is required; use a positional arg or --id" {
		t.Fatalf("expected --id required error, got: %v", err)
	}
}
