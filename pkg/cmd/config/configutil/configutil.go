package configutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/api7/a7/pkg/api"
	"github.com/api7/a7/pkg/cmdutil"
)

type ResourceItem struct {
	Key   string                 `json:"key"`
	Value map[string]interface{} `json:"value,omitempty"`
}

type ResourceDiff struct {
	Create    []ResourceItem `json:"create,omitempty"`
	Update    []ResourceItem `json:"update,omitempty"`
	Delete    []ResourceItem `json:"delete,omitempty"`
	Unchanged []string       `json:"unchanged,omitempty"`
}

func (d ResourceDiff) CreateCount() int { return len(d.Create) }
func (d ResourceDiff) UpdateCount() int { return len(d.Update) }
func (d ResourceDiff) DeleteCount() int { return len(d.Delete) }

func (d ResourceDiff) HasDifferences() bool {
	return len(d.Create) > 0 || len(d.Update) > 0 || len(d.Delete) > 0
}

type DiffResult struct {
	Routes         ResourceDiff `json:"routes"`
	Services       ResourceDiff `json:"services"`
	Upstreams      ResourceDiff `json:"upstreams"`
	Consumers      ResourceDiff `json:"consumers"`
	SSL            ResourceDiff `json:"ssl"`
	GlobalRules    ResourceDiff `json:"global_rules"`
	PluginConfigs  ResourceDiff `json:"plugin_configs"`
	ConsumerGroups ResourceDiff `json:"consumer_groups"`
	StreamRoutes   ResourceDiff `json:"stream_routes"`
	Protos         ResourceDiff `json:"protos"`
	Secrets        ResourceDiff `json:"secrets"`
	PluginMetadata ResourceDiff `json:"plugin_metadata"`
}

func (r *DiffResult) HasDifferences() bool {
	if r == nil {
		return false
	}
	for _, section := range r.Sections() {
		if section.Diff.HasDifferences() {
			return true
		}
	}
	return false
}

type DiffSection struct {
	Name string
	Diff ResourceDiff
}

// Sections returns diff sections ordered for safe sync:
// base resources first, then dependents (routes reference upstreams/services).
func (r *DiffResult) Sections() []DiffSection {
	if r == nil {
		return nil
	}
	return []DiffSection{
		{Name: "upstreams", Diff: r.Upstreams},
		{Name: "services", Diff: r.Services},
		{Name: "consumers", Diff: r.Consumers},
		{Name: "consumer_groups", Diff: r.ConsumerGroups},
		{Name: "plugin_configs", Diff: r.PluginConfigs},
		{Name: "ssl", Diff: r.SSL},
		{Name: "global_rules", Diff: r.GlobalRules},
		{Name: "protos", Diff: r.Protos},
		{Name: "secrets", Diff: r.Secrets},
		{Name: "plugin_metadata", Diff: r.PluginMetadata},
		{Name: "routes", Diff: r.Routes},
		{Name: "stream_routes", Diff: r.StreamRoutes},
	}
}

func ReadConfigFile(file string) (api.ConfigFile, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return api.ConfigFile{}, fmt.Errorf("failed to read file: %w", err)
	}

	var cfg api.ConfigFile
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		if err := json.Unmarshal(trimmed, &cfg); err != nil {
			return api.ConfigFile{}, fmt.Errorf("failed to parse JSON file: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(trimmed, &cfg); err != nil {
			return api.ConfigFile{}, fmt.Errorf("failed to parse YAML file: %w", err)
		}
	}

	return cfg, nil
}

// FetchRemoteConfig fetches all runtime resources from API7 EE
// for the given gateway group and assembles them into a ConfigFile.
func FetchRemoteConfig(client *api.Client, gatewayGroup string) (*api.ConfigFile, error) {
	query := map[string]string{}
	if gatewayGroup != "" {
		query["gateway_group_id"] = gatewayGroup
	}

	routes, err := fetchPaginated[api.Route](client, "/apisix/admin/routes", query)
	if err != nil {
		return nil, err
	}
	services, err := fetchPaginated[api.Service](client, "/apisix/admin/services", query)
	if err != nil {
		return nil, err
	}
	upstreams, err := fetchPaginated[api.Upstream](client, "/apisix/admin/upstreams", query)
	if err != nil {
		return nil, err
	}
	consumers, err := fetchPaginated[api.Consumer](client, "/apisix/admin/consumers", query)
	if err != nil {
		return nil, err
	}
	ssl, err := fetchPaginated[api.SSL](client, "/apisix/admin/ssls", query)
	if err != nil {
		return nil, err
	}
	globalRules, err := fetchPaginated[api.GlobalRule](client, "/apisix/admin/global_rules", query)
	if err != nil {
		return nil, err
	}
	pluginConfigs, err := fetchPaginated[api.PluginConfig](client, "/apisix/admin/plugin_configs", query)
	if err != nil {
		return nil, err
	}
	consumerGroups, err := fetchPaginated[api.ConsumerGroup](client, "/apisix/admin/consumer_groups", query)
	if err != nil {
		return nil, err
	}
	streamRoutes, err := fetchPaginated[api.StreamRoute](client, "/apisix/admin/stream_routes", query)
	if err != nil {
		return nil, err
	}
	protos, err := fetchPaginated[api.Proto](client, "/apisix/admin/protos", query)
	if err != nil {
		return nil, err
	}
	secrets, err := fetchPaginated[api.Secret](client, "/apisix/admin/secrets", query)
	if err != nil {
		return nil, err
	}

	pluginMetadata, err := fetchPluginMetadata(client, query)
	if err != nil {
		return nil, err
	}

	remote := &api.ConfigFile{
		Version:        "1",
		Routes:         stripTimestampsFromSlice(routes),
		Services:       stripTimestampsFromSlice(services),
		Upstreams:      stripTimestampsFromSlice(upstreams),
		Consumers:      stripTimestampsFromSlice(consumers),
		SSL:            stripTimestampsFromSlice(ssl),
		GlobalRules:    stripTimestampsFromSlice(globalRules),
		PluginConfigs:  stripTimestampsFromSlice(pluginConfigs),
		ConsumerGroups: stripTimestampsFromSlice(consumerGroups),
		StreamRoutes:   stripTimestampsFromSlice(streamRoutes),
		Protos:         stripTimestampsFromSlice(protos),
		Secrets:        stripTimestampsFromSlice(secrets),
		PluginMetadata: pluginMetadata,
	}

	return remote, nil
}

func ComputeDiff(local, remote api.ConfigFile) (*DiffResult, error) {
	type diffSpec struct {
		local    interface{}
		remote   interface{}
		keyField string
		name     string
	}

	specs := []diffSpec{
		{local.Routes, remote.Routes, "id", "routes"},
		{local.Services, remote.Services, "id", "services"},
		{local.Upstreams, remote.Upstreams, "id", "upstreams"},
		{local.Consumers, remote.Consumers, "username", "consumers"},
		{local.SSL, remote.SSL, "id", "ssl"},
		{local.GlobalRules, remote.GlobalRules, "id", "global_rules"},
		{local.PluginConfigs, remote.PluginConfigs, "id", "plugin_configs"},
		{local.ConsumerGroups, remote.ConsumerGroups, "id", "consumer_groups"},
		{local.StreamRoutes, remote.StreamRoutes, "id", "stream_routes"},
		{local.Protos, remote.Protos, "id", "protos"},
		{local.Secrets, remote.Secrets, "id", "secrets"},
		{local.PluginMetadata, remote.PluginMetadata, "plugin_name", "plugin_metadata"},
	}

	diffs := make([]ResourceDiff, len(specs))
	for i, s := range specs {
		localMaps, err := toMapSlice(s.local)
		if err != nil {
			return nil, err
		}
		remoteMaps, err := toMapSlice(s.remote)
		if err != nil {
			return nil, err
		}
		d, err := diffByKey(localMaps, remoteMaps, s.keyField, s.name)
		if err != nil {
			return nil, err
		}
		diffs[i] = d
	}

	return &DiffResult{
		Routes:         diffs[0],
		Services:       diffs[1],
		Upstreams:      diffs[2],
		Consumers:      diffs[3],
		SSL:            diffs[4],
		GlobalRules:    diffs[5],
		PluginConfigs:  diffs[6],
		ConsumerGroups: diffs[7],
		StreamRoutes:   diffs[8],
		Protos:         diffs[9],
		Secrets:        diffs[10],
		PluginMetadata: diffs[11],
	}, nil
}

func FormatDiffSummary(result *DiffResult) string {
	if result == nil || !result.HasDifferences() {
		return "No differences found.\n"
	}

	var b strings.Builder
	b.WriteString("Differences found:\n")
	for _, section := range result.Sections() {
		d := section.Diff
		fmt.Fprintf(&b, "%s: create=%d update=%d delete=%d unchanged=%d\n",
			section.Name, len(d.Create), len(d.Update), len(d.Delete), len(d.Unchanged))
		for _, item := range d.Create {
			fmt.Fprintf(&b, "  CREATE %s\n", item.Key)
		}
		for _, item := range d.Update {
			fmt.Fprintf(&b, "  UPDATE %s\n", item.Key)
		}
		for _, item := range d.Delete {
			fmt.Fprintf(&b, "  DELETE %s\n", item.Key)
		}
	}

	return b.String()
}

func diffByKey(localItems, remoteItems []map[string]interface{}, keyField, resourceName string) (ResourceDiff, error) {
	localByKey := make(map[string]map[string]interface{}, len(localItems))
	for i, item := range localItems {
		key, err := extractKey(item, keyField)
		if err != nil {
			return ResourceDiff{}, fmt.Errorf("%s[%d]: %w", resourceName, i, err)
		}
		localByKey[key] = normalizeMap(item)
	}

	remoteByKey := make(map[string]map[string]interface{}, len(remoteItems))
	for i, item := range remoteItems {
		key, err := extractKey(item, keyField)
		if err != nil {
			return ResourceDiff{}, fmt.Errorf("remote %s[%d]: %w", resourceName, i, err)
		}
		remoteByKey[key] = normalizeMap(item)
	}

	result := ResourceDiff{
		Create:    make([]ResourceItem, 0),
		Update:    make([]ResourceItem, 0),
		Delete:    make([]ResourceItem, 0),
		Unchanged: make([]string, 0),
	}

	for key, localItem := range localByKey {
		remoteItem, ok := remoteByKey[key]
		if !ok {
			result.Create = append(result.Create, ResourceItem{Key: key, Value: localItem})
			continue
		}

		if reflect.DeepEqual(localItem, remoteItem) {
			result.Unchanged = append(result.Unchanged, key)
		} else {
			result.Update = append(result.Update, ResourceItem{Key: key, Value: localItem})
		}
	}

	for key, remoteItem := range remoteByKey {
		if _, ok := localByKey[key]; ok {
			continue
		}
		result.Delete = append(result.Delete, ResourceItem{Key: key, Value: remoteItem})
	}

	return result, nil
}

func extractKey(item map[string]interface{}, field string) (string, error) {
	raw, ok := item[field]
	if !ok {
		return "", fmt.Errorf("missing %q field", field)
	}
	key := strings.TrimSpace(fmt.Sprintf("%v", raw))
	if key == "" || key == "<nil>" {
		return "", fmt.Errorf("empty %q field", field)
	}
	return key, nil
}

func normalizeMap(item map[string]interface{}) map[string]interface{} {
	b, _ := json.Marshal(item)
	var out map[string]interface{}
	_ = json.Unmarshal(b, &out)
	stripTimestamps(out)
	return out
}

func stripTimestamps(v interface{}) {
	switch typed := v.(type) {
	case map[string]interface{}:
		delete(typed, "create_time")
		delete(typed, "update_time")
		for _, vv := range typed {
			stripTimestamps(vv)
		}
	case []interface{}:
		for _, vv := range typed {
			stripTimestamps(vv)
		}
	}
}

// stripTimestampsFromSlice marshals items through JSON to strip
// create_time/update_time fields. Returns nil for empty input.
func stripTimestampsFromSlice[T any](items []T) []T {
	if len(items) == 0 {
		return nil
	}
	b, err := json.Marshal(items)
	if err != nil {
		return items
	}
	var maps []map[string]interface{}
	if err := json.Unmarshal(b, &maps); err != nil {
		return items
	}
	for _, m := range maps {
		stripTimestamps(m)
	}
	cleaned, err := json.Marshal(maps)
	if err != nil {
		return items
	}
	var out []T
	if err := json.Unmarshal(cleaned, &out); err != nil {
		return items
	}
	return out
}

func toMapSlice(items interface{}) ([]map[string]interface{}, error) {
	b, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resources: %w", err)
	}
	if len(b) == 0 || string(b) == "null" {
		return []map[string]interface{}{}, nil
	}
	var out []map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("failed to convert resources: %w", err)
	}
	if out == nil {
		return []map[string]interface{}{}, nil
	}
	return out, nil
}

// fetchPaginated fetches all items from a paginated API7 EE list endpoint.
// API7 EE returns ListResponse[T] with .List []T directly (no ListItem wrapper).
func fetchPaginated[T any](client *api.Client, path string, extraQuery map[string]string) ([]T, error) {
	page := 1
	pageSize := 500
	var items []T

	for {
		query := map[string]string{
			"page":      fmt.Sprintf("%d", page),
			"page_size": fmt.Sprintf("%d", pageSize),
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

func fetchPluginMetadata(client *api.Client, query map[string]string) ([]api.PluginMetadataEntry, error) {
	body, err := client.Get("/apisix/admin/plugins/list", query)
	if err != nil {
		if cmdutil.IsOptionalResourceError(err) {
			return nil, nil
		}
		return nil, err
	}

	var plugins []string
	if err := json.Unmarshal(body, &plugins); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var result []api.PluginMetadataEntry
	for _, pluginName := range plugins {
		metaQuery := map[string]string{}
		for k, v := range query {
			metaQuery[k] = v
		}
		metadataBody, err := client.Get(fmt.Sprintf("/apisix/admin/plugin_metadata/%s", pluginName), metaQuery)
		if err != nil {
			if cmdutil.IsNotFound(err) {
				continue
			}
			// Skip plugins that don't support metadata (e.g., "plugin doesn't
			// have metadata_schema"). This is common in API7 EE where some
			// plugins expose the metadata endpoint but return an error.
			errMsg := err.Error()
			if strings.Contains(errMsg, "metadata_schema") ||
				strings.Contains(errMsg, "doesn't have") ||
				cmdutil.IsOptionalResourceError(err) {
				continue
			}
			return nil, err
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(metadataBody, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		entry := api.PluginMetadataEntry{}
		for k, v := range raw {
			entry[k] = v
		}
		entry["plugin_name"] = pluginName
		delete(entry, "create_time")
		delete(entry, "update_time")
		result = append(result, entry)
	}

	return result, nil
}
