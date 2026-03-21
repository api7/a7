package api

// GlobalRule represents a global rule in API7 EE (runtime).
// Global rules apply plugins to all routes.
type GlobalRule struct {
	ID      string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Plugins map[string]interface{} `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Labels  map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
}
