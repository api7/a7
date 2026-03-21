package api

// Upstream represents an upstream in API7 EE.
type Upstream struct {
	ID           string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Name         string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Desc         string                 `json:"desc,omitempty" yaml:"desc,omitempty"`
	Type         string                 `json:"type,omitempty" yaml:"type,omitempty"`
	Nodes        map[string]int         `json:"nodes,omitempty" yaml:"nodes,omitempty"`
	Scheme       string                 `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	Retries      int                    `json:"retries,omitempty" yaml:"retries,omitempty"`
	Timeout      map[string]interface{} `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Labels       map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	Checks       map[string]interface{} `json:"checks,omitempty" yaml:"checks,omitempty"`
	PassHost     string                 `json:"pass_host,omitempty" yaml:"pass_host,omitempty"`
	UpstreamHost string                 `json:"upstream_host,omitempty" yaml:"upstream_host,omitempty"`
}
