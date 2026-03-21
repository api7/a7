package api

// Consumer represents a consumer in API7 EE.
type Consumer struct {
	Username string                 `json:"username" yaml:"username"`
	Desc     string                 `json:"desc,omitempty" yaml:"desc,omitempty"`
	Labels   map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	Plugins  map[string]interface{} `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}
