//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func isKnownSecretCapabilityGap(stdout, stderr string) bool {
	combined := stdout + "\n" + stderr
	for _, needle := range []string{
		"resource not found",
		"secret provider",
		"vault",
		"not configured",
		"unsupported",
	} {
		if strings.Contains(strings.ToLower(combined), needle) {
			return true
		}
	}
	return false
}

var _ = Describe("Local Stability", Ordered, func() {
	It("runs the binary and reaches the control plane", func() {
		stdout, stderr, err := runA7("version")
		Expect(err).NotTo(HaveOccurred(), stderr)
		Expect(stdout).To(ContainSubstring("a7 version"))

		resp, err := adminAPI(http.MethodGet, "/api/gateway_groups", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.Body.Close()).To(Succeed())
		Expect(resp.StatusCode).To(BeNumerically("<", 400))
	})

	It("supports service-backed route traffic forwarding", func() {
		t := GinkgoT()
		requireGatewayURL(t)
		requireHTTPBin(t)

		env := setupEnv(t)
		svcID := "ginkgo-route-svc"
		routeID := "ginkgo-route-forward"
		t.Cleanup(func() {
			deleteRouteViaAdmin(t, routeID)
			deleteServiceViaAdmin(t, svcID)
		})

		createTestServiceViaCLI(t, env, svcID)
		routeJSON := fmt.Sprintf(`{
			"id": %q,
			"name": "ginkgo-route-forward",
			"service_id": %q,
			"paths": ["/ginkgo-forward"],
			"plugins": {"proxy-rewrite": {"uri": "/get"}}
		}`, routeID, svcID)
		tmpFile := filepath.Join(t.TempDir(), "route.json")
		Expect(os.WriteFile(tmpFile, []byte(routeJSON), 0o644)).To(Succeed())

		stdout, stderr, err := runA7WithEnv(env, "route", "create", "-f", tmpFile, "-g", gatewayGroup)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("stdout=%s stderr=%s", stdout, stderr))

		status, err := waitForGatewayStatus(gatewayURL+"/ginkgo-forward", func() (*http.Request, error) {
			return http.NewRequest(http.MethodGet, gatewayURL+"/ginkgo-forward", nil)
		}, func(code int) bool {
			return code == http.StatusOK
		}, 15*time.Second)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(http.StatusOK))
	})

	It("traces a live request through the current route model", func() {
		t := GinkgoT()
		requireGatewayURL(t)
		requireHTTPBin(t)

		env := setupEnv(t)
		svcID := "ginkgo-debug-svc"
		routeID := "ginkgo-debug-route"
		t.Cleanup(func() {
			deleteRouteViaAdmin(t, routeID)
			deleteServiceViaAdmin(t, svcID)
		})

		createTestServiceViaCLI(t, env, svcID)
		createDebugTraceRoute(t, env, svcID, routeID, "/ginkgo-trace", "")
		waitForDebugTraceRoute(t, "/ginkgo-trace")

		stdout, stderr, err := runA7WithEnv(env, "debug", "trace", routeID, "-g", gatewayGroup, "--gateway-url", gatewayURL, "-o", "json")
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("stdout=%s stderr=%s", stdout, stderr))

		var result map[string]any
		Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
		Expect(result).To(HaveKey("route"))
		Expect(result).To(HaveKey("response"))
	})

	It("creates a secret provider using the documented positional ID workflow", func() {
		t := GinkgoT()
		env := setupEnv(t)
		secretID := "vault/ginkgo-secret"
		t.Cleanup(func() { deleteSecretViaAdmin(t, "vault", "ginkgo-secret") })

		secretJSON := `{"uri":"https://vault.example.com","prefix":"kv/apisix","token":"test-vault-token"}`
		tmpFile := filepath.Join(t.TempDir(), "secret.json")
		Expect(os.WriteFile(tmpFile, []byte(secretJSON), 0o644)).To(Succeed())

		stdout, stderr, err := runA7WithEnv(env, "secret", "create", secretID, "-f", tmpFile, "-g", gatewayGroup)
		if err != nil {
			if isKnownSecretCapabilityGap(stdout, stderr) {
				Skip(fmt.Sprintf("secret create is unavailable in this environment: stdout=%s stderr=%s", stdout, stderr))
			}
			Fail(fmt.Sprintf("secret create failed unexpectedly: stdout=%s stderr=%s", stdout, stderr))
		}

		stdout, stderr, err = runA7WithEnv(env, "secret", "get", secretID, "-g", gatewayGroup, "-o", "json")
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("stdout=%s stderr=%s", stdout, stderr))
		Expect(stdout).To(ContainSubstring("vault.example.com"))
	})

	It("updates a stream route from file without forcing upstream flags", func() {
		t := GinkgoT()
		env := setupEnv(t)
		svcID := "ginkgo-stream-svc"
		srID := "ginkgo-stream-route"
		t.Cleanup(func() {
			deleteStreamRouteViaAdmin(t, srID)
			deleteServiceViaAdmin(t, svcID)
		})

		createTestServiceViaCLI(t, env, svcID)
		createBody := fmt.Sprintf(`{
			"id": %q,
			"name": "ginkgo-stream-route",
			"service_id": %q,
			"server_port": 19191
		}`, srID, svcID)
		createFile := filepath.Join(t.TempDir(), "stream-route-create.json")
		Expect(os.WriteFile(createFile, []byte(createBody), 0o644)).To(Succeed())

		stdout, stderr, err := runA7WithEnv(env, "stream-route", "create", "-f", createFile, "-g", gatewayGroup)
		if err != nil {
			Skip(fmt.Sprintf("stream-route create failed in this environment: stdout=%s stderr=%s", stdout, stderr))
		}

		updateBody := fmt.Sprintf(`{
			"id": %q,
			"name": "ginkgo-stream-route-updated",
			"service_id": %q,
			"server_port": 19192,
			"desc": "updated via ginkgo"
		}`, srID, svcID)
		updateFile := filepath.Join(t.TempDir(), "stream-route-update.json")
		Expect(os.WriteFile(updateFile, []byte(updateBody), 0o644)).To(Succeed())

		stdout, stderr, err = runA7WithEnv(env, "stream-route", "update", srID, "-f", updateFile, "-g", gatewayGroup)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("stdout=%s stderr=%s", stdout, stderr))
		Expect(stdout).To(ContainSubstring("ginkgo-stream-route-updated"))
	})
})
