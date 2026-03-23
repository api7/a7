package api

// Route represents a published route in API7 EE (runtime).
type Route struct {
	ID         string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Name       string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Desc       string                 `json:"desc,omitempty" yaml:"desc,omitempty"`
	URI        string                 `json:"uri,omitempty" yaml:"uri,omitempty"`
	URIs       []string               `json:"uris,omitempty" yaml:"uris,omitempty"`
	Paths      []string               `json:"paths,omitempty" yaml:"paths,omitempty"`
	Methods    []string               `json:"methods,omitempty" yaml:"methods,omitempty"`
	Host       string                 `json:"host,omitempty" yaml:"host,omitempty"`
	Hosts      []string               `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	ServiceID  string                 `json:"service_id,omitempty" yaml:"service_id,omitempty"`
	UpstreamID string                 `json:"upstream_id,omitempty" yaml:"upstream_id,omitempty"`
	Upstream   map[string]interface{} `json:"upstream,omitempty" yaml:"upstream,omitempty"`
	Plugins    map[string]interface{} `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	Status     int                    `json:"status,omitempty" yaml:"status,omitempty"`
	Priority   int                    `json:"priority,omitempty" yaml:"priority,omitempty"`
}
