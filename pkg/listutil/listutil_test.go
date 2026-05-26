package listutil

import (
	"errors"
	"net/http"
	"testing"

	"github.com/api7/a7/pkg/api"
	"github.com/api7/a7/pkg/httpmock"
)

func newTestClient(reg *httpmock.Registry) *api.Client {
	return api.NewClient(reg.GetClient(), "http://api.local")
}

// TestFetchPaginated_StrictPropagates404 confirms that with allowOptional=false
// a 404 from the upstream surfaces as an error instead of being silently
// converted to (nil, nil).
func TestFetchPaginated_StrictPropagates404(t *testing.T) {
	reg := &httpmock.Registry{}
	reg.Register(http.MethodGet, "/apisix/admin/services", httpmock.StringResponse(http.StatusNotFound, `{"error_msg":"not found"}`))

	client := newTestClient(reg)
	items, err := FetchPaginated[api.Service](client, "/apisix/admin/services", nil, false)
	if err == nil {
		t.Fatalf("expected error, got items=%v", items)
	}
	if items != nil {
		t.Errorf("expected nil items on error, got %v", items)
	}
	var apiErr *api.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected *api.APIError with status 404, got %T: %v", err, err)
	}
}

// TestFetchPaginated_StrictPropagates400 confirms 400 is also propagated when
// allowOptional=false. A transient 400 on services/consumers must not be
// swallowed because config sync would then plan destructive operations
// against the unintentionally-empty result.
func TestFetchPaginated_StrictPropagates400(t *testing.T) {
	reg := &httpmock.Registry{}
	reg.Register(http.MethodGet, "/apisix/admin/consumers", httpmock.StringResponse(http.StatusBadRequest, `{"error_msg":"bad request"}`))

	client := newTestClient(reg)
	if _, err := FetchPaginated[api.Consumer](client, "/apisix/admin/consumers", nil, false); err == nil {
		t.Fatal("expected error to propagate when allowOptional=false")
	}
}

// TestFetchPaginated_OptionalSwallows404 confirms the opt-in lenient path
// still works: stream_routes / protos / secret_providers callers pass
// allowOptional=true and expect (nil, nil) when the endpoint signals the
// resource is not in use.
func TestFetchPaginated_OptionalSwallows404(t *testing.T) {
	reg := &httpmock.Registry{}
	reg.Register(http.MethodGet, "/apisix/admin/stream_routes", httpmock.StringResponse(http.StatusNotFound, `{"error_msg":"not found"}`))

	client := newTestClient(reg)
	items, err := FetchPaginated[api.StreamRoute](client, "/apisix/admin/stream_routes", nil, true)
	if err != nil {
		t.Fatalf("expected nil error with allowOptional=true, got %v", err)
	}
	if items != nil {
		t.Errorf("expected nil items, got %v", items)
	}
}

// TestFetchPaginated_OptionalSwallows400 mirrors the 404 case: stream mode
// disabled commonly returns 400, and allowOptional=true callers expect it
// suppressed.
func TestFetchPaginated_OptionalSwallows400(t *testing.T) {
	reg := &httpmock.Registry{}
	reg.Register(http.MethodGet, "/apisix/admin/protos", httpmock.StringResponse(http.StatusBadRequest, `{"error_msg":"stream disabled"}`))

	client := newTestClient(reg)
	items, err := FetchPaginated[api.Proto](client, "/apisix/admin/protos", nil, true)
	if err != nil {
		t.Fatalf("expected nil error with allowOptional=true, got %v", err)
	}
	if items != nil {
		t.Errorf("expected nil items, got %v", items)
	}
}

// TestFetchPaginated_OptionalStillPropagatesOther5xx confirms allowOptional
// only relaxes the 400/404 contract; a 500 must still surface.
func TestFetchPaginated_OptionalStillPropagatesOther5xx(t *testing.T) {
	reg := &httpmock.Registry{}
	reg.Register(http.MethodGet, "/apisix/admin/protos", httpmock.StringResponse(http.StatusInternalServerError, `{"error_msg":"boom"}`))

	client := newTestClient(reg)
	if _, err := FetchPaginated[api.Proto](client, "/apisix/admin/protos", nil, true); err == nil {
		t.Fatal("expected 500 to propagate even with allowOptional=true")
	}
}

// TestFetchRoutesForServices_StrictPropagatesPerServiceError confirms that a
// 404 on one service's /routes is not silently dropped when allowOptional is
// false. This is the race scenario from issue #50: if a service is deleted
// between enumeration and the per-service /routes fetch, the user must see
// an error rather than a quietly-shortened list.
func TestFetchRoutesForServices_StrictPropagatesPerServiceError(t *testing.T) {
	reg := &httpmock.Registry{}
	reg.RegisterResponder(http.MethodGet, "/apisix/admin/routes", func(r *http.Request) (httpmock.Response, error) {
		switch r.URL.Query().Get("service_id") {
		case "svc-a":
			return httpmock.JSONResponse(`{"total":1,"list":[{"id":"r-a","service_id":"svc-a"}]}`), nil
		case "svc-gone":
			return httpmock.StringResponse(http.StatusNotFound, `{"error_msg":"not found"}`), nil
		default:
			return httpmock.JSONResponse(`{"total":0,"list":[]}`), nil
		}
	})

	client := newTestClient(reg)
	services := []api.Service{{ID: "svc-a"}, {ID: "svc-gone"}}
	routes, err := FetchRoutesForServices(client, services, nil, false)
	if err == nil {
		t.Fatalf("expected error, got routes=%v", routes)
	}
	var apiErr *api.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected *api.APIError with status 404, got %T: %v", err, err)
	}
}

// TestFetchRoutesForServices_OptionalSkipsPerServiceError documents that the
// lenient path is still wired through for callers that explicitly opt in.
func TestFetchRoutesForServices_OptionalSkipsPerServiceError(t *testing.T) {
	reg := &httpmock.Registry{}
	reg.RegisterResponder(http.MethodGet, "/apisix/admin/routes", func(r *http.Request) (httpmock.Response, error) {
		switch r.URL.Query().Get("service_id") {
		case "svc-a":
			return httpmock.JSONResponse(`{"total":1,"list":[{"id":"r-a","service_id":"svc-a"}]}`), nil
		case "svc-gone":
			return httpmock.StringResponse(http.StatusNotFound, `{"error_msg":"not found"}`), nil
		default:
			return httpmock.JSONResponse(`{"total":0,"list":[]}`), nil
		}
	})

	client := newTestClient(reg)
	services := []api.Service{{ID: "svc-a"}, {ID: "svc-gone"}}
	routes, err := FetchRoutesForServices(client, services, nil, true)
	if err != nil {
		t.Fatalf("expected no error with allowOptional=true, got %v", err)
	}
	if len(routes) != 1 || routes[0].ID != "r-a" {
		t.Errorf("expected only svc-a's routes, got %+v", routes)
	}
}
