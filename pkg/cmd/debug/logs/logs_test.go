package logs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogs_DetectContainerParsing(t *testing.T) {
	parsed := parseDockerPSNames("api7ee\napi7-dashboard\n\n")
	require.Len(t, parsed, 2)
	assert.Equal(t, "api7ee", parsed[0])
	assert.Equal(t, "api7-dashboard", parsed[1])
}

func TestLogs_BuildDockerArgs(t *testing.T) {
	opts := &Options{Tail: 100}
	args := buildDockerArgs(opts, "api7ee")
	assert.Equal(t, []string{"logs", "--tail", "100", "api7ee"}, args)
}

func TestLogs_BuildDockerArgsWithFollow(t *testing.T) {
	opts := &Options{Follow: true, Tail: 10}
	args := buildDockerArgs(opts, "api7ee")
	assert.Equal(t, []string{"logs", "--follow", "--tail", "10", "api7ee"}, args)
}

func TestLogs_BuildDockerArgsWithSince(t *testing.T) {
	opts := &Options{Tail: 5, Since: "1h"}
	args := buildDockerArgs(opts, "api7ee")
	assert.Equal(t, []string{"logs", "--tail", "5", "--since", "1h", "api7ee"}, args)
}

func TestLogs_ReadLastLines(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "api7ee.log")
	content := "line-1\nline-2\nline-3\nline-4\n"
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	lines, err := readLastLines(filePath, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"line-3", "line-4"}, lines)
}

func TestLogs_NoContainerNoFile(t *testing.T) {
	_, err := chooseContainer(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no API7 EE container found")
}
