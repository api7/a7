package trace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveMethod(t *testing.T) {
	assert.Equal(t, "POST", resolveMethod("post", []string{"GET"}))
	assert.Equal(t, "GET", resolveMethod("", []string{"get"}))
	assert.Equal(t, "GET", resolveMethod("", nil))
}

func TestResolvePath(t *testing.T) {
	assert.Equal(t, "/custom", resolvePath("/custom", "/route", nil))
	assert.Equal(t, "/route", resolvePath("", "/route", nil))
	assert.Equal(t, "/match/*", resolvePath("", "", []string{"match/*"}))
	assert.Equal(t, "/", resolvePath("", "", nil))
}

func TestResolveHost(t *testing.T) {
	assert.Equal(t, "flag.example.com", resolveHost("flag.example.com", "route.example.com", []string{"hosts.example.com"}))
	assert.Equal(t, "hosts.example.com", resolveHost("", "route.example.com", []string{"hosts.example.com"}))
	assert.Equal(t, "route.example.com", resolveHost("", "route.example.com", nil))
	assert.Equal(t, "", resolveHost("", "", nil))
}

func TestParseHeaders(t *testing.T) {
	headers, err := parseHeaders([]string{"X-Test: value", "X-Trace: abc"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"X-Test":  "value",
		"X-Trace": "abc",
	}, headers)
}

func TestParseHeaders_InvalidValue(t *testing.T) {
	_, err := parseHeaders([]string{"broken-header"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected 'Key: Value'")
}

func TestJoinURLPath(t *testing.T) {
	got, err := joinURLPath("http://127.0.0.1:9080/base", "/trace")
	require.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1:9080/trace", got)
}

func TestJoinURLPath_InvalidURL(t *testing.T) {
	_, err := joinURLPath("://bad-url", "/trace")
	require.Error(t, err)
}

func TestBuildConfiguredPlugins(t *testing.T) {
	got := buildConfiguredPlugins(map[string]interface{}{
		"limit-req":     map[string]interface{}{},
		"proxy-rewrite": map[string]interface{}{},
	}, map[string]int{
		"limit-req":     1001,
		"proxy-rewrite": 1008,
	})

	require.Len(t, got, 2)
	assert.Equal(t, "proxy-rewrite", got[0].Name)
	assert.Equal(t, 1008, got[0].Priority)
	assert.Equal(t, "limit-req", got[1].Name)
}

func TestParsePluginHeader(t *testing.T) {
	assert.Equal(t, []string{"proxy-rewrite", "limit-req"}, parsePluginHeader("proxy-rewrite, limit-req"))
	assert.Nil(t, parsePluginHeader(""))
}

func TestRouteUpstream(t *testing.T) {
	assert.Equal(t, map[string]interface{}{"type": "roundrobin"}, routeUpstream(map[string]interface{}{"type": "roundrobin"}, ""))
	assert.Equal(t, map[string]string{"upstream_id": "ups-1"}, routeUpstream(nil, "ups-1"))
	assert.Nil(t, routeUpstream(nil, ""))
}

func TestResultHeaders(t *testing.T) {
	assert.Equal(t, map[string]string{"Host": "example.com"}, resultHeaders(nil, "example.com"))
	assert.Equal(t, map[string]string{"X-Test": "1", "Host": "example.com"}, resultHeaders(map[string]string{"X-Test": "1"}, "example.com"))
	assert.Nil(t, resultHeaders(nil, ""))
}
