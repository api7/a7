package delete

import (
	"net/http"
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

func TestDeleteRoute_Success(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodDelete, "/apisix/admin/routes/r1", httpmock.StringResponse(http.StatusNoContent, ""))
	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, GatewayGroup: "gg1", ID: "r1"}
	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(out.String(), `Route "r1" deleted.`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
	registry.Verify(t)
}

func TestDeleteRoute_MissingGatewayGroup(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return (&httpmock.Registry{}).GetClient(), nil }, Config: func() (config.Config, error) { return &mockConfig{baseURL: "http://api.local"}, nil }, ID: "r1"}
	err := actionRun(opts)
	if err == nil || !strings.Contains(err.Error(), "gateway group is required") {
		t.Fatalf("expected gateway group error, got: %v", err)
	}
}

func TestDeleteRoute_WithForce(t *testing.T) {
	ios, _, out, errOut := iostreams.Test()
	ios.SetStdinTTY(true)

	registry := &httpmock.Registry{}
	registry.Register(http.MethodDelete, "/apisix/admin/routes/r1", httpmock.StringResponse(http.StatusNoContent, ""))

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, GatewayGroup: "gg1", ID: "r1", Force: true}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if strings.Contains(errOut.String(), "Delete route") {
		t.Fatalf("unexpected prompt output: %s", errOut.String())
	}
	if !strings.Contains(out.String(), `Route "r1" deleted.`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
	registry.Verify(t)
}

func TestDeleteRoute_ConfirmYes(t *testing.T) {
	ios, in, out, errOut := iostreams.Test()
	ios.SetStdinTTY(true)
	in.WriteString("y\n")

	registry := &httpmock.Registry{}
	registry.Register(http.MethodDelete, "/apisix/admin/routes/r1", httpmock.StringResponse(http.StatusNoContent, ""))

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, GatewayGroup: "gg1", ID: "r1"}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if !strings.Contains(errOut.String(), `Delete route "r1"? (y/N):`) {
		t.Fatalf("expected prompt output, got: %s", errOut.String())
	}
	if !strings.Contains(out.String(), `Route "r1" deleted.`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
	registry.Verify(t)
}

func TestDeleteRoute_ConfirmNo(t *testing.T) {
	ios, in, out, errOut := iostreams.Test()
	ios.SetStdinTTY(true)
	in.WriteString("n\n")

	registry := &httpmock.Registry{}
	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, GatewayGroup: "gg1", ID: "r1"}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if strings.Contains(out.String(), `Route "r1" deleted.`) {
		t.Fatalf("unexpected success output: %s", out.String())
	}
	if !strings.Contains(errOut.String(), "Aborted.") {
		t.Fatalf("expected aborted output, got: %s", errOut.String())
	}
}

func TestDeleteRoute_NonTTY_NoPrompt(t *testing.T) {
	ios, _, out, errOut := iostreams.Test()

	registry := &httpmock.Registry{}
	registry.Register(http.MethodDelete, "/apisix/admin/routes/r1", httpmock.StringResponse(http.StatusNoContent, ""))

	opts := &Options{IO: ios, Client: func() (*http.Client, error) { return registry.GetClient(), nil }, Config: func() (config.Config, error) {
		return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
	}, GatewayGroup: "gg1", ID: "r1"}

	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	if strings.Contains(errOut.String(), "Delete route") {
		t.Fatalf("unexpected prompt output: %s", errOut.String())
	}
	if !strings.Contains(out.String(), `Route "r1" deleted.`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
	registry.Verify(t)
}
