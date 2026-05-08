package update

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/api"
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestMaybeReadFileReadsBareRelativePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	if err := os.WriteFile("key.pem", []byte("file-key"), 0o644); err != nil {
		t.Fatalf("write key: %v", err)
	}

	got, err := maybeReadFile("key.pem")
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

func TestUpdateSSL_PreservesCertificateWhenUpdatingSNI(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/apisix/admin/ssls/ssl1":
			return jsonHTTPResponse(http.StatusOK, `{"value":{"id":"ssl1","cert":"old-cert","key":"old-key","snis":["old.example.com"],"type":"server","status":1}}`), nil
		case req.Method == http.MethodPut && req.URL.Path == "/apisix/admin/ssls/ssl1":
			var body api.SSL
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if body.Cert != "old-cert" || body.Key != "old-key" {
				t.Fatalf("expected cert/key to be preserved, got %#v", body)
			}
			if len(body.SNIs) != 1 || body.SNIs[0] != "new.example.com" {
				t.Fatalf("expected updated sni, got %#v", body.SNIs)
			}
			return jsonHTTPResponse(http.StatusOK, `{"value":{"id":"ssl1","cert":"old-cert","key":"old-key","snis":["new.example.com"],"type":"server","status":1}}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})}

	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return httpClient, nil },
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
}

func TestUpdateSSL_SendsExplicitStatusZero(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/apisix/admin/ssls/ssl1":
			return jsonHTTPResponse(http.StatusOK, `{"value":{"id":"ssl1","cert":"old-cert","key":"old-key","snis":["old.example.com"],"type":"server","status":1}}`), nil
		case req.Method == http.MethodPut && req.URL.Path == "/apisix/admin/ssls/ssl1":
			var payload map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if payload["status"] != float64(0) {
				t.Fatalf("expected explicit status 0, got payload %#v", payload)
			}
			return jsonHTTPResponse(http.StatusOK, `{"value":{"id":"ssl1","cert":"old-cert","key":"old-key","snis":["old.example.com"],"type":"server","status":0}}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})}

	err := actionRun(&Options{
		IO:           ios,
		Client:       func() (*http.Client, error) { return httpClient, nil },
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
}
