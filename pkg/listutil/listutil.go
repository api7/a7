// Package listutil provides shared helpers for paginating and aggregating
// API7 EE list endpoints. The route endpoint requires per-service queries
// under access-token auth, so this package centralizes that pattern for
// reuse across `config sync/dump` and `route list`.
package listutil

import (
	"encoding/json"
	"fmt"

	"github.com/api7/a7/pkg/api"
	"github.com/api7/a7/pkg/cmdutil"
)

const defaultPageSize = 500

// FetchPaginated fetches all items from a paginated API7 EE list endpoint.
// API7 EE returns ListResponse[T] with .List []T directly (no ListItem wrapper).
// Returns (nil, nil) when the endpoint signals that the resource is unavailable
// (e.g., stream mode disabled, endpoint not exposed); callers that need stricter
// behavior should inspect the returned error themselves.
func FetchPaginated[T any](client *api.Client, path string, extraQuery map[string]string) ([]T, error) {
	page := 1
	var items []T

	for {
		query := map[string]string{
			"page":      fmt.Sprintf("%d", page),
			"page_size": fmt.Sprintf("%d", defaultPageSize),
		}
		for k, v := range extraQuery {
			query[k] = v
		}

		body, err := client.Get(path, query)
		if err != nil {
			if cmdutil.IsOptionalResourceError(err) {
				return nil, nil
			}
			return nil, err
		}

		var resp api.ListResponse[T]
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		items = append(items, resp.List...)
		if len(resp.List) == 0 || len(items) >= resp.Total {
			break
		}
		page++
	}

	return items, nil
}

// FetchRoutesForServices fetches routes for each service and aggregates the
// results, deduplicating by route ID. API7 EE requires `service_id` on
// /apisix/admin/routes under access-token auth, so callers that need every
// route in a gateway group must iterate services and merge.
//
// baseQuery is merged into each per-service request (e.g. `gateway_group_id`).
// A service whose route fetch returns an "optional resource" error (400/404)
// is skipped, matching the behavior of FetchPaginated.
func FetchRoutesForServices(client *api.Client, services []api.Service, baseQuery map[string]string) ([]api.Route, error) {
	seen := make(map[string]bool)
	var allRoutes []api.Route
	for _, svc := range services {
		if svc.ID == "" {
			continue
		}
		q := make(map[string]string, len(baseQuery)+1)
		for k, v := range baseQuery {
			q[k] = v
		}
		q["service_id"] = svc.ID
		routes, err := FetchPaginated[api.Route](client, "/apisix/admin/routes", q)
		if err != nil {
			return nil, err
		}
		for _, r := range routes {
			key := r.ID
			if key == "" {
				allRoutes = append(allRoutes, r)
				continue
			}
			if !seen[key] {
				seen[key] = true
				allRoutes = append(allRoutes, r)
			}
		}
	}
	return allRoutes, nil
}
