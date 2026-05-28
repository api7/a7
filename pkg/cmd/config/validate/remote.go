package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/api7/a7/pkg/api"
)

// remoteValidateRequest is the body sent to POST /apisix/admin/configs/validate.
// Field shape mirrors what adc's backend-api7 Validator sends, so the API7 EE
// backend can dry-run the schema check against its live validators.
//
// Reference: adc/libs/backend-api7/src/validator.ts (Validator.validate ->
// POST /apisix/admin/configs/validate).
type remoteValidateRequest struct {
	Routes         []api.Route               `json:"routes"`
	Services       []api.Service             `json:"services"`
	Consumers      []api.Consumer            `json:"consumers"`
	SSLs           []api.SSL                 `json:"ssls"`
	GlobalRules    []api.GlobalRule          `json:"global_rules"`
	StreamRoutes   []api.StreamRoute         `json:"stream_routes"`
	PluginMetadata []api.PluginMetadataEntry `json:"plugin_metadata"`
}

// remoteValidationError mirrors the per-resource error object the EE backend
// returns alongside a 400 response.
type remoteValidationError struct {
	ResourceType string `json:"resource_type"`
	Index        int    `json:"index"`
	Message      string `json:"message,omitempty"`
	Field        string `json:"field,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// remoteValidationResponse is the JSON body API7 EE returns on a 400.
type remoteValidationResponse struct {
	ErrorMsg string                  `json:"error_msg"`
	Errors   []remoteValidationError `json:"errors"`
}

// buildRemoteValidateBody projects a local ConfigFile into the body shape the
// EE validate endpoint expects. Plugin metadata entries have their "plugin_name"
// pseudo-field moved into "id", matching how a7's sync layer encodes them on
// PUT (see sync.putPathAndBody).
func buildRemoteValidateBody(cfg api.ConfigFile) remoteValidateRequest {
	body := remoteValidateRequest{
		Routes:         cfg.Routes,
		Services:       cfg.Services,
		Consumers:      cfg.Consumers,
		SSLs:           cfg.SSL,
		GlobalRules:    cfg.GlobalRules,
		StreamRoutes:   cfg.StreamRoutes,
		PluginMetadata: nil,
	}

	for _, entry := range cfg.PluginMetadata {
		cloned := make(api.PluginMetadataEntry, len(entry))
		for k, v := range entry {
			cloned[k] = v
		}
		if raw, ok := cloned["plugin_name"]; ok {
			if name, ok := raw.(string); ok && name != "" {
				cloned["id"] = name
			}
			delete(cloned, "plugin_name")
		}
		body.PluginMetadata = append(body.PluginMetadata, cloned)
	}

	return body
}

// validateRemote posts the config to API7 EE's /apisix/admin/configs/validate
// dry-run endpoint. On a 200 OK it returns nil errors. On a 400 it decodes the
// structured error list and returns one human-readable string per backend
// error. Any other failure (network, 5xx) is returned as a single error string.
//
// This bypasses api.Client.Post and uses the underlying http.Client directly so
// it can read the full 400 body — Client.Post collapses the body into
// APIError.ErrorMsg and would discard the per-resource "errors" array.
func validateRemote(httpClient *http.Client, baseURL string, cfg api.ConfigFile, gatewayGroup string) []string {
	body := buildRemoteValidateBody(cfg)

	url := baseURL + "/apisix/admin/configs/validate"
	if gatewayGroup != "" {
		url += "?gateway_group_id=" + gatewayGroup
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return []string{fmt.Sprintf("failed to encode remote validate request: %v", err)}
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return []string{fmt.Sprintf("failed to build remote validate request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return []string{fmt.Sprintf("remote validation request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return []string{fmt.Sprintf("failed to read remote validate response: %v", err)}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Only 400 carries the structured per-resource error list. For other
	// non-2xx statuses surface the raw body so auth or availability issues
	// aren't swallowed.
	if resp.StatusCode != http.StatusBadRequest {
		return []string{fmt.Sprintf("remote validation failed (status %d): %s", resp.StatusCode, string(respBody))}
	}

	var parsed remoteValidationResponse
	if jsonErr := json.Unmarshal(respBody, &parsed); jsonErr != nil {
		return []string{fmt.Sprintf("remote validation failed (status 400): %s", string(respBody))}
	}

	if len(parsed.Errors) == 0 {
		msg := parsed.ErrorMsg
		if msg == "" {
			msg = string(respBody)
		}
		if msg == "" {
			msg = "remote validation failed"
		}
		return []string{msg}
	}

	out := make([]string, 0, len(parsed.Errors))
	for _, e := range parsed.Errors {
		out = append(out, formatRemoteError(e))
	}
	return out
}

func formatRemoteError(e remoteValidationError) string {
	prefix := e.ResourceType
	if prefix == "" {
		prefix = "resource"
	}
	if e.Field != "" {
		prefix = fmt.Sprintf("%s[%d].%s", prefix, e.Index, e.Field)
	} else {
		prefix = fmt.Sprintf("%s[%d]", prefix, e.Index)
	}
	msg := e.Message
	if msg == "" {
		msg = e.Reason
	}
	if msg == "" {
		return prefix
	}
	return fmt.Sprintf("%s: %s", prefix, msg)
}
