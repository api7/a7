//go:build e2e

package e2e

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// permResult is one row of the permutation report.
type permResult struct {
	Tier            int      `json:"tier"`
	TierName        string   `json:"tier_name"`
	Name            string   `json:"name"`
	Args            []string `json:"args"`
	EnvOverrides    []string `json:"env_overrides,omitempty"`
	ExitCode        int      `json:"exit_code"`
	ExitErr         string   `json:"exit_err,omitempty"`
	StdoutSHA256    string   `json:"stdout_sha256"`
	StdoutBytes     int      `json:"stdout_bytes"`
	StderrFirst500  string   `json:"stderr_first_500,omitempty"`
	StderrBytes     int      `json:"stderr_bytes"`
	DurationMillis  int64    `json:"duration_ms"`
	Passed          bool     `json:"passed"`
	FailureReason   string   `json:"failure_reason,omitempty"`
	ExpectedFailure bool     `json:"expected_failure,omitempty"`
}

// permRecorder accumulates results across the suite. Safe for concurrent use.
type permRecorder struct {
	mu      sync.Mutex
	results []permResult
}

var recorder = &permRecorder{}

// Record appends a result. Tier name resolution is centralised so the JSON and
// the rendered markdown stay in sync.
func (r *permRecorder) Record(tier int, name string, args, envOverrides []string,
	stdout, stderr string, exitErr error,
	duration time.Duration,
	expectedFailure bool, failureReason string,
) {
	res := permResult{
		Tier:            tier,
		TierName:        tierName(tier),
		Name:            name,
		Args:            append([]string(nil), args...),
		EnvOverrides:    append([]string(nil), envOverrides...),
		StdoutSHA256:    sha256hex(stdout),
		StdoutBytes:     len(stdout),
		StderrBytes:     len(stderr),
		StderrFirst500:  truncate(stderr, 500),
		DurationMillis:  duration.Milliseconds(),
		ExpectedFailure: expectedFailure,
		FailureReason:   failureReason,
	}
	if exitErr != nil {
		res.ExitErr = exitErr.Error()
		res.ExitCode = extractExitCode(exitErr)
	}
	res.Passed = failureReason == ""

	r.mu.Lock()
	r.results = append(r.results, res)
	r.mu.Unlock()
}

// Snapshot returns a copy of the current results, sorted by tier then name.
// Caller may mutate the returned slice freely.
func (r *permRecorder) Snapshot() []permResult {
	r.mu.Lock()
	out := make([]permResult, len(r.results))
	copy(out, r.results)
	r.mu.Unlock()
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Tier != out[j].Tier {
			return out[i].Tier < out[j].Tier
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// WriteReport writes both the JSON and Markdown report files. Directory is
// created if missing. Returns an error if either write fails; callers in
// AfterSuite should log but not fail the suite on report-write failure.
func (r *permRecorder) WriteReport(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create artifact dir: %w", err)
	}
	rows := r.Snapshot()

	jsonPath := filepath.Join(dir, "permutation-report.json")
	jsonBytes, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report json: %w", err)
	}
	if err := os.WriteFile(jsonPath, jsonBytes, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", jsonPath, err)
	}

	mdPath := filepath.Join(dir, "permutation-report.md")
	if err := os.WriteFile(mdPath, []byte(renderMarkdown(rows)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", mdPath, err)
	}
	return nil
}

// renderMarkdown builds a paste-able summary grouped by tier with a failures
// section appended at the end.
func renderMarkdown(rows []permResult) string {
	var b strings.Builder
	b.WriteString("# a7 CLI Permutation Report\n\n")
	b.WriteString(fmt.Sprintf("Generated at: %s\n\n", time.Now().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Target: `%s` (gateway-group resolved id `%s`)\n\n", adminURL, gatewayGroup))

	totalPass, totalFail := 0, 0
	tierBuckets := map[int][]permResult{}
	tierOrder := []int{}
	for _, r := range rows {
		if _, ok := tierBuckets[r.Tier]; !ok {
			tierOrder = append(tierOrder, r.Tier)
		}
		tierBuckets[r.Tier] = append(tierBuckets[r.Tier], r)
		if r.Passed {
			totalPass++
		} else {
			totalFail++
		}
	}
	sort.Ints(tierOrder)

	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("- Total cases: **%d**\n", totalPass+totalFail))
	b.WriteString(fmt.Sprintf("- Passed: **%d**\n", totalPass))
	b.WriteString(fmt.Sprintf("- Failed: **%d**\n\n", totalFail))

	b.WriteString("| Tier | Name | Passed | Failed |\n|---|---|---:|---:|\n")
	for _, t := range tierOrder {
		p, f := 0, 0
		for _, r := range tierBuckets[t] {
			if r.Passed {
				p++
			} else {
				f++
			}
		}
		b.WriteString(fmt.Sprintf("| %d | %s | %d | %d |\n", t, tierName(t), p, f))
	}
	b.WriteString("\n")

	for _, t := range tierOrder {
		b.WriteString(fmt.Sprintf("## Tier %d: %s\n\n", t, tierName(t)))
		b.WriteString("| Status | Case | Duration | Args |\n|---|---|---:|---|\n")
		for _, r := range tierBuckets[t] {
			status := "PASS"
			if !r.Passed {
				status = "FAIL"
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %dms | `%s` |\n",
				status, escapePipes(r.Name), r.DurationMillis, escapePipes(strings.Join(r.Args, " "))))
		}
		b.WriteString("\n")
	}

	if totalFail > 0 {
		b.WriteString("## Failures\n\n")
		for _, r := range rows {
			if r.Passed {
				continue
			}
			b.WriteString(fmt.Sprintf("### Tier %d - %s\n\n", r.Tier, r.Name))
			b.WriteString(fmt.Sprintf("- Reason: %s\n", r.FailureReason))
			b.WriteString(fmt.Sprintf("- Exit code: %d (`%s`)\n", r.ExitCode, r.ExitErr))
			b.WriteString(fmt.Sprintf("- Command: `a7 %s`\n", strings.Join(r.Args, " ")))
			if len(r.EnvOverrides) > 0 {
				b.WriteString(fmt.Sprintf("- Env overrides: `%s`\n", strings.Join(r.EnvOverrides, " ")))
			}
			if r.StderrFirst500 != "" {
				b.WriteString("- Stderr (first 500 bytes):\n\n")
				b.WriteString("```\n")
				b.WriteString(r.StderrFirst500)
				b.WriteString("\n```\n")
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func tierName(t int) string {
	switch t {
	case 1:
		return "Help integrity"
	case 2:
		return "Version / completion / update help"
	case 3:
		return "Context lifecycle"
	case 4:
		return "Output-format matrix"
	case 5:
		return "Auth-source precedence"
	case 6:
		return "CRUD round-trip per resource"
	case 7:
		return "Declarative config workflow"
	case 8:
		return "Negative / error matrix"
	case 9:
		return "Debug commands"
	case 10:
		return "Unsupported commands"
	default:
		return "Unknown"
	}
}

func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

func escapePipes(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}

// extractExitCode best-effort returns the process exit code from an *exec.ExitError
// wrapped by exec.CommandContext. Non-exit failures (timeout, spawn failure)
// return -1.
func extractExitCode(err error) int {
	if err == nil {
		return 0
	}
	// Use a string contains check to avoid importing exec just for the type
	// assertion, since exec is already in setup_test.go and this file shares
	// the package. Prefer direct unwrap when available.
	type exitCoder interface{ ExitCode() int }
	if ec, ok := err.(exitCoder); ok {
		return ec.ExitCode()
	}
	return -1
}
