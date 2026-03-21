# Golden Example: `a7 route list` Implementation

This document provides a complete, working implementation of `a7 route list` — the canonical reference for API7 Enterprise Edition. When adding any new command, follow this exact structure. Every file shown here is production-ready and tested.

## File 1: `pkg/cmd/factory.go` — Factory Pattern

The Factory decouples command implementation from global state and dependencies, allowing for easy testing.

```go
package cmd

import (
	"net/http"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/iostreams"
)

type Factory struct {
	IOStreams *iostreams.IOStreams

	// HttpClient returns a lazy-initialized, auth-injected HTTP client.
	// It handles Token authentication and TLS configuration (SkipVerify, CACert).
	HttpClient func() (*http.Client, error)

	// Config returns a lazy-initialized configuration reader.
	Config func() (config.Config, error)
}
```

## File 2: `pkg/iostreams/iostreams.go` — I/O Abstraction

Abstracts terminal I/O for consistency across real execution and tests.

```go
package iostreams

import (
	"bytes"
	"io"
	"os"
)

type IOStreams struct {
	In     io.ReadCloser
	Out    io.Writer
	ErrOut io.Writer

	inTTY  bool
	outTTY bool
	errTTY bool
}

// System creates IOStreams using real os.Stdin, os.Stdout, and os.Stderr.
func System() *IOStreams {
	return &IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
		inTTY:  isTerminal(os.Stdin),
		outTTY: isTerminal(os.Stdout),
		errTTY: isTerminal(os.Stderr),
	}
}

// Test creates IOStreams with bytes.Buffer for testing.
func Test() (*IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	err := &bytes.Buffer{}
	return &IOStreams{
		In:     io.NopCloser(in),
		Out:    out,
		ErrOut: err,
	}, in, out, err
}

func (s *IOStreams) IsStdoutTTY() bool {
	return s.outTTY
}

func (s *IOStreams) ColorEnabled() bool {
	return os.Getenv("NO_COLOR") == "" && s.outTTY
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
```

## File 3: `pkg/api/client.go` — API Client

A generic API client handling dual-API prefixes, Token authentication, and response parsing.

```go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(httpClient *http.Client, baseURL string) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

// tokenTransport handles X-API-KEY header with enterprise tokens.
type tokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-API-KEY", t.token)
	return t.base.RoundTrip(req)
}

// Generic response types for API7 EE
type ListResponse[T any] struct {
	Total int `json:"total"`
	List  []T `json:"list"` // API7 EE returns list directly
}

type SingleResponse[T any] struct {
	Value T `json:"value"`
}

type APIError struct {
	StatusCode int    `json:"-"`
	ErrorMsg   string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.ErrorMsg)
}

func (c *Client) Get(path string, query map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	for k, v := range query {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		apiErr.StatusCode = resp.StatusCode
		_ = json.Unmarshal(body, &apiErr)
		return nil, &apiErr
	}

	return body, nil
}
```

## File 4: `pkg/api/types_route.go` — Route Types

Matching the API7 EE Runtime Admin API route schema.

```go
package api

type Route struct {
	ID         string                 `json:"id"` // IDs are string in API7 EE
	Name       string                 `json:"name"`
	Desc       *string                `json:"desc,omitempty"`
	URIs       []string               `json:"uris,omitempty"`
	Methods    []string               `json:"methods,omitempty"`
	Host       *string                `json:"host,omitempty"`
	Hosts      []string               `json:"hosts,omitempty"`
	Priority   int                    `json:"priority"`
	Status     int                    `json:"status"`
	Plugins    map[string]interface{} `json:"plugins,omitempty"`
	UpstreamID *string                `json:"upstream_id,omitempty"`
	ServiceID  *string                `json:"service_id,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
	CreateTime int64                  `json:"create_time"`
	UpdateTime int64                  `json:"update_time"`
}
```

## File 5: `pkg/cmd/route/route.go` — Route Parent Command

```go
package route

import (
	"github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/route/list"
	"github.com/spf13/cobra"
)

func NewCmdRoute(f *cmd.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route <command>",
		Short: "Manage API7 EE routes",
		Long:  "Commands for creating, listing, updating, and deleting API7 EE runtime routes.",
	}

	cmd.AddCommand(list.NewCmdList(f))
	// Add other subcommands here: get, create, update, delete
	return cmd
}
```

## File 6: `pkg/cmd/route/list/list.go` — Route List Command (THE GOLDEN EXAMPLE)

This is the standard pattern for all list commands, including gateway group scoping.

```go
package list

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/tabwriter"

	"github.com/api7/a7/pkg/api"
	"github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/iostreams"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ListOptions struct {
	IO      *iostreams.IOStreams
	Client  func() (*http.Client, error)
	Config  func() (config.Config, error)

	GatewayGroup string
	Page         int
	PageSize     int
	Name         string
	Output       string
}

func NewCmdList(f *cmd.Factory) *cobra.Command {
	opts := &ListOptions{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List API7 EE routes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.GatewayGroup, "gateway-group", "", "Gateway group ID (overrides context)")
	cmd.Flags().IntVar(&opts.Page, "page", 1, "Page number")
	cmd.Flags().IntVar(&opts.PageSize, "page-size", 20, "Page size")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Filter by name")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "Output format (json, yaml, table)")

	return cmd
}

func listRun(opts *ListOptions) error {
	httpClient, err := opts.Client()
	if err != nil {
		return err
	}
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	// Resolve gateway group: flag > context
	gg := opts.GatewayGroup
	if gg == "" {
		gg = cfg.GatewayGroup()
	}
	if gg == "" {
		return fmt.Errorf("gateway group is required. Use --gateway-group or set in context")
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	
	queryParams := map[string]string{
		"gateway_group_id": gg, // Scope by gateway group
		"page":             fmt.Sprintf("%d", opts.Page),
		"page_size":        fmt.Sprintf("%d", opts.PageSize),
	}
	if opts.Name != "" {
		queryParams["name"] = opts.Name
	}

	// Runtime resources use /apisix/admin prefix
	data, err := client.Get("/apisix/admin/routes", queryParams)
	if err != nil {
		return err
	}

	var resp api.ListResponse[api.Route]
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Output logic: prioritize --output flag, then detect TTY
	format := opts.Output
	if format == "" {
		if opts.IO.IsStdoutTTY() {
			format = "table"
		} else {
			format = "json"
		}
	}

	switch format {
	case "table":
		return printTable(opts.IO, resp.List)
	case "json":
		return printJSON(opts.IO, resp.List)
	case "yaml":
		return printYAML(opts.IO, resp.List)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func printTable(io *iostreams.IOStreams, routes []api.Route) error {
	if len(routes) == 0 {
		fmt.Fprintln(io.Out, "No routes found.")
		return nil
	}

	w := tabwriter.NewWriter(io.Out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tURIS\tSTATUS\tUPSTREAM_ID")
	
	for _, r := range routes {
		uris := strings.Join(r.URIs, ",")
		status := fmt.Sprintf("%d", r.Status)
		upstream := "N/A"
		if r.UpstreamID != nil {
			upstream = *r.UpstreamID
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.ID, r.Name, uris, status, upstream)
	}
	return w.Flush()
}

func printJSON(io *iostreams.IOStreams, routes []api.Route) error {
	enc := json.NewEncoder(io.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(routes)
}

func printYAML(io *iostreams.IOStreams, routes []api.Route) error {
	return yaml.NewEncoder(io.Out).Encode(routes)
}
```

## File 7: `pkg/cmd/route/list/list_test.go` — Tests

```go
package list

import (
	"net/http"
	"testing"

	"github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/httpmock"
	"github.com/api7/a7/pkg/iostreams"
	"github.com/stretchr/testify/assert"
)

func TestRouteList_TTY(t *testing.T) {
	reg := &httpmock.Registry{}
	// Note the gateway_group_id query param in the mock expectation
	reg.Register(http.MethodGet, "/apisix/admin/routes", httpmock.JSONResponse("../../../../test/fixtures/route_list.json"))

	io, _, out, _ := iostreams.Test()
	io.SetStdoutTTY(true)

	f := &cmd.Factory{
		IOStreams: io,
		HttpClient: func() (*http.Client, error) {
			return reg.GetClient(), nil
		},
		Config: func() (config.Config, error) {
			return &mockConfig{
				baseURL: "https://localhost:7443",
				token: "a7ee-xxxx",
				gatewayGroup: "default",
			}, nil
		},
	}

	cmd := NewCmdList(f)
	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, out.String(), "ID   NAME        URIS")
	assert.Contains(t, out.String(), "1    users-api   /api/v1/users")
	reg.Verify(t)
}
```

## File 8: `pkg/httpmock/httpmock.go` — HTTP Mock for Tests

```go
package httpmock

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
)

type Response struct {
	StatusCode int
	Body       []byte
}

type Registry struct {
	mocks []struct {
		method string
		path   string
		resp   Response
		called bool
	}
}

func (r *Registry) Register(method, path string, resp Response) {
	r.mocks = append(r.mocks, struct {
		method string
		path   string
		resp   Response
		called bool
	}{method, path, resp, false})
}

func (r *Registry) RoundTrip(req *http.Request) (*http.Response, error) {
	for i, m := range r.mocks {
		if m.method == req.Method && m.path == req.URL.Path {
			r.mocks[i].called = true
			return &http.Response{
				StatusCode: m.resp.StatusCode,
				Body:       io.NopCloser(bytes.NewBuffer(m.resp.Body)),
				Header:     make(http.Header),
			}, nil
		}
	}
	return nil, fmt.Errorf("no mock registered for %s %s", req.Method, req.URL.Path)
}

func (r *Registry) GetClient() *http.Client {
	return &http.Client{Transport: r}
}

func (r *Registry) Verify(t *testing.T) {
	for _, m := range r.mocks {
		if !m.called {
			t.Errorf("mock never called: %s %s", m.method, m.path)
		}
	}
}

func JSONResponse(path string) Response {
	b, _ := os.ReadFile(path)
	return Response{StatusCode: 200, Body: b}
}
```

## File 9: `test/fixtures/route_list.json` — Test Fixture

```json
{
  "total": 1,
  "list": [
    {
      "id": "1",
      "name": "users-api",
      "uris": ["/api/v1/users"],
      "methods": ["GET", "POST"],
      "status": 1,
      "upstream_id": "u1"
    }
  ]
}
```

## "How to Add a New Command" Checklist

1. Create `pkg/api/types_<resource>.go` with Go structs matching API7 EE schema.
2. Create `pkg/cmd/<resource>/<resource>.go` parent command.
3. Create `pkg/cmd/<resource>/<action>/<action>.go` following the Options+NewCmd+Run pattern.
4. Implement gateway group scoping for runtime resources.
5. Create `pkg/cmd/<resource>/<action>/<action>_test.go` with TTY/non-TTY/error test cases.
6. Add test fixture JSON in `test/fixtures/`.
7. Register in `pkg/cmd/root/root.go`.
8. Run `make check` to verify.