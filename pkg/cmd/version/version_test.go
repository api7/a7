package version

import (
	"net/http"
	"testing"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/internal/version"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/iostreams"
)

func TestVersionOutput(t *testing.T) {
	// Set version variables to known values
	originalVersion := version.Version
	originalCommit := version.Commit
	originalDate := version.Date
	defer func() {
		version.Version = originalVersion
		version.Commit = originalCommit
		version.Date = originalDate
	}()

	version.Version = "v0.1.0"
	version.Commit = "abc1234"
	version.Date = "2024-01-15"

	// Create test IOStreams
	ios, _, out, _ := iostreams.Test()

	// Create Factory with test IOStreams and stub functions
	factory := &cmd.Factory{
		IOStreams: ios,
		HttpClient: func() (*http.Client, error) {
			return &http.Client{}, nil
		},
		Config: func() (config.Config, error) {
			return nil, nil
		},
	}

	// Execute the version command
	cmd := NewCmd(factory)
	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	// Verify output
	output := out.String()
	expected := "a7 version v0.1.0 (commit: abc1234, built: 2024-01-15)\n"
	if output != expected {
		t.Errorf("output mismatch\nexpected: %q\ngot:      %q", expected, output)
	}
}
