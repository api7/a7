package api

// Proto represents a protobuf definition in API7 EE (runtime).
// Protos are used for gRPC transcoding.
type Proto struct {
	ID      string            `json:"id,omitempty" yaml:"id,omitempty"`
	Desc    string            `json:"desc,omitempty" yaml:"desc,omitempty"`
	Content string            `json:"content,omitempty" yaml:"content,omitempty"`
	Labels  map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}
