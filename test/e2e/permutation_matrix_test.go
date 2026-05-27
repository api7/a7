//go:build e2e

package e2e

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// permCase is the unit of work driven by ginkgo Entry rows. Each case maps
// to exactly one a7 invocation. Cases that need a fixture (e.g. a parent
// service for a route) declare Setup; the cleanup func is deferred by the
// runner regardless of pass/fail.
type permCase struct {
	Name string
	Args []string

	// ExtraEnv is appended after the base env returned by setupEnv. Used by
	// the auth-precedence tier and any case that wants to override A7_TOKEN
	// or similar.
	ExtraEnv []string

	// Setup runs before the a7 invocation. It returns a cleanup func, a
	// possibly-mutated args slice (e.g. with a parent id substituted), and an
	// error. The runner calls cleanup() in a defer even if the case fails.
	Setup func(env []string) (cleanup func(), args []string, err error)

	// ExpectFail flags cases that are expected to exit non-zero (negative
	// matrix, unsupported commands). The runner inverts the pass condition.
	ExpectFail bool

	// Validate inspects the captured output and returns a non-empty failure
	// reason if the case did not behave as expected. nil means default
	// "exit code matches ExpectFail" check is sufficient.
	Validate func(stdout, stderr string, exitErr error) string
}

// ---------- tier 1: help integrity ----------

// helpCases returns one case per (command, --help|-h) pair. The leaf list is
// hardcoded so we get a stable contract rather than re-discovering the surface
// each run. New commands must be added here when added to the CLI.
func helpCases() []permCase {
	leaves := allLeafCommands()
	cases := make([]permCase, 0, len(leaves)*2)
	for _, leaf := range leaves {
		for _, flag := range []string{"--help", "-h"} {
			args := append([]string{}, leaf...)
			args = append(args, flag)
			cases = append(cases, permCase{
				Name: "help " + strings.Join(leaf, " ") + " " + flag,
				Args: args,
				Validate: func(stdout, stderr string, exitErr error) string {
					if exitErr != nil {
						return fmt.Sprintf("--help should exit 0, got error: %v", exitErr)
					}
					if !strings.Contains(stdout, "Usage:") && !strings.Contains(stdout, "a7") {
						return "help output missing Usage/a7 marker"
					}
					return ""
				},
			})
		}
	}
	return cases
}

// allLeafCommands returns every leaf command and parent command we want help
// to render for. Each entry is the argv to a7 minus the binary itself.
func allLeafCommands() [][]string {
	return [][]string{
		// root
		{},
		// version / completion / update / context
		{"version"},
		{"completion"},
		{"update"},
		{"context"}, {"context", "create"}, {"context", "use"}, {"context", "list"}, {"context", "current"}, {"context", "delete"},
		// gateway-group
		{"gateway-group"}, {"gateway-group", "list"}, {"gateway-group", "get"},
		{"gateway-group", "create"}, {"gateway-group", "update"}, {"gateway-group", "delete"},
		// route
		{"route"}, {"route", "list"}, {"route", "get"}, {"route", "create"},
		{"route", "update"}, {"route", "delete"}, {"route", "export"},
		// service
		{"service"}, {"service", "list"}, {"service", "get"}, {"service", "create"},
		{"service", "update"}, {"service", "delete"}, {"service", "export"},
		// consumer
		{"consumer"}, {"consumer", "list"}, {"consumer", "get"}, {"consumer", "create"},
		{"consumer", "update"}, {"consumer", "delete"}, {"consumer", "export"},
		// credential
		{"credential"}, {"credential", "list"}, {"credential", "get"}, {"credential", "create"},
		{"credential", "update"}, {"credential", "delete"},
		// ssl
		{"ssl"}, {"ssl", "list"}, {"ssl", "get"}, {"ssl", "create"},
		{"ssl", "update"}, {"ssl", "delete"}, {"ssl", "export"},
		// secret
		{"secret"}, {"secret", "list"}, {"secret", "get"}, {"secret", "create"},
		{"secret", "update"}, {"secret", "delete"},
		// global-rule
		{"global-rule"}, {"global-rule", "list"}, {"global-rule", "get"}, {"global-rule", "create"},
		{"global-rule", "update"}, {"global-rule", "delete"}, {"global-rule", "export"},
		// stream-route
		{"stream-route"}, {"stream-route", "list"}, {"stream-route", "get"}, {"stream-route", "create"},
		{"stream-route", "update"}, {"stream-route", "delete"}, {"stream-route", "export"},
		// plugin (read-only)
		{"plugin"}, {"plugin", "list"}, {"plugin", "get"},
		// plugin-metadata
		{"plugin-metadata"}, {"plugin-metadata", "get"}, {"plugin-metadata", "create"},
		{"plugin-metadata", "update"}, {"plugin-metadata", "delete"},
		// proto
		{"proto"}, {"proto", "list"}, {"proto", "get"}, {"proto", "create"},
		{"proto", "update"}, {"proto", "delete"}, {"proto", "export"},
		// config
		{"config"}, {"config", "dump"}, {"config", "diff"}, {"config", "sync"}, {"config", "validate"},
		// debug
		{"debug"}, {"debug", "logs"}, {"debug", "trace"},
	}
}

// ---------- tier 2: version / completion / misc ----------

func versionCompletionCases() []permCase {
	cases := []permCase{
		{
			Name: "version prints version string",
			Args: []string{"version"},
			Validate: func(stdout, _ string, err error) string {
				if err != nil {
					return fmt.Sprintf("version exited non-zero: %v", err)
				}
				if !strings.Contains(stdout, "a7") {
					return "version output missing 'a7' substring"
				}
				return ""
			},
		},
		{
			Name: "version -o json",
			Args: []string{"version", "-o", "json"},
			// Best-effort: many version commands don't honor -o json; we accept
			// either valid JSON or graceful pass-through.
		},
	}
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		shell := shell
		cases = append(cases, permCase{
			Name: "completion " + shell,
			Args: []string{"completion", shell},
			Validate: func(stdout, _ string, err error) string {
				if err != nil {
					return fmt.Sprintf("completion %s exited non-zero: %v", shell, err)
				}
				if len(stdout) < 50 {
					return fmt.Sprintf("completion %s produced suspiciously short output (%d bytes)", shell, len(stdout))
				}
				return ""
			},
		})
	}
	cases = append(cases, permCase{
		Name: "completion unsupported shell",
		Args: []string{"completion", "csh"},
		// cobra rejects with non-zero exit and a "valid args" hint.
		ExpectFail: true,
	})
	return cases
}

// ---------- tier 4: output-format matrix ----------

// outputFormatCases targets the read-only verbs (list, get-known-id, export)
// across every resource that exposes them, exercising each output format plus
// an invalid value that must be rejected.
//
// "Known ids" come from a fixture that the runner seeds in BeforeAll (one
// service `a7-perm-fmt-service`). For commands that don't need an id we just
// list/export.
func outputFormatCases() []permCase {
	formats := []string{"table", "json", "yaml"}
	cases := []permCase{}

	// gateway-group list works without any prerequisite resource.
	for _, f := range formats {
		f := f
		cases = append(cases, permCase{
			Name: "gateway-group list -o " + f,
			Args: []string{"gateway-group", "list", "-o", f},
			Validate: validateOutputFormat(f),
		})
	}
	cases = append(cases, permCase{
		Name:       "gateway-group list -o invalid",
		Args:       []string{"gateway-group", "list", "-o", "totally-not-a-format"},
		ExpectFail: true,
	})

	// plugin list works without any prerequisite resource (read-only catalog).
	for _, f := range formats {
		f := f
		cases = append(cases, permCase{
			Name: "plugin list -o " + f,
			Args: []string{"plugin", "list", "-g", gatewayGroup, "-o", f},
			Validate: validateOutputFormat(f),
		})
	}

	// service list with each format.
	for _, f := range formats {
		f := f
		cases = append(cases, permCase{
			Name: "service list -o " + f,
			Args: []string{"service", "list", "-g", gatewayGroup, "-o", f},
			Validate: validateOutputFormat(f),
		})
	}

	// config validate against a tiny known-good file (covers the no-network path).
	for _, f := range formats {
		f := f
		cases = append(cases, permCase{
			Name: "config validate (good file) -o " + f,
			Args: []string{"config", "validate", "-f", "__SUBSTITUTED__"},
			Setup: func(env []string) (func(), []string, error) {
				path, cleanup, err := writeTempYAML("version: \"1\"\nservices: []\n")
				if err != nil {
					return nil, nil, err
				}
				return cleanup, []string{"config", "validate", "-f", path, "-o", f}, nil
			},
			Validate: validateOutputFormat(f),
		})
	}

	return cases
}

func validateOutputFormat(format string) func(stdout, stderr string, err error) string {
	return func(stdout, _ string, err error) string {
		if err != nil {
			return fmt.Sprintf("exited non-zero with -o %s: %v", format, err)
		}
		switch format {
		case "json":
			trimmed := strings.TrimSpace(stdout)
			if trimmed == "" {
				return "empty stdout for -o json"
			}
			var anything interface{}
			if err := json.Unmarshal([]byte(trimmed), &anything); err != nil {
				return fmt.Sprintf("stdout is not valid JSON: %v", err)
			}
		case "yaml":
			if strings.TrimSpace(stdout) == "" {
				return "empty stdout for -o yaml"
			}
		case "table":
			if strings.TrimSpace(stdout) == "" {
				return "empty stdout for -o table"
			}
		}
		return ""
	}
}

// ---------- tier 8: negative / error matrix ----------

func negativeCases() []permCase {
	return []permCase{
		{
			Name:       "service get missing id",
			Args:       []string{"service", "get", "-g", gatewayGroup},
			ExpectFail: true,
		},
		{
			Name:       "service get nonexistent id",
			Args:       []string{"service", "get", "a7-perm-no-such-svc-zzzzz", "-g", gatewayGroup},
			ExpectFail: true,
		},
		{
			Name:       "service delete nonexistent id",
			Args:       []string{"service", "delete", "a7-perm-no-such-svc-zzzzz", "--force", "-g", gatewayGroup},
			ExpectFail: true,
		},
		{
			Name:       "route get missing id",
			Args:       []string{"route", "get", "-g", gatewayGroup},
			ExpectFail: true,
		},
		{
			Name:       "route create without service-id and without file",
			Args:       []string{"route", "create", "-g", gatewayGroup, "--name", "no-svc"},
			ExpectFail: true,
		},
		{
			Name:       "consumer create missing username and file",
			Args:       []string{"consumer", "create", "-g", gatewayGroup},
			ExpectFail: true,
		},
		{
			Name: "config validate against malformed file",
			Args: []string{"config", "validate", "-f", "__SUBSTITUTED__"},
			Setup: func(env []string) (func(), []string, error) {
				path, cleanup, err := writeTempYAML("this is: : not valid yaml\n  : -")
				if err != nil {
					return nil, nil, err
				}
				return cleanup, []string{"config", "validate", "-f", path}, nil
			},
			ExpectFail: true,
		},
		{
			Name: "config validate rejects unsupported top-level upstreams section",
			Args: []string{"config", "validate", "-f", "__SUBSTITUTED__"},
			Setup: func(env []string) (func(), []string, error) {
				path, cleanup, err := writeTempYAML("version: \"1\"\nupstreams:\n  - id: u1\n    nodes:\n      - host: 127.0.0.1\n        port: 80\n        weight: 1\n")
				if err != nil {
					return nil, nil, err
				}
				return cleanup, []string{"config", "validate", "-f", path}, nil
			},
			ExpectFail: true,
		},
		{
			Name: "config validate rejects unsupported consumer_groups section",
			Args: []string{"config", "validate", "-f", "__SUBSTITUTED__"},
			Setup: func(env []string) (func(), []string, error) {
				path, cleanup, err := writeTempYAML("version: \"1\"\nconsumer_groups:\n  - id: cg1\n")
				if err != nil {
					return nil, nil, err
				}
				return cleanup, []string{"config", "validate", "-f", path}, nil
			},
			ExpectFail: true,
		},
		{
			Name:       "gateway-group list with wrong token",
			Args:       []string{"gateway-group", "list", "--token", "a7ee-not-a-real-token-aaa"},
			ExpectFail: true,
		},
		{
			Name:       "gateway-group list against wrong server",
			Args:       []string{"gateway-group", "list", "--server", "https://127.0.0.1:1"},
			ExpectFail: true,
		},
	}
}

// ---------- tier 10: unsupported commands ----------

func unsupportedCases() []permCase {
	return []permCase{
		{Name: "unsupported upstream", Args: []string{"upstream"}, ExpectFail: true,
			Validate: containsAny([]string{"unknown command", "unknown subcommand"})},
		{Name: "unsupported consumer-group", Args: []string{"consumer-group"}, ExpectFail: true,
			Validate: containsAny([]string{"unknown command", "unknown subcommand"})},
		{Name: "unsupported service-template", Args: []string{"service-template"}, ExpectFail: true,
			Validate: containsAny([]string{"unknown command", "unknown subcommand"})},
		{Name: "unsupported plugin-config", Args: []string{"plugin-config"}, ExpectFail: true,
			Validate: containsAny([]string{"unknown command", "unknown subcommand"})},
	}
}

func containsAny(needles []string) func(stdout, stderr string, err error) string {
	return func(stdout, stderr string, err error) string {
		combined := strings.ToLower(stdout + "\n" + stderr)
		for _, n := range needles {
			if strings.Contains(combined, strings.ToLower(n)) {
				return ""
			}
		}
		return fmt.Sprintf("stderr did not contain any of %v; got: %s", needles, truncate(stderr, 200))
	}
}

// ---------- tier 6: per-resource CRUD walker ----------

// resourceSpec describes one resource type for the CRUD walker. Each spec is
// independent — the walker creates everything it needs from scratch and tears
// it down at the end.
type resourceSpec struct {
	// name is the CLI subcommand (e.g. "service", "route").
	name string

	// needsGatewayGroup is true for runtime resources; false for context.
	needsGatewayGroup bool

	// parentSetup optionally provisions a parent resource (e.g. a service for
	// a route) via direct admin API. Returns parent id, cleanup, error.
	parentSetup func() (parentID string, cleanup func(), err error)

	// fileBody builds the JSON body for `create -f <file>`. The id is the
	// caller-chosen unique id; parentID is whatever parentSetup returned.
	fileBody func(id, parentID string) string

	// listArgs returns extra args needed for list to succeed (e.g. routes
	// require --service-id under access-token auth). Empty for resources that
	// can list unscoped.
	listArgs func(parentID string) []string

	// exportArgs returns extra args needed for export. Empty if export is not
	// applicable or works unscoped.
	exportArgs func(parentID string) []string

	// hasExport is false for resources whose CLI lacks an export verb (e.g.
	// secret, plugin-metadata, credential, gateway-group).
	hasExport bool

	// idArgPosition controls whether the id is passed as a positional arg
	// (most resources) or via a flag (none currently — placeholder).
	// Always positional for current commands.

	// cleanup is the resource-specific delete that runs in defer.
	cleanup func(id, parentID string) error
}

// resourceSpecs returns the list of specs covered by tier 6. New resources
// added to the CLI should be added here.
func resourceSpecs() []resourceSpec {
	return []resourceSpec{
		{
			name:              "service",
			needsGatewayGroup: true,
			fileBody: func(id, _ string) string {
				return fmt.Sprintf(`{
  "id": %q,
  "name": "perm-svc",
  "upstream": {
    "type": "roundrobin",
    "nodes": [{"host": "127.0.0.1", "port": 80, "weight": 1}]
  }
}`, id)
			},
			hasExport:  true,
			exportArgs: func(_ string) []string { return nil },
			cleanup:    func(id, _ string) error { return deleteServiceByID(id) },
		},
		{
			name:              "route",
			needsGatewayGroup: true,
			parentSetup:       provisionParentService,
			fileBody: func(id, parentID string) string {
				return fmt.Sprintf(`{
  "id": %q,
  "name": "perm-route",
  "service_id": %q,
  "paths": ["/perm-%s"]
}`, id, parentID, id)
			},
			listArgs:   func(parentID string) []string { return []string{"--service-id", parentID} },
			exportArgs: func(parentID string) []string { return []string{"--service-id", parentID} },
			hasExport:  true,
			cleanup:    func(id, _ string) error { return deleteRouteByID(id) },
		},
		{
			name:              "consumer",
			needsGatewayGroup: true,
			fileBody: func(id, _ string) string {
				return fmt.Sprintf(`{
  "username": %q,
  "desc": "permutation consumer"
}`, id)
			},
			hasExport: true,
			cleanup:   func(id, _ string) error { return deleteConsumerByID(id) },
		},
		{
			name:              "ssl",
			needsGatewayGroup: true,
			fileBody: func(id, _ string) string {
				cert, key, err := generatePermCert(id + ".perm.example.com")
				if err != nil {
					// Return a payload that will fail with a clear marker
					// rather than a malformed JSON.
					return fmt.Sprintf(`{"id": %q, "_cert_gen_error": %q}`, id, err.Error())
				}
				return fmt.Sprintf(`{
  "id": %q,
  "cert": %s,
  "key": %s,
  "snis": ["%s.perm.example.com"]
}`, id, jsonEscape(cert), jsonEscape(key), id)
			},
			hasExport: true,
			cleanup:   func(id, _ string) error { return deleteSSLByID(id) },
		},
		// global-rule is intentionally NOT in this list. Its CLI rejects "id"
		// in create payload (the id is derived from the plugin name), so the
		// generic walker can't drive it. See runGlobalRuleWorkflow.
		{
			name:              "stream-route",
			needsGatewayGroup: true,
			parentSetup:       provisionParentService,
			fileBody: func(id, parentID string) string {
				return fmt.Sprintf(`{
  "id": %q,
  "name": "perm-stream-route",
  "service_id": %q,
  "server_port": 9100
}`, id, parentID)
			},
			listArgs:   func(parentID string) []string { return []string{"--service-id", parentID} },
			exportArgs: func(parentID string) []string { return []string{"--service-id", parentID} },
			hasExport:  true,
			cleanup:    func(id, _ string) error { return deleteStreamRouteByID(id) },
		},
		// secret is intentionally NOT in this list. The /api/secrets admin
		// endpoint is not exposed on every API7 EE build (404 from the
		// dashboard frontend); driving it from the generic walker produces a
		// noisy cascade. See runSecretWorkflow for an isolated check that
		// records a clean "capability gap" outcome.
		{
			name:              "proto",
			needsGatewayGroup: true,
			fileBody: func(id, _ string) string {
				return fmt.Sprintf(`{
  "id": %q,
  "content": "syntax = \"proto3\"; package perm; message M { string s = 1; }"
}`, id)
			},
			hasExport: true,
			cleanup:   func(id, _ string) error { return deleteProtoByID(id) },
		},
	}
}

// provisionParentService creates a parent service via direct admin API so
// downstream route / stream-route cases can attach. Returns the service id
// and a cleanup that removes it.
func provisionParentService() (string, func(), error) {
	id := uniqueResourceID("a7-perm-svc-parent")
	body := fmt.Sprintf(`{
  "id": %q,
  "name": "perm-parent-svc",
  "upstream": {"type": "roundrobin", "nodes": [{"host": "127.0.0.1", "port": 80, "weight": 1}]}
}`, id)
	resp, err := runtimeAdminAPI("PUT", "/apisix/admin/services/"+id, []byte(body))
	if err != nil {
		return "", nil, fmt.Errorf("create parent service: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", nil, fmt.Errorf("create parent service: status %d", resp.StatusCode)
	}
	return id, func() { _ = deleteServiceByID(id) }, nil
}

// writeTempYAML writes the content to a temp file and returns its path plus
// a cleanup func. The temp file lives in the system tempdir and is removed
// by the cleanup.
func writeTempYAML(content string) (string, func(), error) {
	f, err := os.CreateTemp("", "a7-perm-*.yaml")
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		os.Remove(f.Name())
		return "", nil, err
	}
	return f.Name(), func() { os.Remove(f.Name()) }, nil
}

// writeTempJSON writes JSON content and returns its path and cleanup.
func writeTempJSON(content string) (string, func(), error) {
	f, err := os.CreateTemp("", "a7-perm-*.json")
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		os.Remove(f.Name())
		return "", nil, err
	}
	return f.Name(), func() { os.Remove(f.Name()) }, nil
}

// resourceCleanupFile makes a tempfile from the resource spec for create/update.
func resourceCleanupFile(spec resourceSpec, id, parentID string) (string, func(), error) {
	return writeTempJSON(spec.fileBody(id, parentID))
}

// resourceCRUDPath returns the absolute filepath of the artifact dir under
// the module root.
func artifactDir() string {
	wd, _ := os.Getwd()
	// test/e2e is the working dir under ginkgo; resolve to ./_artifacts inside.
	return filepath.Join(wd, "_artifacts")
}

// generatePermCert builds a self-signed ECDSA P-256 cert/key pair for the
// given SNI. Generated fresh on each call so the SSL CRUD case is independent
// of any embedded fixture that might drift out of date.
func generatePermCert(sni string) (certPEM, keyPEM string, err error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("ecdsa generate: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", fmt.Errorf("serial: %w", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   sni,
			Organization: []string{"a7-permutation"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{sni},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return "", "", fmt.Errorf("create cert: %w", err)
	}
	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return "", "", fmt.Errorf("marshal key: %w", err)
	}
	keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
	return certPEM, keyPEM, nil
}

// jsonEscape returns a JSON-string-literal form of s (including surrounding
// quotes). Used to safely embed multi-line PEM blobs into the JSON request
// bodies the CRUD walker writes to disk.
func jsonEscape(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

// isCapabilityGapStderr returns true when the create-step stderr looks like a
// known environmental gap on the target EE rather than a bug in the CLI or
// the test fixture. Used by the CRUD walker to downgrade a cascade of follow-
// up failures into a single informational "skipped" record.
//
// Known patterns:
//   - "can not create a Stream Route to the HTTP Service" — stream routes on
//     an EE deployment that only exposes HTTP services.
//   - "resource not found" returned from a create call — usually means the
//     resource type's admin endpoint is not exposed at all (e.g. secret
//     provider on some builds).
//   - secret-provider / vault gaps already covered by isKnownSecretCapability
//     Gap in local_stability_ginkgo_test.go; cover the same shape here so
//     both suites stay in agreement.
func isCapabilityGapStderr(stderr string) bool {
	low := strings.ToLower(stderr)
	for _, needle := range []string{
		"can not create a stream route",
		"can not create a stream",
		"stream route to the http service",
		"secret provider",
		"vault",
		"not configured",
		"not enabled",
		"not supported",
	} {
		if strings.Contains(low, needle) {
			return true
		}
	}
	return false
}
