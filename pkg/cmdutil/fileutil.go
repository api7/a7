package cmdutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// ReadResourceFile reads a JSON or YAML file and returns the parsed payload.
// If the file starts with '{', it's treated as JSON; otherwise as YAML.
// Accepts "-" to read from stdin.
func ReadResourceFile(path string, stdin io.Reader) (map[string]interface{}, error) {
	var data []byte
	var err error

	if path == "-" {
		data, err = io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read stdin: %w", err)
		}
	} else {
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %q: %w", path, err)
		}
	}

	var payload map[string]interface{}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("file %q is empty", path)
	}

	if trimmed[0] == '{' || trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(trimmed, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	}

	return payload, nil
}

// WriteToFileOrStdout writes data to a file if path is non-empty, otherwise to stdout.
func WriteToFileOrStdout(path string, stdout io.Writer, format string, data interface{}) error {
	var out io.Writer = stdout
	if path != "" {
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file %q: %w", path, err)
		}
		defer f.Close()
		out = f
	}
	return NewExporter(format, out).Write(data)
}
