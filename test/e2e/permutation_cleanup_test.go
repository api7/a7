//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// permCleanupPrefix is the resource-id prefix every CRUD round-trip case must
// use. The post-suite sweep deletes anything matching this regex to backstop
// missed defer cleanups.
const permCleanupPrefix = "a7-perm"

var permIDRegex = regexp.MustCompile(`^a7-perm`)

// cleanupTarget describes how to list+delete leftover resources of one type.
// Lister returns a slice of resource IDs (filtered by permIDRegex by the
// caller). Deleter removes one resource by id.
type cleanupTarget struct {
	name    string
	lister  func() ([]string, error)
	deleter func(id string) error
}

// permCleanupTargets enumerates the resource types the suite mutates. New
// resource types should be added here so the sweep covers them.
func permCleanupTargets() []cleanupTarget {
	return []cleanupTarget{
		{name: "routes", lister: listRouteIDs, deleter: deleteRouteByID},
		{name: "services", lister: listServiceIDs, deleter: deleteServiceByID},
		{name: "consumers", lister: listConsumerIDs, deleter: deleteConsumerByID},
		{name: "ssls", lister: listSSLIDs, deleter: deleteSSLByID},
		{name: "secrets", lister: listSecretIDs, deleter: deleteSecretByID},
		{name: "global_rules", lister: listGlobalRuleIDs, deleter: deleteGlobalRuleByID},
		{name: "stream_routes", lister: listStreamRouteIDs, deleter: deleteStreamRouteByID},
		{name: "protos", lister: listProtoIDs, deleter: deleteProtoByID},
		{name: "plugin_metadata", lister: listPluginMetadataIDs, deleter: deletePluginMetadataByID},
	}
}

// errListUnsupported is returned by listers when the resource type cannot be
// listed without parent context (e.g. routes require service_id under access-
// token auth) or the endpoint is not exposed on this EE build. The sweep
// silently skips these — per-case defers handle cleanup for the resources we
// actually created.
var errListUnsupported = fmt.Errorf("list unsupported on this EE; skipping sweep")

// permSweep runs the full backstop cleanup. Errors are accumulated and
// returned as a single error so the caller can decide whether to fail the
// suite or just log. errListUnsupported is silently dropped.
func permSweep() []error {
	var errs []error
	for _, target := range permCleanupTargets() {
		ids, err := target.lister()
		if err != nil {
			if err == errListUnsupported {
				continue
			}
			errs = append(errs, fmt.Errorf("list %s: %w", target.name, err))
			continue
		}
		for _, id := range ids {
			if !permIDRegex.MatchString(id) {
				continue
			}
			if err := target.deleter(id); err != nil {
				errs = append(errs, fmt.Errorf("delete %s/%s: %w", target.name, id, err))
			}
		}
	}
	return errs
}

// listIDsFromRuntime hits the runtime admin api and pulls .list[].value.id
// (the APISIX wrapped shape) or .list[].id (the flat shape, used by control-
// plane endpoints). Returns all observed ids, unfiltered. Returns
// errListUnsupported for HTML responses (frontend served the path), 400
// (parameter required), or 404 (endpoint not exposed) so the sweep can skip.
func listIDsFromRuntime(path string) ([]string, error) {
	resp, err := runtimeAdminAPI(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if listResponseIsUnsupported(resp.StatusCode, body) {
		return nil, errListUnsupported
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET %s -> %d: %s", path, resp.StatusCode, truncate(string(body), 200))
	}
	return parseListIDs(body)
}

func listIDsFromAdmin(path string) ([]string, error) {
	resp, err := adminAPI(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if listResponseIsUnsupported(resp.StatusCode, body) {
		return nil, errListUnsupported
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET %s -> %d: %s", path, resp.StatusCode, truncate(string(body), 200))
	}
	return parseListIDs(body)
}

// listResponseIsUnsupported returns true when the response looks like
// (a) an HTML 404 served by the dashboard frontend (the API path is not
// routed), (b) a 400 because the endpoint needs a required parent id, or
// (c) any 404 from the admin API itself.
func listResponseIsUnsupported(status int, body []byte) bool {
	if status == http.StatusNotFound {
		return true
	}
	if status == http.StatusBadRequest && strings.Contains(string(body), "is required but missing") {
		return true
	}
	if strings.HasPrefix(strings.TrimSpace(string(body)), "<") {
		return true
	}
	return false
}

// parseListIDs handles both shapes: {list:[{id,...}]} (control-plane) and
// {list:[{value:{id,...},...}]} (runtime APISIX wrapped).
func parseListIDs(body []byte) ([]string, error) {
	var envelope struct {
		List []struct {
			ID    string                 `json:"id"`
			Value map[string]interface{} `json:"value"`
		} `json:"list"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode list envelope: %w (body: %s)", err, truncate(string(body), 200))
	}
	out := make([]string, 0, len(envelope.List))
	for _, item := range envelope.List {
		if item.ID != "" {
			out = append(out, item.ID)
			continue
		}
		if item.Value != nil {
			if id, ok := item.Value["id"].(string); ok && id != "" {
				out = append(out, id)
			}
		}
	}
	return out, nil
}

// deleteByPath issues a DELETE with `force=true` query to bypass server-side
// confirmation. Returns nil for 2xx and 404 (already gone).
func deleteByPath(useRuntime bool, path string) error {
	withForce := path
	if strings.Contains(withForce, "?") {
		withForce += "&force=true"
	} else {
		withForce += "?force=true"
	}
	var resp *http.Response
	var err error
	if useRuntime {
		resp, err = runtimeAdminAPI(http.MethodDelete, withForce, nil)
	} else {
		resp, err = adminAPI(http.MethodDelete, withForce, nil)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DELETE %s -> %d: %s", withForce, resp.StatusCode, truncate(string(body), 200))
	}
	return nil
}

// Per-resource list/delete functions. Each is a thin wrapper so the cleanup
// target table is data-driven.

func listRouteIDs() ([]string, error) {
	return listIDsFromRuntime("/apisix/admin/routes")
}
func deleteRouteByID(id string) error {
	return deleteByPath(true, fmt.Sprintf("/apisix/admin/routes/%s", id))
}

func listServiceIDs() ([]string, error) {
	return listIDsFromRuntime("/apisix/admin/services")
}
func deleteServiceByID(id string) error {
	return deleteByPath(true, fmt.Sprintf("/apisix/admin/services/%s", id))
}

func listConsumerIDs() ([]string, error) {
	ids, err := listIDsFromRuntime("/apisix/admin/consumers")
	if err != nil {
		return nil, err
	}
	// Consumers in APISIX are keyed by `username`. The wrapped value may put
	// the username under "username" rather than "id" — handle that here.
	if len(ids) == 0 {
		ids, err = listConsumerUsernames()
	}
	return ids, err
}
func listConsumerUsernames() ([]string, error) {
	resp, err := runtimeAdminAPI(http.MethodGet, "/apisix/admin/consumers", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET consumers -> %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var env struct {
		List []struct {
			Username string                 `json:"username"`
			Value    map[string]interface{} `json:"value"`
		} `json:"list"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(env.List))
	for _, item := range env.List {
		if item.Username != "" {
			out = append(out, item.Username)
			continue
		}
		if item.Value != nil {
			if u, ok := item.Value["username"].(string); ok && u != "" {
				out = append(out, u)
			}
		}
	}
	return out, nil
}
func deleteConsumerByID(id string) error {
	return deleteByPath(true, fmt.Sprintf("/apisix/admin/consumers/%s", id))
}

func listSSLIDs() ([]string, error) {
	return listIDsFromRuntime("/apisix/admin/ssls")
}
func deleteSSLByID(id string) error {
	return deleteByPath(true, fmt.Sprintf("/apisix/admin/ssls/%s", id))
}

func listSecretIDs() ([]string, error) {
	return listIDsFromAdmin("/api/secrets")
}
func deleteSecretByID(id string) error {
	return deleteByPath(false, fmt.Sprintf("/api/secrets/%s", id))
}

func listGlobalRuleIDs() ([]string, error) {
	return listIDsFromRuntime("/apisix/admin/global_rules")
}
func deleteGlobalRuleByID(id string) error {
	return deleteByPath(true, fmt.Sprintf("/apisix/admin/global_rules/%s", id))
}

func listStreamRouteIDs() ([]string, error) {
	return listIDsFromRuntime("/apisix/admin/stream_routes")
}
func deleteStreamRouteByID(id string) error {
	return deleteByPath(true, fmt.Sprintf("/apisix/admin/stream_routes/%s", id))
}

func listProtoIDs() ([]string, error) {
	return listIDsFromAdmin("/api/protos")
}
func deleteProtoByID(id string) error {
	return deleteByPath(false, fmt.Sprintf("/api/protos/%s", id))
}

func listPluginMetadataIDs() ([]string, error) {
	return listIDsFromAdmin("/api/plugin_metadata")
}
func deletePluginMetadataByID(id string) error {
	return deleteByPath(false, fmt.Sprintf("/api/plugin_metadata/%s", id))
}
