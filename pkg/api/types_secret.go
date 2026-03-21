package api

// Secret represents a secret provider in API7 EE (runtime).
// Secrets integrate with external secret managers (vault, AWS, etc.).
// The ID format is typically "{manager}/{id}" e.g. "vault/my-secret".
type Secret struct {
	ID     string            `json:"id,omitempty" yaml:"id,omitempty"`
	URI    string            `json:"uri,omitempty" yaml:"uri,omitempty"`
	Prefix string            `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	Token  string            `json:"token,omitempty" yaml:"token,omitempty"`
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}
