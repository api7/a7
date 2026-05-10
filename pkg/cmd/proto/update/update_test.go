package update

import (
	"encoding/json"
	"io"
	"net/http"
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

func TestUpdateProto_JSON(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/apisix/admin/protos/p1", httpmock.JSONResponse(`{
		"id":"p1",
		"desc":"d1",
		"content":"syntax = \"proto3\"; message X {}",
		"labels":{"env":"old"}
	}`))
	registry.RegisterResponder(http.MethodPut, "/apisix/admin/protos/p1", func(r *http.Request) (httpmock.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return httpmock.Response{}, err
		}
		var payload api.Proto
		if err := json.Unmarshal(body, &payload); err != nil {
			return httpmock.Response{}, err
		}
		if payload.Content != `syntax = "proto3"; message X {}` {
			t.Fatalf("expected existing proto content to be preserved, got %q", payload.Content)
		}
		if payload.Desc != "d2" {
			t.Fatalf("expected desc to be updated, got %q", payload.Desc)
		}
		if payload.Labels["env"] != "dev" {
			t.Fatalf("expected labels to be replaced from flags, got %+v", payload.Labels)
		}
		return httpmock.JSONResponse(`{"id":"p1","desc":"d2","content":"syntax = \"proto3\"; message X {}","labels":{"env":"dev"}}`), nil
	})

	opts := &Options{
		IO:     ios,
		Client: func() (*http.Client, error) { return registry.GetClient(), nil },
		Config: func() (config.Config, error) {
			return &mockConfig{baseURL: "http://api.local", gatewayGroup: "gg1"}, nil
		},
		GatewayGroup: "gg1",
		ID:           "p1",
		Desc:         "d2",
		DescSet:      true,
		Labels:       []string{"env=dev"},
		LabelsSet:    true,
	}
	if err := actionRun(opts); err != nil {
		t.Fatalf("actionRun failed: %v", err)
	}
	var item api.Proto
	if err := json.Unmarshal(out.Bytes(), &item); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if item.ID != "p1" || item.Desc != "d2" {
		t.Fatalf("unexpected item: %+v", item)
	}
	registry.Verify(t)
}
