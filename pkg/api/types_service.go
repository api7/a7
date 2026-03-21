package api

// Service represents a published service in API7 EE (runtime).
type Service struct {
	ID              string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Name            string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Desc            string                 `json:"desc,omitempty" yaml:"desc,omitempty"`
	Labels          map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	Upstream        map[string]interface{} `json:"upstream,omitempty" yaml:"upstream,omitempty"`
	UpstreamID      string                 `json:"upstream_id,omitempty" yaml:"upstream_id,omitempty"`
	Plugins         map[string]interface{} `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Hosts           []string               `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	EnableWebsocket bool                   `json:"enable_websocket,omitempty" yaml:"enable_websocket,omitempty"`
}
