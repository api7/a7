package api

// GatewayGroup represents an API7 EE Gateway Group.
type GatewayGroup struct {
	ID          string            `json:"id,omitempty" yaml:"id,omitempty"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Prefix      string            `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Status      int               `json:"status,omitempty" yaml:"status,omitempty"`
	CreatedAt   int64             `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	UpdatedAt   int64             `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
}
