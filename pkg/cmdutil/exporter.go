package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// ValidateOutputFormat returns nil when the given --output value is one of
// the four shapes the CLI claims to support: empty (default table), "table",
// "json", "yaml". Anything else (typos, abbreviations, unknown formats) is
// rejected with a clear message listing the valid set.
//
// Every command that accepts `-o`/`--output` should call this before
// dispatching, so a typo like `-o jzon` fails fast with a helpful error
// instead of silently falling back to table rendering.
func ValidateOutputFormat(format string) error {
	switch format {
	case "", "table", "json", "yaml":
		return nil
	default:
		return fmt.Errorf("unsupported output format: %q (valid: table, json, yaml)", format)
	}
}

// IsStructuredOutput returns true when the format should be rendered through
// the JSON/YAML exporter, false when the command should fall through to its
// own table renderer. Pairs with ValidateOutputFormat.
func IsStructuredOutput(format string) bool {
	return format == "json" || format == "yaml"
}

// Exporter formats and writes data to the given writer.
type Exporter struct {
	format string
	writer io.Writer
}

// NewExporter creates a new Exporter for the given format and writer.
// Supported formats: "json", "yaml".
func NewExporter(format string, writer io.Writer) *Exporter {
	return &Exporter{
		format: format,
		writer: writer,
	}
}

// Write formats and writes the given data.
func (e *Exporter) Write(data interface{}) error {
	switch e.format {
	case "json":
		return e.writeJSON(data)
	case "yaml":
		return e.writeYAML(data)
	default:
		return fmt.Errorf("unsupported output format: %s", e.format)
	}
}

// WriteAPIResponse decodes a JSON API response body and writes it via the
// exporter. Writing a raw []byte (e.g. json.RawMessage) directly would cause
// yaml.v3 to emit the bytes as a sequence of integers; decoding first ensures
// both -o yaml and -o json render a proper object.
func (e *Exporter) WriteAPIResponse(body []byte) error {
	var decoded interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return e.Write(decoded)
}

func (e *Exporter) writeJSON(data interface{}) error {
	enc := json.NewEncoder(e.writer)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func (e *Exporter) writeYAML(data interface{}) error {
	enc := yaml.NewEncoder(e.writer)
	defer enc.Close()
	return enc.Encode(data)
}
