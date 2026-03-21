//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Completion Tests ---

func TestCompletion_Bash(t *testing.T) {
	stdout, stderr, err := runA7("completion", "bash")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
	// Bash completion scripts contain function definitions.
	assert.True(t, strings.Contains(stdout, "bash") || strings.Contains(stdout, "complete") || strings.Contains(stdout, "__"),
		"bash completion should contain shell-specific content")
}

func TestCompletion_Zsh(t *testing.T) {
	stdout, stderr, err := runA7("completion", "zsh")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
	// Zsh completion scripts typically contain compdef or _a7.
	assert.True(t, strings.Contains(stdout, "zsh") || strings.Contains(stdout, "compdef") || strings.Contains(stdout, "_a7"),
		"zsh completion should contain shell-specific content")
}

func TestCompletion_Fish(t *testing.T) {
	stdout, stderr, err := runA7("completion", "fish")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
	// Fish completion scripts contain "complete" commands.
	assert.True(t, strings.Contains(stdout, "complete") || strings.Contains(stdout, "fish"),
		"fish completion should contain shell-specific content")
}

func TestCompletion_Powershell(t *testing.T) {
	stdout, stderr, err := runA7("completion", "powershell")
	require.NoError(t, err, stderr)
	assert.NotEmpty(t, stdout)
}

func TestCompletion_InvalidShell(t *testing.T) {
	_, _, err := runA7("completion", "invalid-shell")
	assert.Error(t, err)
}

func TestCompletion_NoArgs(t *testing.T) {
	// completion without a shell arg should error (ExactArgs(1)).
	_, _, err := runA7("completion")
	assert.Error(t, err)
}

// --- Version Tests ---

func TestVersion(t *testing.T) {
	stdout, stderr, err := runA7("version")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "a7")
	assert.Contains(t, stdout, "version")
}

func TestVersion_ContainsCommitInfo(t *testing.T) {
	stdout, stderr, err := runA7("version")
	require.NoError(t, err, stderr)
	// The version output format is: "a7 version <ver> (commit: <hash>, built: <date>)"
	assert.Contains(t, stdout, "commit:")
	assert.Contains(t, stdout, "built:")
}

// --- Help Tests ---

func TestHelp(t *testing.T) {
	stdout, stderr, err := runA7("--help")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "a7")
	// Help should list available commands.
	assert.Contains(t, stdout, "route")
	assert.Contains(t, stdout, "service")
	assert.Contains(t, stdout, "upstream")
}

func TestHelp_SubcommandRoute(t *testing.T) {
	stdout, stderr, err := runA7("route", "--help")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "list")
	assert.Contains(t, stdout, "get")
	assert.Contains(t, stdout, "create")
	assert.Contains(t, stdout, "delete")
}

func TestHelp_SubcommandConfig(t *testing.T) {
	stdout, stderr, err := runA7("config", "--help")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "dump")
	assert.Contains(t, stdout, "diff")
	assert.Contains(t, stdout, "sync")
	assert.Contains(t, stdout, "validate")
}

func TestHelp_SubcommandDebug(t *testing.T) {
	stdout, stderr, err := runA7("debug", "--help")
	require.NoError(t, err, stderr)
	assert.Contains(t, stdout, "trace")
	assert.Contains(t, stdout, "logs")
}

func TestUnknownCommand(t *testing.T) {
	_, _, err := runA7("nonexistent-command")
	assert.Error(t, err)
}
