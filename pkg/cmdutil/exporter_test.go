package cmdutil

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExporter_WriteAPIResponse_YAML_DecodesObject(t *testing.T) {
	// The original bug: writing json.RawMessage([]byte) directly to the YAML
	// encoder serialized the bytes as a list of integers instead of a map.
	// WriteAPIResponse must decode first so YAML emits a proper object.
	body := []byte(`{"id":"r1","name":"demo","status":1}`)

	var buf bytes.Buffer
	if err := NewExporter("yaml", &buf).WriteAPIResponse(body); err != nil {
		t.Fatalf("WriteAPIResponse failed: %v", err)
	}

	out := buf.String()
	if strings.HasPrefix(strings.TrimSpace(out), "- ") {
		t.Fatalf("yaml output looks like a byte sequence, not an object: %q", out)
	}

	var decoded map[string]interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("yaml output is not a valid map: %v (output: %q)", err, out)
	}
	if decoded["id"] != "r1" || decoded["name"] != "demo" {
		t.Fatalf("yaml output missing expected fields: got %v", decoded)
	}
}

func TestExporter_WriteAPIResponse_JSON_RoundTripsObject(t *testing.T) {
	body := []byte(`{"id":"r1","name":"demo"}`)

	var buf bytes.Buffer
	if err := NewExporter("json", &buf).WriteAPIResponse(body); err != nil {
		t.Fatalf("WriteAPIResponse failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("json output is not a valid object: %v (output: %q)", err, buf.String())
	}
	if decoded["id"] != "r1" || decoded["name"] != "demo" {
		t.Fatalf("json output missing expected fields: got %v", decoded)
	}
}

func TestExporter_WriteAPIResponse_InvalidJSON_ReturnsError(t *testing.T) {
	body := []byte(`not json`)
	err := NewExporter("yaml", &bytes.Buffer{}).WriteAPIResponse(body)
	if err == nil {
		t.Fatal("expected error decoding invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Fatalf("error did not mention decoding: %v", err)
	}
}
