package list

import (
	"net/http"
	"strings"
	"testing"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/httpmock"
	"github.com/api7/a7/pkg/iostreams"
)

// mockConfig implements the config.Config interface with minimal methods.
type mockConfig struct{}

func (m *mockConfig) BaseURL() string {
	return ""
}

func (m *mockConfig) Token() string {
	return ""
}

func (m *mockConfig) GatewayGroup() string {
	return ""
}

func (m *mockConfig) TLSSkipVerify() bool {
	return false
}

func (m *mockConfig) CACert() string {
	return ""
}

func (m *mockConfig) CurrentContext() string {
	return ""
}

func (m *mockConfig) Contexts() []config.Context {
	return nil
}

func (m *mockConfig) GetContext(name string) (*config.Context, error) {
	return nil, nil
}

func (m *mockConfig) AddContext(ctx config.Context) error {
	return nil
}

func (m *mockConfig) RemoveContext(name string) error {
	return nil
}

func (m *mockConfig) SetCurrentContext(name string) error {
	return nil
}

func (m *mockConfig) Save() error {
	return nil
}

func (m *mockConfig) Path() string {
	return ""
}

func TestListGatewayGroups_Table(t *testing.T) {
	// Setup httpmock registry
	reg := &httpmock.Registry{}
	jsonResp := `{"total":2,"list":[{"id":"gw1","name":"default","description":"Default group","status":1},{"id":"gw2","name":"staging","description":"Staging group","status":1}]}`
	reg.Register("GET", "/api/gateway_groups", httpmock.JSONResponse(jsonResp))

	// Setup IOStreams
	ios, _, out, _ := iostreams.Test()

	// Create Options
	opts := &Options{
		IO: ios,
		Client: func() (*http.Client, error) {
			return reg.GetClient(), nil
		},
		Config: func() (config.Config, error) {
			return &mockConfig{}, nil
		},
		Output: "",
	}

	// Run the command
	err := listRun(opts)
	if err != nil {
		t.Fatalf("listRun failed: %v", err)
	}

	// Verify output contains headers and rows
	output := out.String()
	if !strings.Contains(output, "ID") {
		t.Error("table should contain ID header")
	}
	if !strings.Contains(output, "NAME") {
		t.Error("table should contain NAME header")
	}
	if !strings.Contains(output, "DESCRIPTION") {
		t.Error("table should contain DESCRIPTION header")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("table should contain STATUS header")
	}
	if !strings.Contains(output, "gw1") {
		t.Error("table should contain first gateway group ID")
	}
	if !strings.Contains(output, "default") {
		t.Error("table should contain first gateway group name")
	}
	if !strings.Contains(output, "gw2") {
		t.Error("table should contain second gateway group ID")
	}
	if !strings.Contains(output, "staging") {
		t.Error("table should contain second gateway group name")
	}

	// Verify mock was called
	reg.Verify(t)
}

func TestListGatewayGroups_JSON(t *testing.T) {
	// Setup httpmock registry
	reg := &httpmock.Registry{}
	jsonResp := `{"total":2,"list":[{"id":"gw1","name":"default","description":"Default group","status":1},{"id":"gw2","name":"staging","description":"Staging group","status":1}]}`
	reg.Register("GET", "/api/gateway_groups", httpmock.JSONResponse(jsonResp))

	// Setup IOStreams
	ios, _, out, _ := iostreams.Test()

	// Create Options with JSON output
	opts := &Options{
		IO: ios,
		Client: func() (*http.Client, error) {
			return reg.GetClient(), nil
		},
		Config: func() (config.Config, error) {
			return &mockConfig{}, nil
		},
		Output: "json",
	}

	// Run the command
	err := listRun(opts)
	if err != nil {
		t.Fatalf("listRun failed: %v", err)
	}

	// Verify JSON output
	output := out.String()
	if !strings.Contains(output, "gw1") {
		t.Error("JSON output should contain first gateway group ID")
	}
	if !strings.Contains(output, "default") {
		t.Error("JSON output should contain first gateway group name")
	}
	if !strings.Contains(output, "gw2") {
		t.Error("JSON output should contain second gateway group ID")
	}

	// Verify mock was called
	reg.Verify(t)
}

func TestListGatewayGroups_Empty(t *testing.T) {
	// Setup httpmock registry with empty list
	reg := &httpmock.Registry{}
	jsonResp := `{"total":0,"list":[]}`
	reg.Register("GET", "/api/gateway_groups", httpmock.JSONResponse(jsonResp))

	// Setup IOStreams
	ios, _, out, _ := iostreams.Test()

	// Create Options
	opts := &Options{
		IO: ios,
		Client: func() (*http.Client, error) {
			return reg.GetClient(), nil
		},
		Config: func() (config.Config, error) {
			return &mockConfig{}, nil
		},
		Output: "",
	}

	// Run the command
	err := listRun(opts)
	if err != nil {
		t.Fatalf("listRun failed: %v", err)
	}

	// Verify output contains only headers (no data rows)
	output := out.String()
	if !strings.Contains(output, "ID") {
		t.Error("table should contain ID header")
	}
	if !strings.Contains(output, "NAME") {
		t.Error("table should contain NAME header")
	}
	if !strings.Contains(output, "DESCRIPTION") {
		t.Error("table should contain DESCRIPTION header")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("table should contain STATUS header")
	}

	// Verify mock was called
	reg.Verify(t)
}

func TestListGatewayGroups_APIError(t *testing.T) {
	// Setup httpmock registry with error response
	reg := &httpmock.Registry{}
	jsonResp := `{"message":"Internal server error"}`
	reg.Register("GET", "/api/gateway_groups", httpmock.StringResponse(500, jsonResp))

	// Setup IOStreams
	ios, _, _, _ := iostreams.Test()

	// Create Options
	opts := &Options{
		IO: ios,
		Client: func() (*http.Client, error) {
			return reg.GetClient(), nil
		},
		Config: func() (config.Config, error) {
			return &mockConfig{}, nil
		},
		Output: "",
	}

	// Run the command and expect error
	err := listRun(opts)
	if err == nil {
		t.Fatal("should return error for API failure")
	}
	if !strings.Contains(err.Error(), "failed to list gateway groups") {
		t.Errorf("error message should mention list failure, got: %v", err)
	}

	// Verify mock was called
	reg.Verify(t)
}
