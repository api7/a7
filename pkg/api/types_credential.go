package api

// Credential represents a consumer credential in API7 EE (runtime).
// Credentials are scoped to a consumer (path: /consumers/{username}/credentials).
type Credential struct {
	ID      string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Desc    string                 `json:"desc,omitempty" yaml:"desc,omitempty"`
	Plugins map[string]interface{} `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Labels  map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
}
