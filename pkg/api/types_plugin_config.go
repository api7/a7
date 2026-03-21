package api

// PluginConfig represents a reusable plugin configuration in API7 EE (runtime).
// Plugin configs can be referenced by routes to share plugin settings.
type PluginConfig struct {
	ID      string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Desc    string                 `json:"desc,omitempty" yaml:"desc,omitempty"`
	Plugins map[string]interface{} `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Labels  map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
}
