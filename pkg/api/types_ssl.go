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

const RedactedSSLKey = "<redacted>"

// RedactSSL returns a copy safe for CLI output.
func RedactSSL(ssl SSL) SSL {
	if ssl.Key != "" {
		ssl.Key = RedactedSSLKey
	}
	return ssl
}

// RedactSSLs returns copies safe for CLI output.
func RedactSSLs(ssls []SSL) []SSL {
	redacted := make([]SSL, len(ssls))
	for i, ssl := range ssls {
		redacted[i] = RedactSSL(ssl)
	}
	return redacted
}
