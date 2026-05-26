package create

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

func TestCreateCredential_JSONOutput(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPost, "/apisix/admin/consumers/alice/credentials", httpmock.JSONResponse(`{"id":"cred1","desc":"first","plugins":{"key-auth":{}},"labels":{"env":"dev"}}`))

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, Consumer: "alice", GatewayGroup: "gg1", Desc: "first", PluginsJSON: `{"key-auth":{}}`, Labels: []string{"env=dev"}}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}

	var item api.Credential
	if err := json.Unmarshal(out.Bytes(), &item); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if item.ID != "cred1" || item.Desc != "first" {
		t.Fatalf("unexpected output: %+v", item)
	}

	registry.Verify(t)
}

func TestCreateCredential_PositionalForwardsAsIDViaPUT(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.RegisterResponder(http.MethodPut, "/apisix/admin/consumers/alice/credentials/cred1", func(req *http.Request) (httpmock.Response, error) {
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			return httpmock.Response{}, fmt.Errorf("decode request body: %w", err)
		}
		if _, ok := payload["id"]; ok {
			return httpmock.Response{}, fmt.Errorf("expected id to be carried in URL, not body, got: %#v", payload)
		}
		return httpmock.JSONResponse(`{"id":"cred1","name":"key-auth"}`), nil
	})

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, Consumer: "alice", GatewayGroup: "gg1", ID: "cred1", Name: "key-auth", PluginsJSON: `{"key-auth":{"key":"k"}}`}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	var item api.Credential
	if err := json.Unmarshal(out.Bytes(), &item); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if item.ID != "cred1" {
		t.Fatalf("expected credential id from server response, got %+v", item)
	}
	registry.Verify(t)
}

func TestCreateCredential_NameFlagPOSTs(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.RegisterResponder(http.MethodPost, "/apisix/admin/consumers/alice/credentials", func(req *http.Request) (httpmock.Response, error) {
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			return httpmock.Response{}, fmt.Errorf("decode request body: %w", err)
		}
		if payload["name"] != "display-name" {
			return httpmock.Response{}, fmt.Errorf("expected --name to populate name, got: %#v", payload)
		}
		return httpmock.JSONResponse(`{"id":"generated-uuid","name":"display-name"}`), nil
	})

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, Consumer: "alice", GatewayGroup: "gg1", Name: "display-name", PluginsJSON: `{"key-auth":{"key":"k"}}`}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	var item api.Credential
	if err := json.Unmarshal(out.Bytes(), &item); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if item.ID != "generated-uuid" || item.Name != "display-name" {
		t.Fatalf("expected server-generated id and explicit name, got %+v", item)
	}
	registry.Verify(t)
}

func TestCreateCredential_FileIDForwardedViaPUT(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	tmpFile := t.TempDir() + "/credential.json"
	if err := os.WriteFile(tmpFile, []byte(`{"id":"cred-file","plugins":{"key-auth":{"key":"k"}}}`), 0o644); err != nil {
		t.Fatalf("write credential file: %v", err)
	}

	registry := &httpmock.Registry{}
	registry.RegisterResponder(http.MethodPut, "/apisix/admin/consumers/alice/credentials/cred-file", func(req *http.Request) (httpmock.Response, error) {
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			return httpmock.Response{}, fmt.Errorf("decode request body: %w", err)
		}
		if _, ok := payload["id"]; ok {
			return httpmock.Response{}, fmt.Errorf("expected id to be stripped from body when carried in URL, got: %#v", payload)
		}
		return httpmock.JSONResponse(`{"id":"cred-file"}`), nil
	})

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, Consumer: "alice", GatewayGroup: "gg1", File: tmpFile}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	var item api.Credential
	if err := json.Unmarshal(out.Bytes(), &item); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if item.ID != "cred-file" {
		t.Fatalf("expected credential id from file, got %+v", item)
	}
	registry.Verify(t)
}

func TestCreateCredential_PositionalOverridesFileID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	tmpFile := t.TempDir() + "/credential.json"
	if err := os.WriteFile(tmpFile, []byte(`{"id":"from-file","plugins":{"key-auth":{"key":"k"}}}`), 0o644); err != nil {
		t.Fatalf("write credential file: %v", err)
	}

	registry := &httpmock.Registry{}
	registry.RegisterResponder(http.MethodPut, "/apisix/admin/consumers/alice/credentials/from-positional", func(req *http.Request) (httpmock.Response, error) {
		return httpmock.JSONResponse(`{"id":"from-positional"}`), nil
	})

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, Consumer: "alice", GatewayGroup: "gg1", ID: "from-positional", File: tmpFile}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	registry.Verify(t)
}

func TestCreateCredential_FileRejectsInvalidID(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	tmpFile := t.TempDir() + "/credential.json"
	if err := os.WriteFile(tmpFile, []byte(`{"id":123,"plugins":{"key-auth":{"key":"k"}}}`), 0o644); err != nil {
		t.Fatalf("write credential file: %v", err)
	}

	opts := &Options{IO: ios, Client: func() (*http.Client, error) {
		t.Fatal("unexpected HTTP client call")
		return nil, nil
	}, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, Consumer: "alice", GatewayGroup: "gg1", File: tmpFile}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "credential id must be a non-empty string") {
		t.Fatalf("expected invalid credential id error, got: %v", err)
	}
}

func TestCreateCredential_MissingGatewayGroup(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local"}, nil
	}, Consumer: "alice"}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "gateway group is required") {
		t.Fatalf("expected missing gateway group error, got: %v", err)
	}
}

func TestCreateCredential_MissingConsumer(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, GatewayGroup: "gg1"}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "--consumer is required") {
		t.Fatalf("expected missing consumer error, got: %v", err)
	}
}

func TestCreateCredential_APIError(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodPost, "/apisix/admin/consumers/alice/credentials", httpmock.StringResponse(http.StatusInternalServerError, `{"message":"boom"}`))

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, Consumer: "alice", GatewayGroup: "gg1"}

	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "API error") {
		t.Fatalf("expected api error, got: %v", err)
	}

	registry.Verify(t)
}
