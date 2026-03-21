package api

// PluginMetadata represents plugin-level metadata in API7 EE (runtime).
// Unlike plugin configs, metadata is per-plugin-name and not per-route.
// There is no list endpoint — each plugin name has at most one metadata entry.
type PluginMetadata struct {
	// PluginName is used as the path segment, not in the body.
	PluginName string                 `json:"-" yaml:"-"`
	ID         string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
