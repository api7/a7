package api

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestUnwrapValueEnvelope_SingleResource(t *testing.T) {
	body := []byte(`{"value":{"id":"r1","name":"route-1"}}`)
	got := unwrapValueEnvelope(body)
	assert.JSONEq(t, `{"id":"r1","name":"route-1"}`, string(got))
}

func TestUnwrapValueEnvelope_ListResponsePassesThrough(t *testing.T) {
	body := []byte(`{"total":1,"list":[{"id":"r1"}]}`)
	got := unwrapValueEnvelope(body)
	assert.JSONEq(t, string(body), string(got))
}

func TestUnwrapValueEnvelope_InvalidJSONPassesThrough(t *testing.T) {
	body := []byte(`{not-json`)
	got := unwrapValueEnvelope(body)
	assert.Equal(t, body, got)
}

func TestDefaultTransport_TLSMode(t *testing.T) {
	rt := defaultTransport(true, "")
	transport, ok := rt.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)
	assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestAPIKeyTransport_RoundTripInjectsHeader(t *testing.T) {
	var headerValue string
	transport := &apiKeyTransport{
		apiKey: "token-123",
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			headerValue = req.Header.Get("X-API-KEY")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`ok`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.com/routes", nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, "token-123", headerValue)
	assert.Empty(t, req.Header.Get("X-API-KEY"))
}

func TestClientDo_UnwrapsSingleValueEnvelope(t *testing.T) {
	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "http://example.com/routes/1", req.URL.String())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"value":{"id":"1"}}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}, "http://example.com")

	body, err := client.Get("/routes/1", nil)
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"1"}`, string(body))
}

func TestClientDo_APIError(t *testing.T) {
	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(`{"error_msg":"bad request"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}, "http://example.com")

	body, err := client.Get("/routes", nil)
	require.Error(t, err)
	assert.Nil(t, body)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Equal(t, "bad request", apiErr.ErrorMsg)
}
