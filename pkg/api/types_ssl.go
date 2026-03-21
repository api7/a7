package api

// SSL represents an SSL certificate in API7 EE.
type SSL struct {
	ID     string            `json:"id,omitempty" yaml:"id,omitempty"`
	Cert   string            `json:"cert,omitempty" yaml:"cert,omitempty"`
	Key    string            `json:"key,omitempty" yaml:"key,omitempty"`
	SNIs   []string          `json:"snis,omitempty" yaml:"snis,omitempty"`
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Status int               `json:"status,omitempty" yaml:"status,omitempty"`
	Type   string            `json:"type,omitempty" yaml:"type,omitempty"`
}
