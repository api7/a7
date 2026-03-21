package api

// ConsumerGroup represents a consumer group in API7 EE (runtime).
// Consumer groups allow sharing plugin configurations across consumers.
type ConsumerGroup struct {
	ID      string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Desc    string                 `json:"desc,omitempty" yaml:"desc,omitempty"`
	Plugins map[string]interface{} `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Labels  map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
}
