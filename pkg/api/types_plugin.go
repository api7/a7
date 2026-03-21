package api

// Plugin represents a plugin schema in API7 EE.
type Plugin struct {
	Name     string `json:"name" yaml:"name"`
	Priority int    `json:"priority,omitempty" yaml:"priority,omitempty"`
	Phase    string `json:"phase,omitempty" yaml:"phase,omitempty"`
	Version  string `json:"version,omitempty" yaml:"version,omitempty"`
}
