//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// runCase invokes the a7 binary for a single permCase and records the result
// to the package-level recorder. Setup cleanup runs in a defer, so the next
// case always starts from a clean slate even if this one explodes.
func runCase(tier int, baseEnv []string, c permCase) {
	args := c.Args
	if c.Setup != nil {
		cleanup, mutatedArgs, err := c.Setup(baseEnv)
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			recorder.Record(tier, c.Name, args, c.ExtraEnv, "", "", err, 0, c.ExpectFail,
				fmt.Sprintf("setup failed: %v", err))
			return
		}
		if mutatedArgs != nil {
			args = mutatedArgs
		}
	}

	env := append([]string(nil), baseEnv...)
	env = append(env, c.ExtraEnv...)

	start := time.Now()
	stdout, stderr, err := runA7WithEnv(env, args...)
	duration := time.Since(start)

	failureReason := evaluateCase(c, stdout, stderr, err)
	recorder.Record(tier, c.Name, args, c.ExtraEnv, stdout, stderr, err, duration, c.ExpectFail, failureReason)

	if failureReason != "" {
		AddReportEntry(fmt.Sprintf("tier %d FAIL", tier),
			fmt.Sprintf("case=%q reason=%q stderr=%s", c.Name, failureReason, truncate(stderr, 200)))
	}
}

// evaluateCase returns "" on success or a non-empty reason on failure.
// The order is: hard error overrides everything, then ExpectFail mismatch,
// then case-specific Validate.
func evaluateCase(c permCase, stdout, stderr string, exitErr error) string {
	if exitErr != nil && !c.ExpectFail {
		// Truly unexpected failure unless Validate explicitly handles it.
		if c.Validate != nil {
			if reason := c.Validate(stdout, stderr, exitErr); reason != "" {
				return reason
			}
			return ""
		}
		return fmt.Sprintf("unexpected non-zero exit: %v; stderr=%s", exitErr, truncate(stderr, 200))
	}
	if exitErr == nil && c.ExpectFail {
		return "expected non-zero exit but command succeeded"
	}
	if c.Validate != nil {
		return c.Validate(stdout, stderr, exitErr)
	}
	return ""
}

// failsInTier returns the number of recorded failures for a given tier.
func failsInTier(tier int) int {
	n := 0
	for _, r := range recorder.Snapshot() {
		if r.Tier == tier && !r.Passed {
			n++
		}
	}
	return n
}

// runCRUDWalker drives the full CRUD round-trip for one resource spec,
// recording each step as a separate tier-6 result. Cleanup is always
// attempted in defer regardless of intermediate failures.
func runCRUDWalker(baseEnv []string, spec resourceSpec) {
	id := uniqueResourceID("a7-perm-" + spec.name)
	parentID := ""

	if spec.parentSetup != nil {
		pid, pcleanup, err := spec.parentSetup()
		if err != nil {
			recorder.Record(6, spec.name+" parent setup", nil, nil, "", "", err, 0, false,
				fmt.Sprintf("parent setup failed: %v", err))
			return
		}
		defer pcleanup()
		parentID = pid
	}

	defer func() {
		if spec.cleanup != nil {
			_ = spec.cleanup(id, parentID)
		}
	}()

	gg := []string{}
	if spec.needsGatewayGroup {
		gg = []string{"-g", gatewayGroup}
	}

	// Step 1: create via file. Run directly (not via walkStep) so we can
	// inspect stderr for known capability-gap patterns. If the EE rejects
	// the resource type for environmental reasons (e.g. stream-route on an
	// HTTP-only deployment), the remaining steps would all cascade fail; we
	// downgrade those to a single informational "skipped" record instead.
	filePath, fileCleanup, err := resourceCleanupFile(spec, id, parentID)
	if err != nil {
		recorder.Record(6, fmt.Sprintf("%s create-via-file write", spec.name), nil, nil, "", "", err, 0, false,
			fmt.Sprintf("tempfile write failed: %v", err))
		return
	}
	defer fileCleanup()

	createArgs := append([]string{spec.name, "create", "-f", filePath}, gg...)
	createStart := time.Now()
	createStdout, createStderr, createErr := runA7WithEnv(baseEnv, createArgs...)
	if createErr != nil && isCapabilityGapStderr(createStderr) {
		recorder.Record(6, fmt.Sprintf("%s CRUD skipped (capability gap)", spec.name),
			createArgs, nil, createStdout, createStderr, nil, time.Since(createStart),
			false, "")
		return
	}
	failureReason := ""
	if createErr != nil {
		failureReason = fmt.Sprintf("unexpected non-zero exit: %v; stderr=%s", createErr, truncate(createStderr, 200))
	}
	recorder.Record(6, fmt.Sprintf("%s create -f", spec.name),
		createArgs, nil, createStdout, createStderr, createErr, time.Since(createStart),
		false, failureReason)
	if createErr != nil {
		// create failed for a reason other than a known capability gap; the
		// remaining steps would all fail with "resource not found". Stop here
		// rather than producing cascade noise.
		return
	}

	// Step 2: get table
	walkStep(baseEnv, 6, fmt.Sprintf("%s get default", spec.name),
		append([]string{spec.name, "get", id}, gg...), nil, false, func(stdout, _ string, err error) string {
			if err != nil {
				return fmt.Sprintf("get failed: %v", err)
			}
			if !strings.Contains(stdout, id) {
				return "get output missing id"
			}
			return ""
		})

	// Step 3: get json
	walkStep(baseEnv, 6, fmt.Sprintf("%s get -o json", spec.name),
		append([]string{spec.name, "get", id, "-o", "json"}, gg...), nil, false, validateOutputFormat("json"))

	// Step 4: get yaml
	walkStep(baseEnv, 6, fmt.Sprintf("%s get -o yaml", spec.name),
		append([]string{spec.name, "get", id, "-o", "yaml"}, gg...), nil, false, validateOutputFormat("yaml"))

	// Step 5: list (with extra args if required)
	listExtra := []string{}
	if spec.listArgs != nil {
		listExtra = spec.listArgs(parentID)
	}
	walkStep(baseEnv, 6, fmt.Sprintf("%s list", spec.name),
		append(append([]string{spec.name, "list"}, gg...), listExtra...), nil, false, func(stdout, _ string, err error) string {
			if err != nil {
				return fmt.Sprintf("list failed: %v", err)
			}
			if !strings.Contains(stdout, id) {
				return "list output did not include the just-created resource id"
			}
			return ""
		})

	// Step 6: export (skip resources without an export verb)
	if spec.hasExport {
		exportExtra := []string{}
		if spec.exportArgs != nil {
			exportExtra = spec.exportArgs(parentID)
		}
		walkStep(baseEnv, 6, fmt.Sprintf("%s export -o json", spec.name),
			append(append([]string{spec.name, "export"}, gg...), append(exportExtra, "-o", "json")...),
			nil, false, validateOutputFormat("json"))
	}

	// Step 7: delete without --force should decline when stdin echoes "n"
	// (we approximate by passing --force=false explicitly; the binary will
	// require a tty, so this case is allowed to fail-soft and is informational).
	// Skipping interactive-decline here to keep the matrix deterministic.

	// Step 8: delete with --force
	walkStep(baseEnv, 6, fmt.Sprintf("%s delete --force", spec.name),
		append([]string{spec.name, "delete", id, "--force"}, gg...), nil, false, nil)

	// Step 9: verify gone
	walkStep(baseEnv, 6, fmt.Sprintf("%s get after delete", spec.name),
		append([]string{spec.name, "get", id}, gg...), nil, true, nil)
}

// walkStep is a one-liner wrapper around runCase for the CRUD walker.
func walkStep(env []string, tier int, name string, args, extraEnv []string, expectFail bool, validate func(string, string, error) string) {
	runCase(tier, env, permCase{
		Name:       name,
		Args:       args,
		ExtraEnv:   extraEnv,
		ExpectFail: expectFail,
		Validate:   validate,
	})
}

// runContextLifecycle exercises tier 3. It runs in its own isolated config
// dir so it does not interfere with the rest of the suite.
func runContextLifecycle() {
	tmpDir, err := os.MkdirTemp("", "a7-perm-ctx-*")
	if err != nil {
		recorder.Record(3, "context isolation setup", nil, nil, "", "", err, 0, false,
			fmt.Sprintf("tempdir failed: %v", err))
		return
	}
	defer os.RemoveAll(tmpDir)
	env := []string{"A7_CONFIG_DIR=" + tmpDir}

	steps := []permCase{
		{
			Name: "context create perm-ctx-a",
			Args: []string{"context", "create", "perm-ctx-a",
				"--server", adminURL, "--token", adminToken,
				"--gateway-group", gatewayGroup,
				"--tls-skip-verify", "--skip-validation"},
		},
		{Name: "context list", Args: []string{"context", "list"},
			Validate: func(stdout, _ string, err error) string {
				if err != nil {
					return fmt.Sprintf("list failed: %v", err)
				}
				if !strings.Contains(stdout, "perm-ctx-a") {
					return "list output missing perm-ctx-a"
				}
				return ""
			}},
		{Name: "context current", Args: []string{"context", "current"}},
		{Name: "context create perm-ctx-b",
			Args: []string{"context", "create", "perm-ctx-b",
				"--server", adminURL, "--token", adminToken,
				"--gateway-group", gatewayGroup,
				"--tls-skip-verify", "--skip-validation"}},
		{Name: "context use perm-ctx-b", Args: []string{"context", "use", "perm-ctx-b"}},
		{Name: "context delete perm-ctx-a", Args: []string{"context", "delete", "perm-ctx-a"}},
		{Name: "context delete perm-ctx-b", Args: []string{"context", "delete", "perm-ctx-b"}},
		{Name: "context use missing", Args: []string{"context", "use", "no-such-context"}, ExpectFail: true},
	}
	for _, s := range steps {
		s.ExtraEnv = append(s.ExtraEnv, env...)
		runCase(3, nil, s)
	}
}

// runAuthPrecedence exercises tier 5: which source wins among flag, env, and
// context-file. We use `gateway-group list` as the read-only probe and a known
// wrong token in env or context to detect leakage. The base env from setupEnv
// already points at a context with tls-skip-verify, so we never pass that
// flag on read-only commands here (it's only valid for `context create`).
func runAuthPrecedence(baseEnv []string) {
	// 5.1: flag-only token overrides bad env token. Command must succeed.
	runCase(5, baseEnv, permCase{
		Name:     "flag --token overrides A7_TOKEN env",
		Args:     []string{"gateway-group", "list", "--token", adminToken, "--server", adminURL},
		ExtraEnv: []string{"A7_TOKEN=a7ee-known-bad-env-token"},
		Validate: func(stdout, _ string, err error) string {
			if err != nil {
				return fmt.Sprintf("expected flag to override env, got error: %v", err)
			}
			return ""
		},
	})

	// 5.2: env token works alongside the context file (same good value).
	runCase(5, baseEnv, permCase{
		Name:     "A7_TOKEN env alongside context-file token",
		Args:     []string{"gateway-group", "list"},
		ExtraEnv: []string{"A7_TOKEN=" + adminToken},
		Validate: func(_, _ string, err error) string {
			if err != nil {
				return fmt.Sprintf("env+context combination failed: %v", err)
			}
			return ""
		},
	})

	// 5.3: bad flag must beat good env -> command fails (negative direction).
	runCase(5, baseEnv, permCase{
		Name:       "bad --token overrides good env",
		Args:       []string{"gateway-group", "list", "--token", "a7ee-bad-flag"},
		ExtraEnv:   []string{"A7_TOKEN=" + adminToken},
		ExpectFail: true,
	})
}

// runGlobalRuleWorkflow drives global-rule CRUD by `update` (the upsert path)
// because `create` rejects any payload that includes an "id" field — the id
// is derived server-side from the plugin name. Records each step under tier 6
// so its results appear next to the generic walker's output.
func runGlobalRuleWorkflow(baseEnv []string) {
	pluginID := "response-rewrite"
	defer func() { _ = deleteGlobalRuleByID(pluginID) }()

	body := fmt.Sprintf(`{
  "plugins": {%q: {"headers": {"X-Perm": "1"}}}
}`, pluginID)
	path, cleanup, err := writeTempJSON(body)
	if err != nil {
		recorder.Record(6, "global-rule write tempfile", nil, nil, "", "", err, 0, false,
			fmt.Sprintf("tempfile: %v", err))
		return
	}
	defer cleanup()

	gg := []string{"-g", gatewayGroup}
	walkStep(baseEnv, 6, "global-rule update -f (upsert)",
		append([]string{"global-rule", "update", pluginID, "-f", path}, gg...),
		nil, false, nil)
	walkStep(baseEnv, 6, "global-rule get",
		append([]string{"global-rule", "get", pluginID}, gg...),
		nil, false, func(stdout, _ string, err error) string {
			if err != nil {
				return fmt.Sprintf("get failed: %v", err)
			}
			if !strings.Contains(stdout, pluginID) {
				return "get output missing id"
			}
			return ""
		})
	walkStep(baseEnv, 6, "global-rule list",
		append([]string{"global-rule", "list"}, gg...),
		nil, false, func(stdout, _ string, err error) string {
			if err != nil {
				return fmt.Sprintf("list failed: %v", err)
			}
			if !strings.Contains(stdout, pluginID) {
				return "list output missing id"
			}
			return ""
		})
	walkStep(baseEnv, 6, "global-rule export -o json",
		append([]string{"global-rule", "export", "-o", "json"}, gg...),
		nil, false, validateOutputFormat("json"))
	walkStep(baseEnv, 6, "global-rule delete --force",
		append([]string{"global-rule", "delete", pluginID, "--force"}, gg...),
		nil, false, nil)
	walkStep(baseEnv, 6, "global-rule get after delete",
		append([]string{"global-rule", "get", pluginID}, gg...),
		nil, true, nil)
}

// runSecretWorkflow probes the /api/secrets endpoint first; if it isn't
// exposed on this EE build the workflow records an informational "capability
// gap" outcome (PASS) and returns. Otherwise it drives a normal CRUD walk.
func runSecretWorkflow(baseEnv []string) {
	probe, err := adminAPI("GET", "/api/secrets?gateway_group_id="+gatewayGroup, nil)
	if err != nil {
		recorder.Record(6, "secret probe", nil, nil, "", "", err, 0, false,
			fmt.Sprintf("probe failed: %v", err))
		return
	}
	defer probe.Body.Close()
	if probe.StatusCode == 404 {
		recorder.Record(6, "secret CRUD skipped (capability gap: /api/secrets is 404)",
			nil, nil, "", "", nil, 0, false, "")
		return
	}
	// Endpoint exists — drive a minimal create+get+delete by hand. Stays out
	// of the generic walker because secret has no `export` verb.
	id := uniqueResourceID("a7-perm-secret")
	defer func() { _ = deleteSecretByID(id) }()
	body := fmt.Sprintf(`{
  "id": %q,
  "manager": "vault",
  "uri": "https://example.com",
  "prefix": "kv",
  "token": "dummy"
}`, id)
	path, cleanup, err := writeTempJSON(body)
	if err != nil {
		recorder.Record(6, "secret write tempfile", nil, nil, "", "", err, 0, false,
			fmt.Sprintf("tempfile: %v", err))
		return
	}
	defer cleanup()
	gg := []string{"-g", gatewayGroup}
	walkStep(baseEnv, 6, "secret create -f",
		append([]string{"secret", "create", "-f", path}, gg...), nil, false, nil)
	walkStep(baseEnv, 6, "secret get",
		append([]string{"secret", "get", id}, gg...), nil, false, nil)
	walkStep(baseEnv, 6, "secret delete --force",
		append([]string{"secret", "delete", id, "--force"}, gg...), nil, false, nil)
}

// runDeclarativeConfigWorkflow exercises tier 7: dump -> validate (clean).
// The full diff/sync round-trip is deferred until first-run results show the
// non-mutating dump+validate is stable here.
func runDeclarativeConfigWorkflow(baseEnv []string) {
	dumpFile := tempName("a7-perm-dump", "yaml")
	defer os.Remove(dumpFile)

	walkStep(baseEnv, 7, "config dump to file",
		[]string{"config", "dump", "-g", gatewayGroup, "-f", dumpFile, "-o", "yaml"},
		nil, false, func(_, _ string, err error) string {
			if err != nil {
				return fmt.Sprintf("dump failed: %v", err)
			}
			info, statErr := os.Stat(dumpFile)
			if statErr != nil {
				return fmt.Sprintf("dump file missing: %v", statErr)
			}
			if info.Size() == 0 {
				return "dump file empty"
			}
			return ""
		})

	walkStep(baseEnv, 7, "config validate dumped file",
		[]string{"config", "validate", "-f", dumpFile},
		nil, false, nil)

	walkStep(baseEnv, 7, "config diff against dumped file (clean expected)",
		[]string{"config", "diff", "-g", gatewayGroup, "-f", dumpFile},
		nil, false, nil)
}

// runDebugTier covers tier 9 with a single safe smoke. The trace path needs
// a live gateway and is intentionally skipped unless A7_GATEWAY_URL is set.
func runDebugTier(baseEnv []string) {
	walkStep(baseEnv, 9, "debug logs --help",
		[]string{"debug", "logs", "--help"}, nil, false, nil)

	if gatewayURL == "" {
		recorder.Record(9, "debug trace (skipped, no A7_GATEWAY_URL)", nil, nil, "", "", nil, 0, false, "")
		return
	}
	walkStep(baseEnv, 9, "debug trace --help",
		[]string{"debug", "trace", "--help"}, nil, false, nil)
}

func tempName(prefix, ext string) string {
	f, err := os.CreateTemp("", prefix+"-*."+ext)
	if err != nil {
		return ""
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return name
}

// The ginkgo container for the permutation suite. One Describe per top-level
// concern; each tier is one It so a tier failure does not short-circuit later
// tiers.
//
// Label("permutation") opts this suite out of the regular `make test-e2e`
// target. The default CI runs that target on every PR; the permutation matrix
// is much larger than the existing per-resource happy-path coverage and is
// intentionally manual via the dedicated `make test-e2e-permutation` target.
var _ = Describe("Permutation", Ordered, ContinueOnFailure, Label("permutation"), func() {
	var env []string

	BeforeAll(func() {
		env = setupEnv(GinkgoT())
	})

	AfterAll(func() {
		dir := artifactDir()
		if err := recorder.WriteReport(dir); err != nil {
			GinkgoWriter.Printf("permutation report write failed: %v\n", err)
		} else {
			GinkgoWriter.Printf("permutation report written to %s\n", dir)
		}
		if errs := permSweep(); len(errs) > 0 {
			for _, e := range errs {
				GinkgoWriter.Printf("permutation sweep: %v\n", e)
			}
		}
	})

	It("Tier 1: help integrity", func() {
		for _, c := range helpCases() {
			runCase(1, env, c)
		}
		Expect(failsInTier(1)).To(Equal(0), "see permutation-report.md")
	})

	It("Tier 2: version / completion", func() {
		for _, c := range versionCompletionCases() {
			runCase(2, env, c)
		}
		Expect(failsInTier(2)).To(Equal(0), "see permutation-report.md")
	})

	It("Tier 3: context lifecycle", func() {
		runContextLifecycle()
		Expect(failsInTier(3)).To(Equal(0), "see permutation-report.md")
	})

	It("Tier 4: output-format matrix", func() {
		for _, c := range outputFormatCases() {
			runCase(4, env, c)
		}
		Expect(failsInTier(4)).To(Equal(0), "see permutation-report.md")
	})

	It("Tier 5: auth-source precedence", func() {
		runAuthPrecedence(env)
		Expect(failsInTier(5)).To(Equal(0), "see permutation-report.md")
	})

	It("Tier 6: CRUD round-trip per resource", func() {
		for _, spec := range resourceSpecs() {
			runCRUDWalker(env, spec)
		}
		runGlobalRuleWorkflow(env)
		runSecretWorkflow(env)
		Expect(failsInTier(6)).To(Equal(0), "see permutation-report.md")
	})

	It("Tier 7: declarative config workflow", func() {
		runDeclarativeConfigWorkflow(env)
		Expect(failsInTier(7)).To(Equal(0), "see permutation-report.md")
	})

	It("Tier 8: negative / error matrix", func() {
		for _, c := range negativeCases() {
			runCase(8, env, c)
		}
		Expect(failsInTier(8)).To(Equal(0), "see permutation-report.md")
	})

	It("Tier 9: debug commands smoke", func() {
		runDebugTier(env)
		Expect(failsInTier(9)).To(Equal(0), "see permutation-report.md")
	})

	It("Tier 10: unsupported commands", func() {
		for _, c := range unsupportedCases() {
			runCase(10, env, c)
		}
		Expect(failsInTier(10)).To(Equal(0), "see permutation-report.md")
	})
})
