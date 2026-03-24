//go:build e2e

// Package e2e provides end-to-end tests for the a7 CLI.
// These tests run against a real API7 Enterprise Edition instance and require
// the following environment variables:
//
//   - A7_ADMIN_URL: API7 EE Dashboard/control-plane URL (required)
//   - A7_TOKEN: API7 EE access token (required)
//   - A7_GATEWAY_GROUP: Gateway group name (default: "default")
//   - A7_GATEWAY_URL: Gateway data-plane URL (optional — gateway traffic tests skipped if empty)
//   - HTTPBIN_URL: httpbin URL (optional — traffic forwarding tests skipped if empty)
//
// Run with: go test -v -tags e2e -count=1 -timeout 10m ./test/e2e/...
package e2e

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	binaryPath   string
	adminURL     string // API7 EE Dashboard/control-plane URL (HTTPS)
	gatewayURL   string // API7 EE Gateway URL (HTTP)
	httpbinURL   string
	adminToken   string // API7 EE access token (a7ee prefix)
	gatewayGroup = "default"

	// httpClient with TLS skip verify for self-signed certs.
	insecureClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 30 * time.Second,
	}
)

func TestMain(m *testing.M) {
	adminURL = envOrDefault("A7_ADMIN_URL", "")
	gatewayURL = envOrDefault("A7_GATEWAY_URL", "")
	httpbinURL = envOrDefault("HTTPBIN_URL", "")
	adminToken = envOrDefault("A7_TOKEN", "")

	if g := os.Getenv("A7_GATEWAY_GROUP"); g != "" {
		gatewayGroup = g
	}

	if adminURL == "" {
		fmt.Fprintln(os.Stderr, "A7_ADMIN_URL environment variable is required for E2E tests")
		os.Exit(1)
	}

	if adminToken == "" {
		fmt.Fprintln(os.Stderr, "A7_TOKEN environment variable is required for E2E tests")
		os.Exit(1)
	}

	// Build the a7 binary into a temp directory.
	tmpDir, err := os.MkdirTemp("", "a7-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "a7")

	// Resolve the module root so `go build ./cmd/a7` works regardless of
	// the working directory the test runner uses.
	modRoot, err := resolveModuleRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve module root: %v\n", err)
		os.Exit(1)
	}

	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/a7")
	buildCmd.Dir = modRoot
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build a7 binary: %v\n", err)
		os.Exit(1)
	}

	// Wait for API7 EE Dashboard API to become healthy.
	// Try the /api/status endpoint first, fall back to /api/gateway_groups.
	healthURL := adminURL + "/api/gateway_groups"
	if err := waitForHealthy(healthURL, 120*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "API7 EE not ready: %v\n", err)
		os.Exit(1)
	}

	// Resolve the actual gateway group UUID. API7 EE uses UUID-style IDs,
	// not human-readable names like "default".
	if gatewayGroup == "default" || gatewayGroup == "" {
		ggID, err := resolveFirstGatewayGroupID()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to resolve gateway group ID: %v\n", err)
			os.Exit(1)
		}
		gatewayGroup = ggID
		fmt.Fprintf(os.Stderr, "Resolved gateway group ID: %s\n", gatewayGroup)
	}

	os.Exit(m.Run())
}

// runA7 executes the a7 binary with the given arguments and returns
// captured stdout, stderr, and any error.
func runA7(args ...string) (string, string, error) {
	cmd := exec.Command(binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// runA7WithEnv executes the a7 binary with custom environment variables.
func runA7WithEnv(env []string, args ...string) (string, string, error) {
	cmd := exec.Command(binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), env...)
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// adminAPI sends an HTTP request to the API7 EE Dashboard API.
// Used for test setup and cleanup — not for testing the CLI itself.
// Uses insecureClient because API7 EE typically uses self-signed certs.
func adminAPI(method, path string, body []byte) (*http.Response, error) {
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, adminURL+path, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, adminURL+path, nil)
	}
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", adminToken)
	req.Header.Set("Content-Type", "application/json")
	return insecureClient.Do(req)
}

// runtimeAdminAPI sends an HTTP request to the API7 EE runtime Admin API
// (APISIX admin endpoints). These endpoints require gateway_group_id as a
// query parameter.
func runtimeAdminAPI(method, path string, body []byte) (*http.Response, error) {
	var req *http.Request
	var err error
	url := adminURL + path
	// Append gateway_group_id query parameter.
	if strings.Contains(url, "?") {
		url += "&gateway_group_id=" + gatewayGroup
	} else {
		url += "?gateway_group_id=" + gatewayGroup
	}
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", adminToken)
	req.Header.Set("Content-Type", "application/json")
	return insecureClient.Do(req)
}

// waitForHealthy polls the given URL until it returns a successful response
// or the timeout is reached. Uses insecureClient for TLS.
func waitForHealthy(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		req.Header.Set("X-API-KEY", adminToken)
		resp, err := insecureClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode < 400 {
			return nil
		}
		lastErr = fmt.Errorf("unexpected status %d", resp.StatusCode)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for %s: %v", url, lastErr)
}

// setupEnv returns env vars and creates a context pointing at the real API7 EE instance.
// Each test gets an isolated config directory to avoid context conflicts.
func setupEnv(t *testing.T) []string {
	t.Helper()
	env := []string{
		"A7_CONFIG_DIR=" + t.TempDir(),
	}
	_, _, err := runA7WithEnv(env, "context", "create", "test",
		"--server", adminURL,
		"--token", adminToken,
		"--gateway-group", gatewayGroup,
		"--tls-skip-verify",
	)
	if err != nil {
		t.Fatalf("failed to create test context: %v", err)
	}
	return env
}

// envOrDefault returns the value of the environment variable named by key,
// or fallback if the variable is not set or empty.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func resolveModuleRoot() (string, error) {
	cmd := exec.Command("go", "env", "GOMOD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go env GOMOD: %w", err)
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		return "", fmt.Errorf("not inside a Go module")
	}
	return filepath.Dir(gomod), nil
}

func requireGatewayURL(t *testing.T) {
	t.Helper()
	if gatewayURL == "" {
		t.Skip("A7_GATEWAY_URL not set — skipping gateway traffic test")
	}
}

func requireHTTPBin(t *testing.T) {
	t.Helper()
	if httpbinURL == "" {
		t.Skip("HTTPBIN_URL not set — skipping httpbin-dependent test")
	}
}

// upstreamNode returns a valid upstream node address for test fixtures.
// When HTTPBIN_URL is set, it returns the host:port from that URL.
// Otherwise, it returns a safe dummy address so that routes can be created
// even when no real upstream is available.
func upstreamNode() string {
	if httpbinURL != "" {
		node := strings.TrimPrefix(httpbinURL, "http://")
		node = strings.TrimPrefix(node, "https://")
		return node
	}
	return "127.0.0.1:80"
}

// upstreamNodeHost returns the host portion (without port) for array-format nodes.
func upstreamNodeHost() string {
	node := upstreamNode()
	if idx := strings.LastIndex(node, ":"); idx > 0 {
		return node[:idx]
	}
	return node
}

// upstreamNodePort returns the port portion for array-format nodes.
func upstreamNodePort() int {
	node := upstreamNode()
	if idx := strings.LastIndex(node, ":"); idx > 0 {
		port := 80
		fmt.Sscanf(node[idx+1:], "%d", &port)
		return port
	}
	return 80
}

// createTestRouteWithServiceViaCLI creates a route that belongs to an existing service.
// The service must already exist. This is needed because API7 EE requires service_id for routes.
func createTestRouteWithServiceViaCLI(t *testing.T, env []string, routeID, serviceID string) {
	t.Helper()
	routeJSON := fmt.Sprintf(`{
		"id": %q,
		"name": "e2e-route-%s",
		"service_id": %q,
		"paths": ["/test-%s"],
		"upstream": {
			"type": "roundrobin",
			"nodes": [{"host": %q, "port": %d, "weight": 1}]
		}
	}`, routeID, routeID, serviceID, routeID, upstreamNodeHost(), upstreamNodePort())

	tmpFile := filepath.Join(t.TempDir(), "route.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(routeJSON), 0644))

	stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
}

func resolveFirstGatewayGroupID() (string, error) {
	req, err := http.NewRequest(http.MethodGet, adminURL+"/api/gateway_groups", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-API-KEY", adminToken)
	resp, err := insecureClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("GET /api/gateway_groups returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		List []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"list"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse gateway groups: %w (body: %s)", err, string(body))
	}
	if len(result.List) == 0 {
		return "", fmt.Errorf("no gateway groups found")
	}
	return result.List[0].ID, nil
}
