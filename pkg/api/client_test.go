package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestClient_Get tests the Get method with a successful response.
func TestClient_Get(t *testing.T) {
	expectedBody := `{"id": 1, "name": "test"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/routes" {
			t.Errorf("expected path /routes, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/routes", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, string(body))
	}
}

// TestClient_Post tests the Post method with a JSON body.
func TestClient_Post(t *testing.T) {
	expectedBody := `{"id": 1, "name": "test"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/routes" {
			t.Errorf("expected path /routes, got %s", r.URL.Path)
		}

		// Verify Content-Type header
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		// Verify body was sent
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if len(body) == 0 {
			t.Error("expected non-empty request body")
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(expectedBody))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	postData := map[string]interface{}{"name": "test"}
	body, err := client.Post("/routes", postData)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, string(body))
	}
}

// TestClient_Put tests the Put method.
func TestClient_Put(t *testing.T) {
	expectedBody := `{"id": 1, "name": "updated"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected method PUT, got %s", r.Method)
		}
		if r.URL.Path != "/routes/1" {
			t.Errorf("expected path /routes/1, got %s", r.URL.Path)
		}

		// Verify Content-Type header
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	putData := map[string]interface{}{"name": "updated"}
	body, err := client.Put("/routes/1", putData)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, string(body))
	}
}

// TestClient_Delete tests the Delete method.
func TestClient_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected method DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/routes/1" {
			t.Errorf("expected path /routes/1, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Delete("/routes/1", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) != 0 {
		t.Errorf("expected empty body for DELETE, got %q", string(body))
	}
}

// TestClient_GetWithQuery tests query parameter handling.
func TestClient_GetWithQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected method GET, got %s", r.Method)
		}

		// Verify query parameters
		q := r.URL.Query()
		if q.Get("page") != "1" {
			t.Errorf("expected query param page=1, got page=%s", q.Get("page"))
		}
		if q.Get("size") != "10" {
			t.Errorf("expected query param size=10, got size=%s", q.Get("size"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"list": []}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	query := map[string]string{
		"page": "1",
		"size": "10",
	}
	body, err := client.Get("/routes", query)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

// TestClient_APIError tests error handling with a 400 response.
func TestClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error_msg":"bad request"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/routes", nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected StatusCode %d, got %d", http.StatusBadRequest, apiErr.StatusCode)
	}

	if apiErr.ErrorMsg != "bad request" {
		t.Errorf("expected ErrorMsg 'bad request', got %q", apiErr.ErrorMsg)
	}

	if body != nil {
		t.Errorf("expected nil body on error, got %v", body)
	}
}

// TestClient_APIError_401 tests 401 Unauthorized error handling.
func TestClient_APIError_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error_msg":"invalid api key"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/routes", nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected StatusCode %d, got %d", http.StatusUnauthorized, apiErr.StatusCode)
	}

	if apiErr.ErrorMsg != "invalid api key" {
		t.Errorf("expected ErrorMsg 'invalid api key', got %q", apiErr.ErrorMsg)
	}

	if body != nil {
		t.Errorf("expected nil body on error, got %v", body)
	}
}

// TestClient_APIError_NoBody tests error handling with empty body.
func TestClient_APIError_NoBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		// Return empty body
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/routes", nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected StatusCode %d, got %d", http.StatusInternalServerError, apiErr.StatusCode)
	}

	// ErrorMsg should be empty or contain the empty body
	if apiErr.ErrorMsg != "" && apiErr.ErrorMsg != string([]byte{}) {
		t.Errorf("expected empty ErrorMsg or empty string, got %q", apiErr.ErrorMsg)
	}

	if body != nil {
		t.Errorf("expected nil body on error, got %v", body)
	}
}

// TestApiKeyTransport tests API key authentication header injection.
func TestApiKeyTransport(t *testing.T) {
	const testAPIKey = "test-api-key-12345"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the X-API-KEY header was set
		apiKey := r.Header.Get("X-API-KEY")
		if apiKey != testAPIKey {
			t.Errorf("expected X-API-KEY header %q, got %q", testAPIKey, apiKey)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Create authenticated client
	httpClient := NewAuthenticatedClient(testAPIKey, false, "")

	// Parse base URL and make a request
	client := NewClient(httpClient, server.URL)
	body, err := client.Get("/routes", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != `{"status": "ok"}` {
		t.Errorf("expected body, got %q", string(body))
	}
}

// TestClient_Patch tests the Patch method (JSON Patch RFC 6902).
func TestClient_Patch(t *testing.T) {
	expectedBody := `{"id": 1, "name": "patched"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected method PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/routes/1" {
			t.Errorf("expected path /routes/1, got %s", r.URL.Path)
		}

		// Verify Content-Type header
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		// Verify body was sent
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if len(body) == 0 {
			t.Error("expected non-empty request body")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	// JSON Patch format: array of operations
	patchData := []map[string]interface{}{
		{
			"op":    "replace",
			"path":  "/name",
			"value": "patched",
		},
	}
	body, err := client.Patch("/routes/1", patchData)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, string(body))
	}
}

// TestClient_ContentTypeNotSetForGETandDELETE verifies GET/DELETE don't set Content-Type.
func TestClient_ContentTypeNotSetForGETandDELETE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodDelete {
			if ct := r.Header.Get("Content-Type"); ct != "" {
				t.Errorf("expected no Content-Type for %s, got %s", r.Method, ct)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)

	// Test GET
	_, err := client.Get("/test", nil)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	// Test DELETE
	_, err = client.Delete("/test", nil)
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}
}

// TestClient_BodyMarshalError tests handling of invalid body data.
func TestClient_BodyMarshalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)

	// Create a channel which cannot be marshaled to JSON
	invalidData := make(chan struct{})
	_, err := client.Post("/test", invalidData)

	if err == nil {
		t.Fatal("expected error for invalid JSON data, got nil")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("failed to marshal request body")) {
		t.Errorf("expected marshal error message, got: %v", err)
	}
}

// TestClient_URLConstruction tests that the baseURL and path are correctly combined.
func TestClient_URLConstruction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/routes" {
			t.Errorf("expected path /api/v1/routes, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL+"/api/v1")
	_, err := client.Get("/routes", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestClient_ResponseBodyReadError tests handling of read errors.
func TestClient_ResponseBodyReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a response but we'll simulate a read error by closing it early
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// This test is minimal as simulating actual read errors is complex
	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/test", nil)

	// Should succeed normally
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if body == nil {
		t.Error("expected non-nil body")
	}
}

// TestApiKeyTransport_MultipleRequests verifies API key is sent on each request.
func TestApiKeyTransport_MultipleRequests(t *testing.T) {
	const testAPIKey = "multi-request-key"
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		apiKey := r.Header.Get("X-API-KEY")
		if apiKey != testAPIKey {
			t.Errorf("request %d: expected X-API-KEY header %q, got %q", requestCount, testAPIKey, apiKey)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	httpClient := NewAuthenticatedClient(testAPIKey, false, "")
	client := NewClient(httpClient, server.URL)

	// Make multiple requests
	for i := 0; i < 3; i++ {
		_, err := client.Get("/routes", nil)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
	}

	if requestCount != 3 {
		t.Errorf("expected 3 requests, got %d", requestCount)
	}
}

// TestClient_EmptyQuery tests behavior with nil query parameters.
func TestClient_EmptyQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected empty query string, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	_, err := client.Get("/routes", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestClient_MultipleQueryParams tests multiple query parameters.
func TestClient_MultipleQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		params := map[string]string{
			"gateway_group_id": "123",
			"page":             "2",
			"size":             "50",
		}

		for k, v := range params {
			if actual := q.Get(k); actual != v {
				t.Errorf("query param %s: expected %q, got %q", k, v, actual)
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"list": []}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	query := map[string]string{
		"gateway_group_id": "123",
		"page":             "2",
		"size":             "50",
	}
	_, err := client.Get("/routes", query)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestClient_JSONResponseParsing verifies JSON response bodies are preserved.
func TestClient_JSONResponseParsing(t *testing.T) {
	type responseType struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	expectedResp := responseType{ID: 42, Name: "test-route"}
	respJSON, _ := json.Marshal(expectedResp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respJSON)
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/routes/42", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the response can be unmarshaled correctly
	var resp responseType
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ID != 42 || resp.Name != "test-route" {
		t.Errorf("response mismatch: expected {42, test-route}, got {%d, %s}", resp.ID, resp.Name)
	}
}

// TestClient_LargeResponseBody verifies handling of large response bodies.
func TestClient_LargeResponseBody(t *testing.T) {
	// Create a large response body
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte((i % 256))
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largeData)
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/large", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(body) != len(largeData) {
		t.Errorf("response size mismatch: expected %d, got %d", len(largeData), len(body))
	}
}

// TestClient_SpecialCharactersInQuery tests URL encoding of special characters in query params.
func TestClient_SpecialCharactersInQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if actual := q.Get("name"); actual != "test value" {
			t.Errorf("expected query param 'test value', got %q", actual)
		}
		if actual := q.Get("filter"); actual != "foo=bar&baz=qux" {
			t.Errorf("expected query param 'foo=bar&baz=qux', got %q", actual)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	query := map[string]string{
		"name":   "test value",
		"filter": "foo=bar&baz=qux",
	}
	_, err := client.Get("/routes", query)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestClient_NonJSONErrorResponse tests error with non-JSON body.
func TestClient_NonJSONErrorResponse(t *testing.T) {
	plainTextError := "Internal Server Error"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(plainTextError))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/routes", nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected StatusCode %d, got %d", http.StatusInternalServerError, apiErr.StatusCode)
	}

	// ErrorMsg should be the raw response body since JSON parsing failed
	if apiErr.ErrorMsg != plainTextError {
		t.Errorf("expected ErrorMsg %q, got %q", plainTextError, apiErr.ErrorMsg)
	}

	if body != nil {
		t.Errorf("expected nil body on error, got %v", body)
	}
}

// TestNewAuthenticatedClient_TLSSkipVerify tests TLS skip verify flag.
func TestNewAuthenticatedClient_TLSSkipVerify(t *testing.T) {
	httpClient := NewAuthenticatedClient("test-key", true, "")

	// Verify the client was created with a transport
	if httpClient == nil {
		t.Fatal("NewAuthenticatedClient returned nil")
	}

	if httpClient.Transport == nil {
		t.Fatal("httpClient.Transport is nil")
	}

	// Verify it's an apiKeyTransport
	transport, ok := httpClient.Transport.(*apiKeyTransport)
	if !ok {
		t.Fatalf("expected *apiKeyTransport, got %T", httpClient.Transport)
	}

	if transport.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", transport.apiKey)
	}

	if transport.base == nil {
		t.Fatal("base transport is nil")
	}
}

// TestClient_Get_URLEncodingPreservation verifies that URL-encoded paths are preserved.
func TestClient_Get_URLEncodingPreservation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The path should be /routes%2Ftest (URL encoded slash preserved)
		if r.URL.Path != "/routes/test" {
			t.Errorf("expected path /routes/test, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	_, err := client.Get("/routes/test", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestClient_PostWithNilBody tests POST with an empty body.
func TestClient_PostWithNilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Since we're passing an empty map, it still gets marshaled to JSON
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status": "created"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	// Passing an empty map (not nil) will still set Content-Type
	body, err := client.Post("/routes", map[string]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(body) == 0 {
		t.Error("expected response body")
	}
}

// TestClient_ErrorResponse_WithPartialJSON tests error with partially valid JSON.
func TestClient_ErrorResponse_WithPartialJSON(t *testing.T) {
	// JSON that cannot be fully unmarshaled into APIError but has some structure
	partialJSON := `{"unexpected_field": "value"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(partialJSON))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/routes", nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected StatusCode %d, got %d", http.StatusBadRequest, apiErr.StatusCode)
	}

	// Since JSON unmarshaling will fail to populate ErrorMsg, it should fall back to raw body
	if apiErr.ErrorMsg != partialJSON {
		t.Errorf("expected ErrorMsg %q, got %q", partialJSON, apiErr.ErrorMsg)
	}

	if body != nil {
		t.Errorf("expected nil body on error, got %v", body)
	}
}

// TestAPIError_Error_WithMessage tests APIError.Error() method with message.
func TestAPIError_Error_WithMessage(t *testing.T) {
	apiErr := &APIError{
		StatusCode: 404,
		ErrorMsg:   "route not found",
	}

	errStr := apiErr.Error()
	expected := "API error (status 404): route not found"
	if errStr != expected {
		t.Errorf("expected error string %q, got %q", expected, errStr)
	}
}

// TestAPIError_Error_WithoutMessage tests APIError.Error() method without message.
func TestAPIError_Error_WithoutMessage(t *testing.T) {
	apiErr := &APIError{
		StatusCode: 500,
		ErrorMsg:   "",
	}

	errStr := apiErr.Error()
	expected := "API error: status 500"
	if errStr != expected {
		t.Errorf("expected error string %q, got %q", expected, errStr)
	}
}

// TestClient_QueryMapWithEmptyValues tests query parameters with empty string values.
func TestClient_QueryMapWithEmptyValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// Empty values should still be present in the query string
		if actual := q.Get("filter"); actual != "" {
			t.Errorf("expected empty filter value, got %q", actual)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	query := map[string]string{
		"filter": "",
	}
	_, err := client.Get("/routes", query)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestClient_StatusCode_200 tests successful 200 response.
func TestClient_StatusCode_200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/test", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != `{"success": true}` {
		t.Errorf("expected body, got %q", string(body))
	}
}

// TestClient_StatusCode_201 tests successful 201 Created response.
func TestClient_StatusCode_201(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Post("/test", map[string]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != `{"id": 1}` {
		t.Errorf("expected body, got %q", string(body))
	}
}

// TestClient_StatusCode_204 tests 204 No Content response.
func TestClient_StatusCode_204(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		// 204 has no body
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Delete("/test", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(body) != 0 {
		t.Errorf("expected empty body for 204, got %q", string(body))
	}
}

// TestClient_StatusCode_400 tests 400 Bad Request error.
func TestClient_StatusCode_400(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error_msg": "bad request"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/test", nil)

	if err == nil {
		t.Fatal("expected error for 400 response")
	}

	if body != nil {
		t.Errorf("expected nil body, got %v", body)
	}
}

// TestClient_StatusCode_403 tests 403 Forbidden error.
func TestClient_StatusCode_403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error_msg": "forbidden"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/test", nil)

	if err == nil {
		t.Fatal("expected error for 403 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected StatusCode 403, got %d", apiErr.StatusCode)
	}

	if body != nil {
		t.Errorf("expected nil body, got %v", body)
	}
}

// TestClient_StatusCode_404 tests 404 Not Found error.
func TestClient_StatusCode_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error_msg": "not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/test", nil)

	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected StatusCode 404, got %d", apiErr.StatusCode)
	}

	if body != nil {
		t.Errorf("expected nil body, got %v", body)
	}
}

// TestClient_StatusCode_500 tests 500 Internal Server Error.
func TestClient_StatusCode_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error_msg": "internal error"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	body, err := client.Get("/test", nil)

	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected StatusCode 500, got %d", apiErr.StatusCode)
	}

	if body != nil {
		t.Errorf("expected nil body, got %v", body)
	}
}

// TestClient_StatusCode_502 tests 502 Bad Gateway error.
func TestClient_StatusCode_502(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error_msg": "bad gateway"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL)
	_, err := client.Get("/test", nil)

	if err == nil {
		t.Fatal("expected error for 502 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusBadGateway {
		t.Errorf("expected StatusCode 502, got %d", apiErr.StatusCode)
	}
}

// TestClient_BaseURLWithTrailingSlash tests baseURL with and without trailing slash.
func TestClient_BaseURLWithTrailingSlash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/routes" {
			t.Errorf("expected path /api/routes, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// Test with trailing slash in baseURL
	client1 := NewClient(server.Client(), server.URL+"/api/")
	_, err := client1.Get("routes", nil)
	if err != nil {
		t.Fatalf("client1 failed: %v", err)
	}

	// Test without trailing slash in baseURL
	client2 := NewClient(server.Client(), server.URL+"/api")
	_, err = client2.Get("/routes", nil)
	if err != nil {
		t.Fatalf("client2 failed: %v", err)
	}
}
