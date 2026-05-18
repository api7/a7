package api

// ConfigFile is the declarative configuration file format for a7.
// It holds all runtime resources that can be dumped/synced.
type ConfigFile struct {
	Version        string          `json:"version" yaml:"version"`
	Routes         []Route         `json:"routes,omitempty" yaml:"routes,omitempty"`
	Services       []Service       `json:"services,omitempty" yaml:"services,omitempty"`
	Upstreams      []Upstream      `json:"upstreams,omitempty" yaml:"upstreams,omitempty"`
	Consumers      []Consumer      `json:"consumers,omitempty" yaml:"consumers,omitempty"`
	SSL            []SSL           `json:"ssl,omitempty" yaml:"ssl,omitempty"`
	GlobalRules    []GlobalRule    `json:"global_rules,omitempty" yaml:"global_rules,omitempty"`
	ConsumerGroups []ConsumerGroup `json:"consumer_groups,omitempty" yaml:"consumer_groups,omitempty"`
	// PluginConfigs and ServiceTemplates are captured only so the sections
	// can be explicitly rejected; API7 EE does not expose them as top-level
	// resources.
	PluginConfigs    []interface{}         `json:"plugin_configs,omitempty" yaml:"plugin_configs,omitempty"`
	ServiceTemplates []interface{}         `json:"service_templates,omitempty" yaml:"service_templates,omitempty"`
	StreamRoutes     []StreamRoute         `json:"stream_routes,omitempty" yaml:"stream_routes,omitempty"`
	Protos           []Proto               `json:"protos,omitempty" yaml:"protos,omitempty"`
	Secrets          []Secret              `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	PluginMetadata   []PluginMetadataEntry `json:"plugin_metadata,omitempty" yaml:"plugin_metadata,omitempty"`
}

// PluginMetadataEntry is a freeform map representing a plugin's metadata.
// The "plugin_name" key identifies which plugin the metadata belongs to.
type PluginMetadataEntry map[string]interface{}
