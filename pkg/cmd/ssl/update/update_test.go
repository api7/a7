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

func TestMaybeReadFileReadsBareRelativePath(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "key.pem")
	if err := os.WriteFile(path, []byte("file-key"), 0o644); err != nil {
		t.Fatalf("write key: %v", err)
	}

	got, err := maybeReadFile(path)
	if err != nil {
		t.Fatalf("maybeReadFile failed: %v", err)
	}
	if got != "file-key" {
		t.Fatalf("expected file contents, got %q", got)
	}
}

func TestMaybeReadFileKeepsEmptyAndPEMLiteral(t *testing.T) {
	got, err := maybeReadFile("")
	if err != nil {
		t.Fatalf("maybeReadFile failed: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty input to stay empty")
	}

	input := "-----BEGIN PRIVATE KEY-----\ninline\n-----END PRIVATE KEY-----"
	got, err = maybeReadFile(input)
	if err != nil {
		t.Fatalf("maybeReadFile failed: %v", err)
	}
	if got != input {
		t.Fatalf("expected inline PEM to stay unchanged")
	}
}

func TestMaybeReadFileTreatsMissingBareFilenameAsPath(t *testing.T) {
	_, err := maybeReadFile("missing-cert.pem")
	if err == nil || !strings.Contains(err.Error(), `failed to read file "missing-cert.pem"`) {
		t.Fatalf("expected missing bare filename to be treated as a path, got %v", err)
	}
}

func TestUpdateSSL_PreservesCertificateWhenUpdatingSNI(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/ssls/ssl1", httpmock.JSONResponse(`{"value":{"id":"ssl1","cert":"old-cert","key":"old-key","snis":["old.example.com"],"type":"server","status":1}}`))
	registry.RegisterResponder(http.MethodPut, "/apisix/admin/ssls/ssl1", func(req *http.Request) (httpmock.Response, error) {
		var body api.SSL
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			return httpmock.Response{}, fmt.Errorf("decode request: %w", err)
		}
		if body.Cert != "old-cert" || body.Key != "old-key" {
			return httpmock.Response{}, fmt.Errorf("expected cert/key to be preserved, got %#v", body)
		}
		if len(body.SNIs) != 1 || body.SNIs[0] != "new.example.com" {
			return httpmock.Response{}, fmt.Errorf("expected updated sni, got %#v", body.SNIs)
		}
		return httpmock.JSONResponse(`{"value":{"id":"ssl1","cert":"old-cert","key":"old-key","snis":["new.example.com"],"type":"server","status":1}}`), nil
	})

	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return registry.GetClient(), nil },
		GatewayGroup: "gg1",
		ID:           "ssl1",
		SNIs:         []string{"new.example.com"},
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
	})
	if err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), "new.example.com") {
		t.Fatalf("expected updated ssl output, got %s", out.String())
	}
	var output api.SSL
	if err := json.Unmarshal(out.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if strings.Contains(out.String(), "old-key") || output.Key != api.RedactedSSLKey {
		t.Fatalf("expected ssl key to be redacted in output, got %+v", output)
	}
	registry.Verify(t)
}

func TestUpdateSSL_SendsExplicitStatusZero(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/ssls/ssl1", httpmock.JSONResponse(`{"value":{"id":"ssl1","cert":"old-cert","key":"old-key","snis":["old.example.com"],"type":"server","status":1}}`))
	registry.RegisterResponder(http.MethodPut, "/apisix/admin/ssls/ssl1", func(req *http.Request) (httpmock.Response, error) {
		var payload map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			return httpmock.Response{}, fmt.Errorf("decode request: %w", err)
		}
		if payload["status"] != float64(0) {
			return httpmock.Response{}, fmt.Errorf("expected explicit status 0, got payload %#v", payload)
		}
		return httpmock.JSONResponse(`{"value":{"id":"ssl1","cert":"old-cert","key":"old-key","snis":["old.example.com"],"type":"server","status":0}}`), nil
	})

	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return registry.GetClient(), nil },
		GatewayGroup: "gg1",
		ID:           "ssl1",
		Status:       0,
		StatusSet:    true,
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", token: "test", gatewayGroup: "gg1"}, nil
		},
	})
	if err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	registry.Verify(t)
}
