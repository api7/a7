package api

// StreamRoute represents a stream (L4) route in API7 EE (runtime).
type StreamRoute struct {
	ID         string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Desc       string                 `json:"desc,omitempty" yaml:"desc,omitempty"`
	RemoteAddr string                 `json:"remote_addr,omitempty" yaml:"remote_addr,omitempty"`
	ServerAddr string                 `json:"server_addr,omitempty" yaml:"server_addr,omitempty"`
	ServerPort int                    `json:"server_port,omitempty" yaml:"server_port,omitempty"`
	SNI        string                 `json:"sni,omitempty" yaml:"sni,omitempty"`
	UpstreamID string                 `json:"upstream_id,omitempty" yaml:"upstream_id,omitempty"`
	Upstream   map[string]interface{} `json:"upstream,omitempty" yaml:"upstream,omitempty"`
	Plugins    map[string]interface{} `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
}
