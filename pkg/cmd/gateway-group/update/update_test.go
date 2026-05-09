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

type mockConfig struct{}

func (m *mockConfig) BaseURL() string                                 { return "" }
func (m *mockConfig) Token() string                                   { return "" }
func (m *mockConfig) GatewayGroup() string                            { return "" }
func (m *mockConfig) TLSSkipVerify() bool                             { return false }
func (m *mockConfig) CACert() string                                  { return "" }
func (m *mockConfig) CurrentContext() string                          { return "" }
func (m *mockConfig) Contexts() []config.Context                      { return nil }
func (m *mockConfig) GetContext(name string) (*config.Context, error) { return nil, nil }
func (m *mockConfig) AddContext(ctx config.Context) error             { return nil }
func (m *mockConfig) RemoveContext(name string) error                 { return nil }
func (m *mockConfig) SetCurrentContext(name string) error             { return nil }
func (m *mockConfig) Save() error                                     { return nil }

func TestUpdateGatewayGroup_PreservesRequiredFields(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	registry := &httpmock.Registry{}
	registry.Register(http.MethodGet, "/api/gateway_groups/gg1", httpmock.JSONResponse(`{
		"id":"gg1",
		"name":"default",
		"description":"old description",
		"prefix":"/old",
		"labels":{"env":"old"},
		"status":1
	}`))
	registry.RegisterResponder(http.MethodPut, "/api/gateway_groups/gg1", func(r *http.Request) (httpmock.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return httpmock.Response{}, err
		}
		var payload api.GatewayGroup
		if err := json.Unmarshal(body, &payload); err != nil {
			return httpmock.Response{}, err
		}
		if payload.Name != "default" {
			t.Fatalf("expected existing name to be preserved, got %q", payload.Name)
		}
		if payload.Description != "new description" {
			t.Fatalf("expected description to be updated, got %q", payload.Description)
		}
		if payload.Prefix != "/old" {
			t.Fatalf("expected existing prefix to be preserved, got %q", payload.Prefix)
		}
		if payload.Labels["env"] != "new" {
			t.Fatalf("expected labels to be replaced from flags, got %+v", payload.Labels)
		}
		return httpmock.JSONResponse(`{"id":"gg1","name":"default","description":"new description","prefix":"/old","labels":{"env":"new"},"status":1}`), nil
	})

	opts := &Options{
		IO:             ios,
		Client:         func() (*http.Client, error) { return registry.GetClient(), nil },
		Config:         func() (config.Config, error) { return &mockConfig{}, nil },
		Output:         "json",
		ID:             "gg1",
		Description:    "new description",
		DescriptionSet: true,
		Labels:         []string{"env=new"},
		LabelsSet:      true,
	}
	if err := updateRun(opts); err != nil {
		t.Fatalf("updateRun failed: %v", err)
	}

	var item api.GatewayGroup
	if err := json.Unmarshal(out.Bytes(), &item); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if item.Name != "default" || item.Description != "new description" {
		t.Fatalf("unexpected gateway group output: %+v", item)
	}
	registry.Verify(t)
}
