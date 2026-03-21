package api

// ServiceTemplate represents an API7 EE Service Template (design-time service).
type ServiceTemplate struct {
	ID          string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	Upstream    map[string]interface{} `json:"upstream,omitempty" yaml:"upstream,omitempty"`
	Plugins     map[string]interface{} `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Hosts       []string               `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	PathPrefix  string                 `json:"path_prefix,omitempty" yaml:"path_prefix,omitempty"`
	Status      int                    `json:"status,omitempty" yaml:"status,omitempty"`
	CreatedAt   int64                  `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	UpdatedAt   int64                  `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
}
